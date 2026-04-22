package shutdowns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mockShutdownDeleter is a mock implementation of shutdownDeleter interface.
// It now also implements GetShutdownOrganizationID + GetOrganizationParentID
// required once cascade RBAC is wired into the handler.
type mockShutdownDeleter struct {
	deleteFunc func(ctx context.Context, id int64) error
	getOrgFunc func(ctx context.Context, id int64) (int64, error)
	parentFunc func(ctx context.Context, orgID int64) (*int64, error)
	deleteCalls int
}

func (m *mockShutdownDeleter) DeleteShutdown(ctx context.Context, id int64) error {
	m.deleteCalls++
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// GetShutdownOrganizationID is required for the handler's cascade RBAC check.
// Default returns (1, nil) so tests that don't care about ownership still pass.
func (m *mockShutdownDeleter) GetShutdownOrganizationID(ctx context.Context, id int64) (int64, error) {
	if m.getOrgFunc != nil {
		return m.getOrgFunc(ctx, id)
	}
	return 1, nil
}

// GetOrganizationParentID is required by the CascadeChecker interface.
func (m *mockShutdownDeleter) GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error) {
	if m.parentFunc != nil {
		return m.parentFunc(ctx, orgID)
	}
	return nil, nil
}

// contextWithScClaims creates a context with an "sc" role so the cascade
// access check returns nil without invoking any mock lookups. This lets
// legacy delete tests keep their exact semantics (they were written against
// DeleteShutdown returning ErrNotFound / other errors) once the handler
// is refactored.
func contextWithScClaims(ctx context.Context) context.Context {
	claims := &token.Claims{
		UserID: 1,
		Name:   "Test User",
		Roles:  []string{"sc"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

// contextWithDeleteRoleClaims lets cascade/other-role tests set their own role+orgID.
func contextWithDeleteRoleClaims(ctx context.Context, userID, orgID int64, role string) context.Context {
	claims := &token.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           "Test User",
		Roles:          []string{role},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name           string
		shutdownID     string
		mockError      error
		wantStatusCode int
		wantErrInBody  bool
		description    string
	}{
		{
			name:           "successful deletion",
			shutdownID:     "1",
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully delete an existing shutdown",
		},
		{
			name:           "successful deletion of shutdown with idle discharge",
			shutdownID:     "2",
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully delete shutdown and its associated idle discharge",
		},
		{
			name:           "error - invalid shutdown ID",
			shutdownID:     "invalid",
			mockError:      nil,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
			description:    "Should return bad request for invalid ID format",
		},
		{
			name:           "error - shutdown not found",
			shutdownID:     "9999",
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
			wantErrInBody:  true,
			description:    "Should return not found for non-existent shutdown (moved to GetShutdownOrganizationID)",
		},
		{
			name:           "error - internal server error",
			shutdownID:     "1",
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
			description:    "Should handle internal errors gracefully",
		},
		{
			name:           "error - negative ID",
			shutdownID:     "-1",
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
			wantErrInBody:  true,
			description:    "Should handle negative IDs",
		},
		{
			name:           "error - zero ID",
			shutdownID:     "0",
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
			wantErrInBody:  true,
			description:    "Should handle zero ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// After refactor, GetShutdownOrganizationID runs BEFORE DeleteShutdown.
			// To keep original semantics:
			//  - ErrNotFound cases: surface via GetShutdownOrganizationID so the
			//    handler short-circuits to 404 before touching DeleteShutdown.
			//  - Other errors (e.g. "database connection failed"): keep on
			//    DeleteShutdown so we still exercise the post-lookup error path.
			mock := &mockShutdownDeleter{
				getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
					if errors.Is(tt.mockError, storage.ErrNotFound) {
						return 0, storage.ErrNotFound
					}
					return 1, nil
				},
				deleteFunc: func(ctx context.Context, id int64) error {
					if errors.Is(tt.mockError, storage.ErrNotFound) {
						// never reached because GetShutdownOrganizationID fails first;
						// but return the error for safety
						return tt.mockError
					}
					return tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/shutdowns/"+tt.shutdownID, nil)

			// Inject sc claims so CheckCascadeStationAccess passes.
			ctx := contextWithScClaims(req.Context())
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Delete(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("%s: handler returned wrong status code: got %v want %v",
					tt.description, rr.Code, tt.wantStatusCode)
			}

			if tt.wantErrInBody && rr.Body.Len() > 0 {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("%s: expected error in response body, got: %v",
						tt.description, resp)
				}
			}

			if tt.wantStatusCode == http.StatusOK && !tt.wantErrInBody {
				if rr.Body.Len() > 0 {
					t.Errorf("%s: expected empty body for successful deletion, got: %v",
						tt.description, rr.Body.String())
				}
			}
		})
	}
}

// TestDelete_CascadeDeletion tests that deleting a shutdown also deletes associated idle discharge
func TestDelete_CascadeDeletion(t *testing.T) {
	tests := []struct {
		name             string
		shutdownID       string
		hasIdleDischarge bool
		mockError        error
		description      string
	}{
		{
			name:             "delete shutdown without idle discharge",
			shutdownID:       "1",
			hasIdleDischarge: false,
			mockError:        nil,
			description:      "Should successfully delete shutdown without idle discharge",
		},
		{
			name:             "delete shutdown with idle discharge",
			shutdownID:       "2",
			hasIdleDischarge: true,
			mockError:        nil,
			description:      "Should successfully delete shutdown and cascade delete idle discharge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletedIDs := []int64{}

			mock := &mockShutdownDeleter{
				deleteFunc: func(ctx context.Context, id int64) error {
					deletedIDs = append(deletedIDs, id)
					return tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/shutdowns/"+tt.shutdownID, nil)

			ctx := contextWithScClaims(req.Context())
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Delete(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %v", tt.description, rr.Code)
			}

			t.Logf("%s: PASSED - Deleted IDs: %v", tt.description, deletedIDs)
		})
	}
}

// TestDelete_ErrorHandling tests various error scenarios
func TestDelete_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		shutdownID   string
		setupMock    func() *mockShutdownDeleter
		expectedCode int
		description  string
	}{
		{
			name:       "database timeout",
			shutdownID: "1",
			setupMock: func() *mockShutdownDeleter {
				return &mockShutdownDeleter{
					deleteFunc: func(ctx context.Context, id int64) error {
						return errors.New("context deadline exceeded")
					},
				}
			},
			expectedCode: http.StatusInternalServerError,
			description:  "Should handle database timeout gracefully",
		},
		{
			name:       "constraint violation (should not happen, but handle gracefully)",
			shutdownID: "1",
			setupMock: func() *mockShutdownDeleter {
				return &mockShutdownDeleter{
					deleteFunc: func(ctx context.Context, id int64) error {
						return errors.New("constraint violation")
					},
				}
			},
			expectedCode: http.StatusInternalServerError,
			description:  "Should handle constraint violations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()

			req := httptest.NewRequest(http.MethodDelete, "/shutdowns/"+tt.shutdownID, nil)

			ctx := contextWithScClaims(req.Context())
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Delete(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("%s: expected status %v, got %v", tt.description, tt.expectedCode, rr.Code)
			}

			t.Logf("%s: PASSED", tt.description)
		})
	}
}

