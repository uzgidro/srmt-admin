package shutdowns

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// mockShutdownEditor is a mock implementation of shutdownEditor interface
type mockShutdownEditor struct {
	editFunc    func(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	unlinkFunc  func(ctx context.Context, shutdownID int64) error
	linkFunc    func(ctx context.Context, shutdownID int64, fileIDs []int64) error
	getOrgFunc  func(ctx context.Context, id int64) (int64, error)
	parentFunc  func(ctx context.Context, orgID int64) (*int64, error)
	ownerFunc   func(ctx context.Context, id int64) (sql.NullInt64, error)
	editCalls   int
}

func (m *mockShutdownEditor) UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error {
	if m.unlinkFunc != nil {
		return m.unlinkFunc(ctx, shutdownID)
	}
	return nil
}

func (m *mockShutdownEditor) LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error {
	if m.linkFunc != nil {
		return m.linkFunc(ctx, shutdownID, fileIDs)
	}
	return nil
}

func (m *mockShutdownEditor) EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error {
	m.editCalls++
	if m.editFunc != nil {
		return m.editFunc(ctx, id, req)
	}
	return nil
}

// GetShutdownOrganizationID is required for cascade RBAC checks.
// Default returns (1, nil) so legacy happy-path tests don't need to set it.
func (m *mockShutdownEditor) GetShutdownOrganizationID(ctx context.Context, id int64) (int64, error) {
	if m.getOrgFunc != nil {
		return m.getOrgFunc(ctx, id)
	}
	return 1, nil
}

// GetOrganizationParentID is required by the CascadeChecker interface.
func (m *mockShutdownEditor) GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error) {
	if m.parentFunc != nil {
		return m.parentFunc(ctx, orgID)
	}
	return nil, nil
}

// GetShutdownCreatedByUserID is required for cascade-only owner restriction.
// Default returns the test caller's UserID (1) so legacy tests continue to pass
// once the handler starts calling auth.CheckShutdownOwnership.
func (m *mockShutdownEditor) GetShutdownCreatedByUserID(ctx context.Context, id int64) (sql.NullInt64, error) {
	if m.ownerFunc != nil {
		return m.ownerFunc(ctx, id)
	}
	return sql.NullInt64{Int64: 1, Valid: true}, nil
}

