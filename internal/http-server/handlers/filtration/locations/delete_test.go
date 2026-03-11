package locations

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/storage"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockLocationDeleter struct {
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockLocationDeleter) DeleteFiltrationLocation(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name           string
		locationID     string
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "successful deletion",
			locationID:     "1",
			mockError:      nil,
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "invalid id parameter",
			locationID:     "invalid",
			mockError:      nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "location not found",
			locationID:     "9999",
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			locationID:     "1",
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLocationDeleter{
				deleteFunc: func(ctx context.Context, id int64) error {
					return tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/filtration/locations/"+tt.locationID, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.locationID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Delete(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}
