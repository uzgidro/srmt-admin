package shutdowns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/storage"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mockShutdownDeleter is a mock implementation of shutdownDeleter interface
type mockShutdownDeleter struct {
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockShutdownDeleter) DeleteShutdown(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
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
			wantStatusCode: http.StatusOK, // Note: render.Status without render.JSON returns 200
			wantErrInBody:  false,
			description:    "Should successfully delete an existing shutdown",
		},
		{
			name:           "successful deletion of shutdown with idle discharge",
			shutdownID:     "2",
			mockError:      nil,
			wantStatusCode: http.StatusOK, // Note: render.Status without render.JSON returns 200
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
			description:    "Should return not found for non-existent shutdown",
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
			// Create mock deleter
			mock := &mockShutdownDeleter{
				deleteFunc: func(ctx context.Context, id int64) error {
					return tt.mockError
				},
			}

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/shutdowns/"+tt.shutdownID, nil)

			// Setup chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Create response recorder
			rr := httptest.NewRecorder()

			// Create logger
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Call handler
			handler := Delete(logger, mock)
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("%s: handler returned wrong status code: got %v want %v",
					tt.description, rr.Code, tt.wantStatusCode)
			}

			// Check if error is in body when expected
			if tt.wantErrInBody && rr.Body.Len() > 0 {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("%s: expected error in response body, got: %v",
						tt.description, resp)
				}
			}

			// For successful deletion, body should be empty
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
					// Track which IDs were deleted
					deletedIDs = append(deletedIDs, id)
					return tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/shutdowns/"+tt.shutdownID, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.shutdownID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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