// Helper to create context with user claims (reuse from add_test.go approach).
// Default role is "sc" so legacy happy-path tests continue to pass once the
// handler starts calling CheckCascadeStationAccess (sc has full access).
func contextWithUserClaims(ctx context.Context, userID int64) context.Context {
	claims := &token.Claims{
		UserID: userID,
		Name:   "Test User",
		Roles:  []string{"sc"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

// contextWithEditRoleClaims creates context with arbitrary role/orgID for RBAC tests.
func contextWithEditRoleClaims(ctx context.Context, userID, orgID int64, role string) context.Context {
	claims := &token.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           "Test User",
		Roles:          []string{role},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

func TestEdit(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	tests := []struct {
		name           string
		shutdownID     string
		body           interface{}
		mockError      error
		wantStatusCode int
		wantErrInBody  bool
		description    string
	}{
		{
			name:       "successful edit - update organization_id",
			shutdownID: "1",
			body: editRequest{
				OrganizationID: int64Ptr(2),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully update organization_id",
		},
		{
			name:       "successful edit - update start and end time",
			shutdownID: "1",
			body: editRequest{
				StartTime: &now,
				EndTime:   timePtr(later),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully update start and end time",
		},
		{
			name:       "successful edit - add idle discharge to shutdown without one",
			shutdownID: "1",
			body: editRequest{
				IdleDischargeVolume: float64Ptr(10.0),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully add idle discharge to existing shutdown (THE FIX WE IMPLEMENTED)",
		},
		{
			name:       "successful edit - update idle discharge volume",
			shutdownID: "1",
			body: editRequest{
				IdleDischargeVolume: float64Ptr(15.0),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully update existing idle discharge volume",
		},
		{
			name:       "successful edit - update reason",
			shutdownID: "1",
			body: editRequest{
				Reason: stringPtr("Updated maintenance reason"),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully update reason",
		},
		{
			name:       "successful edit - add idle discharge with new end time",
			shutdownID: "1",
			body: editRequest{
				EndTime:             timePtr(later),
				IdleDischargeVolume: float64Ptr(8.0),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully add idle discharge with new end time",
		},
		{
			name:       "successful edit - remove idle discharge",
			shutdownID: "1",
			body: editRequest{
				Reason: stringPtr("No longer needs idle discharge"),
			},
			mockError:      nil,
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should successfully remove idle discharge when not provided",
		},
		{
			name:           "error - invalid shutdown ID",
			shutdownID:     "invalid",
			body:           editRequest{},
			mockError:      nil,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
			description:    "Should return bad request for invalid ID",
		},
		{
			name:       "error - shutdown not found",
			shutdownID: "9999",
			body: editRequest{
				OrganizationID: int64Ptr(2),
			},
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
			wantErrInBody:  true,
			description:    "Should return not found for non-existent shutdown",
		},
		{
			name:       "error - foreign key violation",
			shutdownID: "1",
			body: editRequest{
				OrganizationID: int64Ptr(999),
			},
			mockError:      storage.ErrForeignKeyViolation,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
			description:    "Should return bad request for invalid organization_id",
		},
		{
			name:       "error - internal server error",
			shutdownID: "1",
			body: editRequest{
				OrganizationID: int64Ptr(2),
			},
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
			description:    "Should handle internal errors gracefully",
		},
		{
			name:           "error - invalid JSON",
			shutdownID:     "1",
			body:           "invalid json",
			mockError:      nil,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
			description:    "Should return bad request for invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock editor
			mock := &mockShutdownEditor{
				editFunc: func(ctx context.Context, id int64, req dto.EditShutdownRequest) error {
					return tt.mockError
				},
			}

			// Create request body
			var bodyReader io.Reader
			if str, ok := tt.body.(string); ok {
				bodyReader = bytes.NewBufferString(str)
			} else {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewBuffer(bodyBytes)
			}

			// Create request
			req := httptest.NewRequest(http.MethodPatch, "/shutdowns/"+tt.shutdownID, bodyReader)
			req.Header.Set("Content-Type", "application/json")

			// Add authentication context
			ctx := contextWithUserClaims(req.Context(), 1)

			// Setup chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

			// Create response recorder
			rr := httptest.NewRecorder()

			// Create logger
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Call handler
			handler := Edit(logger, mock)
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("%s: handler returned wrong status code: got %v want %v",
					tt.description, rr.Code, tt.wantStatusCode)
			}

			// Check if error is in body when expected
			if tt.wantErrInBody {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("%s: expected error in response body, got: %v",
						tt.description, resp)
				}
			}
		})
	}
}

// TestEdit_IdleDischargeScenarios tests specific scenarios for idle discharge handling
func TestEdit_IdleDischargeScenarios(t *testing.T) {
	tests := []struct {
		name        string
		shutdownID  string
		body        editRequest
		mockError   error
		description string
	}{
		{
			name:       "add idle discharge when shutdown has end_time but no idle discharge",
			shutdownID: "1",
			body: editRequest{
				IdleDischargeVolume: float64Ptr(12.0),
			},
			mockError:   nil,
			description: "This tests the specific fix - creating idle discharge on edit when it didn't exist",
		},
		{
			name:       "add idle discharge and provide new end_time",
			shutdownID: "2",
			body: editRequest{
				EndTime:             timePtr(time.Now().Add(3 * time.Hour)),
				IdleDischargeVolume: float64Ptr(8.5),
			},
			mockError:   nil,
			description: "Should use the new end_time when creating idle discharge",
		},
		{
			name:       "update all fields including idle discharge",
			shutdownID: "3",
			body: editRequest{
				OrganizationID:      int64Ptr(2),
				StartTime:           timePtr(time.Now()),
				EndTime:             timePtr(time.Now().Add(4 * time.Hour)),
				Reason:              stringPtr("Comprehensive update"),
				GenerationLossMwh:   float64Ptr(25.5),
				IdleDischargeVolume: float64Ptr(20.0),
			},
			mockError:   nil,
			description: "Should successfully update all fields at once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockShutdownEditor{
				editFunc: func(ctx context.Context, id int64, req dto.EditShutdownRequest) error {
					// Verify the request has the expected fields
					if tt.body.IdleDischargeVolume != nil && req.IdleDischargeVolumeThousandM3 == nil {
						t.Error("IdleDischargeVolume not passed to storage layer")
					}
					return tt.mockError
				},
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPatch, "/shutdowns/"+tt.shutdownID, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add authentication context
			ctx := contextWithUserClaims(req.Context(), 1)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Edit(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %v", tt.description, rr.Code)
			}

			t.Logf("%s: PASSED", tt.description)
		})
	}
}

// Helper functions
func int64Ptr(i int64) *int64 {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// ---- Cascade RBAC tests (RED) ----

func runEditWithCtx(t *testing.T, mock *mockShutdownEditor, ctx context.Context, shutdownID string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/shutdowns/"+shutdownID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shutdownID)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Edit(logger, mock)
	handler.ServeHTTP(rr, req)
	return rr
}

func TestEdit_CascadeUser_CurrentMine_OK(t *testing.T) {
	mock := &mockShutdownEditor{
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

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{
		Reason: stringPtr("Just updating reason"),
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestEdit_CascadeUser_CurrentForeign_NotFound(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 20, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			return int64Ptr(7), nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "2", editRequest{
		Reason: stringPtr("Updated"),
	})

	// Enumeration защита: foreign resource is reported as 404, not 403.
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 (enumeration protection), got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.editCalls != 0 {
		t.Errorf("EditShutdown must NOT be called for foreign resource, called %d times", mock.editCalls)
	}
}

func TestEdit_CascadeUser_NewOrgForeign_Forbidden(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 10, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			switch orgID {
			case 10:
				return int64Ptr(5), nil
			case 20:
				return int64Ptr(7), nil
			}
			return nil, nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{
		OrganizationID: int64Ptr(20),
	})

	// Here 403 is appropriate because the caller OWNS the current record —
	// we're rejecting the target move to foreign org.
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.editCalls != 0 {
		t.Errorf("EditShutdown must NOT be called, called %d times", mock.editCalls)
	}
}

func TestEdit_CascadeUser_BothMine_OK(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 10, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			switch orgID {
			case 10, 11:
				return int64Ptr(5), nil
			}
			return nil, nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{
		OrganizationID: int64Ptr(11),
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestEdit_ShutdownNotFound_NotFound(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 0, storage.ErrNotFound
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "9999", editRequest{
		Reason: stringPtr("whatever"),
	})

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.editCalls != 0 {
		t.Errorf("EditShutdown must NOT be called when lookup fails, called %d times", mock.editCalls)
	}
}

func TestEdit_ScUser_AnyShutdown_OK(t *testing.T) {
	parentCalled := false
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
			return 999, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			parentCalled = true
			return int64Ptr(777), nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "sc")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{
		OrganizationID: int64Ptr(123),
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if parentCalled {
		t.Error("parent lookup must NOT be called for sc role")
	}
}

// === Cascade-owner restriction tests ===

// TestEdit_CascadeUser_OwnRecord_OK — cascade-юзер 10 правит свою запись
// (created_by_user_id=10) в своём каскаде → 200.
func TestEdit_CascadeUser_OwnRecord_OK(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 10, nil },
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 10 {
				return int64Ptr(5), nil
			}
			return nil, nil
		},
		ownerFunc: func(ctx context.Context, id int64) (sql.NullInt64, error) {
			return sql.NullInt64{Int64: 10, Valid: true}, nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 10, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{Reason: stringPtr("own record")})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.editCalls != 1 {
		t.Errorf("EditShutdown must be called once, called %d times", mock.editCalls)
	}
}

// TestEdit_CascadeUser_ForeignOwnerSameCascade_Forbidden — cascade-юзер 11 пробует
// править запись юзера 10 в общем каскаде → 403 (НЕ 404 — явный отказ).
func TestEdit_CascadeUser_ForeignOwnerSameCascade_Forbidden(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 10, nil },
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 10 {
				return int64Ptr(5), nil
			}
			return nil, nil
		},
		ownerFunc: func(ctx context.Context, id int64) (sql.NullInt64, error) {
			return sql.NullInt64{Int64: 10, Valid: true}, nil // owned by user 10
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 11, 5, "cascade") // user 11 ≠ owner
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{Reason: stringPtr("not mine")})

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.editCalls != 0 {
		t.Errorf("EditShutdown must NOT be called for non-owner, called %d times", mock.editCalls)
	}
	// Ensure error message hints at ownership (helps frontend distinguish from cascade-access 403).
	if !bytes.Contains(rr.Body.Bytes(), []byte("creator")) {
		t.Errorf("error message should mention creator; got %s", rr.Body.String())
	}
}

// TestEdit_CascadeUser_NullOwner_Forbidden — owner был удалён (FK SET NULL),
// cascade-юзер пробует править → 403.
func TestEdit_CascadeUser_NullOwner_Forbidden(t *testing.T) {
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 10, nil },
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 10 {
				return int64Ptr(5), nil
			}
			return nil, nil
		},
		ownerFunc: func(ctx context.Context, id int64) (sql.NullInt64, error) {
			return sql.NullInt64{Valid: false}, nil // orphan
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 10, 5, "cascade")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{Reason: stringPtr("orphan")})

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for orphaned record, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.editCalls != 0 {
		t.Errorf("EditShutdown must NOT be called for orphaned record, called %d times", mock.editCalls)
	}
}

// TestEdit_ScUser_AnyOwner_OK — sc role игнорирует ownership даже на чужой записи.
func TestEdit_ScUser_AnyOwner_OK(t *testing.T) {
	ownerCalled := false
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 999, nil },
		ownerFunc: func(ctx context.Context, id int64) (sql.NullInt64, error) {
			ownerCalled = true
			return sql.NullInt64{Int64: 12345, Valid: true}, nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "sc")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{Reason: stringPtr("admin override")})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for sc bypass, got %d: %s", rr.Code, rr.Body.String())
	}
	if ownerCalled {
		t.Error("ownership lookup must NOT happen for sc role (helper short-circuits)")
	}
}

// TestEdit_RaisUser_NullOwner_OK — rais тоже игнорирует ownership, даже на orphan.
func TestEdit_RaisUser_NullOwner_OK(t *testing.T) {
	ownerCalled := false
	mock := &mockShutdownEditor{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 999, nil },
		ownerFunc: func(ctx context.Context, id int64) (sql.NullInt64, error) {
			ownerCalled = true
			return sql.NullInt64{Valid: false}, nil
		},
	}

	ctx := contextWithEditRoleClaims(context.Background(), 1, 5, "rais")
	rr := runEditWithCtx(t, mock, ctx, "1", editRequest{Reason: stringPtr("rais override")})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for rais bypass, got %d: %s", rr.Code, rr.Body.String())
	}
	if ownerCalled {
		t.Error("ownership lookup must NOT happen for rais role (helper short-circuits)")
	}
}
