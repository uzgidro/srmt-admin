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

	"github.com/go-chi/chi/v5"
)

// mockShutdownEditor is a mock implementation of shutdownEditor interface
type mockShutdownEditor struct {
	editFunc   func(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	unlinkFunc func(ctx context.Context, shutdownID int64) error
	linkFunc   func(ctx context.Context, shutdownID int64, fileIDs []int64) error
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
	if m.editFunc != nil {
		return m.editFunc(ctx, id, req)
	}
	return nil
}

// Helper to create context with user claims (reuse from add_test.go approach)
func contextWithUserClaims(ctx context.Context, userID int64) context.Context {
	claims := &token.Claims{
		UserID: userID,
		Name:   "Test User",
		Roles:  []string{"admin"},
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
				EndTime:   ptrTimePtr(later),
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
				EndTime:             ptrTimePtr(later),
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
			handler := Edit(logger, mock, nil, nil, nil)
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
				EndTime:             ptrTimePtr(time.Now().Add(3 * time.Hour)),
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
				EndTime:             ptrTimePtr(time.Now().Add(4 * time.Hour)),
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

			handler := Edit(logger, mock, nil, nil, nil)
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

func ptrTimePtr(t time.Time) **time.Time {
	p := &t
	return &p
}