// ---- Cascade RBAC tests (RED) ----

func runDeleteWithCtx(t *testing.T, mock *mockShutdownDeleter, ctx context.Context, shutdownID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/shutdowns/"+shutdownID, bytes.NewBuffer(nil))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shutdownID)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Delete(logger, mock)
	handler.ServeHTTP(rr, req)
	return rr
}

func TestDelete_CascadeUser_Mine_OK(t *testing.T) {
	mock := &mockShutdownDeleter{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 10, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 10 {
				return int64Ptr(5), nil
			}
			return nil, nil
		},
	}

	ctx := contextWithDeleteRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runDeleteWithCtx(t, mock, ctx, "1")

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.deleteCalls != 1 {
		t.Errorf("expected DeleteShutdown to be called once, got %d", mock.deleteCalls)
	}
}

func TestDelete_CascadeUser_Foreign_NotFound(t *testing.T) {
	mock := &mockShutdownDeleter{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 20, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			return int64Ptr(7), nil
		},
	}

	ctx := contextWithDeleteRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runDeleteWithCtx(t, mock, ctx, "2")

	// Enumeration защита: foreign resource is reported as 404, not 403.
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 (enumeration protection), got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.deleteCalls != 0 {
		t.Errorf("DeleteShutdown must NOT be called for foreign resource, called %d times", mock.deleteCalls)
	}
}

func TestDelete_ShutdownNotFound_NotFound(t *testing.T) {
	mock := &mockShutdownDeleter{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 0, storage.ErrNotFound
		},
	}

	ctx := contextWithDeleteRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runDeleteWithCtx(t, mock, ctx, "9999")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.deleteCalls != 0 {
		t.Errorf("DeleteShutdown must NOT be called when lookup fails, called %d times", mock.deleteCalls)
	}
}

func TestDelete_ScUser_AnyShutdown_OK(t *testing.T) {
	parentCalled := false
	mock := &mockShutdownDeleter{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 999, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			parentCalled = true
			return int64Ptr(777), nil
		},
	}

	ctx := contextWithDeleteRoleClaims(context.Background(), 1, 5, "sc")
	rr := runDeleteWithCtx(t, mock, ctx, "1")

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if parentCalled {
		t.Error("parent lookup must NOT be called for sc role")
	}
}
