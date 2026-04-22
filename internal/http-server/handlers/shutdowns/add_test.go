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
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"
	"time"
)

// mockShutdownAdder is a mock implementation of ShutdownAdder interface
type mockShutdownAdder struct {
	addFunc    func(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
	linkFunc   func(ctx context.Context, shutdownID int64, fileIDs []int64) error
	parentFunc func(ctx context.Context, orgID int64) (*int64, error)
	addCalls   int
}

func (m *mockShutdownAdder) LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error {
	if m.linkFunc != nil {
		return m.linkFunc(ctx, shutdownID, fileIDs)
	}
	return nil
}

func (m *mockShutdownAdder) AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error) {
	m.addCalls++
	if m.addFunc != nil {
		return m.addFunc(ctx, req)
	}
	return 1, nil
}

// GetOrganizationParentID is required by the new CascadeChecker interface.
func (m *mockShutdownAdder) GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error) {
	if m.parentFunc != nil {
		return m.parentFunc(ctx, orgID)
	}
	return nil, nil
}

// Helper to create context with user claims using the middleware's test helper.
// Default role is "sc" so legacy happy-path tests continue to pass once the
// handler starts calling CheckCascadeStationAccess (sc has full access).
func contextWithClaims(ctx context.Context, userID int64) context.Context {
	claims := &token.Claims{
		UserID: userID,
		Name:   "Test User",
		Roles:  []string{"sc"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

// contextWithRoleClaims creates a context with specified role/orgID for RBAC tests.
func contextWithRoleClaims(ctx context.Context, userID, orgID int64, role string) context.Context {
	claims := &token.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           "Test User",
		Roles:          []string{role},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

func TestAdd(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	tests := []struct {
		name           string
		body           interface{}
		userID         int64
		mockResponse   int64
		mockError      error
		wantStatusCode int
		wantErrInBody  bool
	}{
		{
			name: "successful shutdown creation without idle discharge",
			body: addRequest{
				OrganizationID:    1,
				StartTime:         now,
				EndTime:           &later,
				Reason:            stringPtr("Maintenance"),
				GenerationLossMwh: float64Ptr(10.5),
			},
			userID:         1,
			mockResponse:   1,
			mockError:      nil,
			wantStatusCode: http.StatusCreated,
			wantErrInBody:  false,
		},
		{
			name: "successful shutdown creation with idle discharge",
			body: addRequest{
				OrganizationID:      1,
				StartTime:           now,
				EndTime:             &later,
				IdleDischargeVolume: float64Ptr(5.0),
			},
			userID:         1,
			mockResponse:   2,
			mockError:      nil,
			wantStatusCode: http.StatusCreated,
			wantErrInBody:  false,
		},
		{
			name: "validation error - missing required organization_id",
			body: addRequest{
				StartTime: now,
				EndTime:   &later,
			},
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name: "validation error - missing required start_time",
			body: addRequest{
				OrganizationID: 1,
				EndTime:        &later,
			},
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name: "validation error - idle_discharge_volume without end_time",
			body: addRequest{
				OrganizationID:      1,
				StartTime:           now,
				IdleDischargeVolume: float64Ptr(5.0),
			},
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name: "validation error - end_time before start_time",
			body: addRequest{
				OrganizationID: 1,
				StartTime:      later,
				EndTime:        &now,
			},
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name: "foreign key violation error",
			body: addRequest{
				OrganizationID: 999,
				StartTime:      now,
				EndTime:        &later,
			},
			userID:         1,
			mockError:      storage.ErrForeignKeyViolation,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name: "conflict - ongoing discharge exists without force",
			body: addRequest{
				OrganizationID:      1,
				StartTime:           now,
				EndTime:             &later,
				IdleDischargeVolume: float64Ptr(5.0),
			},
			userID:         1,
			mockError:      storage.ErrOngoingDischargeExists,
			wantStatusCode: http.StatusConflict,
			wantErrInBody:  true,
		},
		{
			name: "force close ongoing discharge and create new",
			body: addRequest{
				OrganizationID:      1,
				StartTime:           now,
				EndTime:             &later,
				IdleDischargeVolume: float64Ptr(5.0),
				Force:               true,
			},
			userID:         1,
			mockResponse:   3,
			mockError:      nil,
			wantStatusCode: http.StatusCreated,
			wantErrInBody:  false,
		},
		{
			name: "internal server error",
			body: addRequest{
				OrganizationID: 1,
				StartTime:      now,
				EndTime:        &later,
			},
			userID:         1,
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
		},
		{
			name:           "invalid JSON",
			body:           "invalid json",
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock adder
			mock := &mockShutdownAdder{
				addFunc: func(ctx context.Context, req dto.AddShutdownRequest) (int64, error) {
					if tt.mockError != nil {
						return 0, tt.mockError
					}
					return tt.mockResponse, nil
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
			req := httptest.NewRequest(http.MethodPost, "/shutdowns", bodyReader)
			req.Header.Set("Content-Type", "application/json")

			// Add claims to context (simulating authenticated user)
			ctx := contextWithClaims(req.Context(), tt.userID)
			req = req.WithContext(ctx)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Create logger
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Call handler
			handler := Add(logger, mock)
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}

			// Check if error is in body when expected
			if tt.wantErrInBody {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("expected error in response body, got: %v", resp)
				}
			}

			// For successful cases, verify ID is in response
			if tt.wantStatusCode == http.StatusCreated && !tt.wantErrInBody {
				var resp addResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				if err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.ID != tt.mockResponse {
					t.Errorf("response ID = %v, want %v", resp.ID, tt.mockResponse)
				}
			}
		})
	}
}

func TestAdd_NoUserID(t *testing.T) {
	mock := &mockShutdownAdder{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	body := addRequest{
		OrganizationID: 1,
		StartTime:      time.Now(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/shutdowns", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// Don't add user ID to context

	rr := httptest.NewRecorder()

	handler := Add(logger, mock)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %v, got %v", http.StatusUnauthorized, rr.Code)
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func intPtr(i int64) *int64 {
	return &i
}

// ---- Cascade RBAC tests (RED) ----

// runAddWithCtx is a small helper to reduce boilerplate in cascade tests.
func runAddWithCtx(t *testing.T, mock *mockShutdownAdder, ctx context.Context, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/shutdowns", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Add(logger, mock)
	handler.ServeHTTP(rr, req)
	return rr
}

func TestAdd_CascadeUser_OwnStation_OK(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	mock := &mockShutdownAdder{
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 10 {
				return intPtr(5), nil
			}
			return nil, nil
		},
	}

	ctx := contextWithRoleClaims(context.Background(), 1, 5, "cascade")
	body := addRequest{
		OrganizationID: 10,
		StartTime:      now,
		EndTime:        &later,
	}

	rr := runAddWithCtx(t, mock, ctx, body)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdd_CascadeUser_OwnCascade_DirectOrg_OK(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	mock := &mockShutdownAdder{
		// parent mock not required when orgID == claims.OrganizationID, but safe default.
	}

	ctx := contextWithRoleClaims(context.Background(), 1, 5, "cascade")
	body := addRequest{
		OrganizationID: 5,
		StartTime:      now,
		EndTime:        &later,
	}

	rr := runAddWithCtx(t, mock, ctx, body)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdd_CascadeUser_ForeignStation_Forbidden(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	mock := &mockShutdownAdder{
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 20 {
				return intPtr(7), nil
			}
			return nil, nil
		},
	}

	ctx := contextWithRoleClaims(context.Background(), 1, 5, "cascade")
	body := addRequest{
		OrganizationID: 20,
		StartTime:      now,
		EndTime:        &later,
	}

	rr := runAddWithCtx(t, mock, ctx, body)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.addCalls != 0 {
		t.Errorf("AddShutdown must NOT be called on forbidden, called %d times", mock.addCalls)
	}
}

func TestAdd_CascadeUser_ForeignStation_WithForce_Forbidden(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	mock := &mockShutdownAdder{
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 20 {
				return intPtr(7), nil
			}
			return nil, nil
		},
	}

	ctx := contextWithRoleClaims(context.Background(), 1, 5, "cascade")
	body := addRequest{
		OrganizationID:      20,
		StartTime:           now,
		EndTime:             &later,
		IdleDischargeVolume: float64Ptr(5.0),
		Force:               true,
	}

	rr := runAddWithCtx(t, mock, ctx, body)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 even with force, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.addCalls != 0 {
		t.Errorf("AddShutdown must NOT be called on forbidden (force must not bypass RBAC), called %d times", mock.addCalls)
	}
}

func TestAdd_CascadeUser_NoOrgID_Forbidden(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	mock := &mockShutdownAdder{
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			return intPtr(5), nil
		},
	}

	// cascade user with no org assigned
	ctx := contextWithRoleClaims(context.Background(), 1, 0, "cascade")
	body := addRequest{
		OrganizationID: 10,
		StartTime:      now,
		EndTime:        &later,
	}

	rr := runAddWithCtx(t, mock, ctx, body)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.addCalls != 0 {
		t.Errorf("AddShutdown must NOT be called, called %d times", mock.addCalls)
	}
}

func TestAdd_RaisUser_AnyStation_OK(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	parentCalled := false
	mock := &mockShutdownAdder{
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			parentCalled = true
			return intPtr(999), nil
		},
	}

	ctx := contextWithRoleClaims(context.Background(), 1, 999, "rais")
	body := addRequest{
		OrganizationID: 123, // any org
		StartTime:      now,
		EndTime:        &later,
	}

	rr := runAddWithCtx(t, mock, ctx, body)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if parentCalled {
		t.Error("parent lookup must NOT be called for rais role")
	}
}
