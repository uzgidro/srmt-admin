package piezometers

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

type mockPiezometerDeleter struct {
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockPiezometerDeleter) DeletePiezometer(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name           string
		piezometerID   string
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "successful deletion",
			piezometerID:   "1",
			mockError:      nil,
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "invalid id parameter",
			piezometerID:   "invalid",
			mockError:      nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "piezometer not found",
			piezometerID:   "9999",
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			piezometerID:   "1",
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPiezometerDeleter{
				deleteFunc: func(ctx context.Context, id int64) error {
					return tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/filtration/piezometers/"+tt.piezometerID, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.piezometerID)
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
