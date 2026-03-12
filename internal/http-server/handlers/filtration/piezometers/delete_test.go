package piezometers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
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

type mockPiezometerOrgGetterForDelete struct {
	getOrgFunc func(ctx context.Context, id int64) (int64, error)
}

func (m *mockPiezometerOrgGetterForDelete) GetPiezometerOrgID(ctx context.Context, id int64) (int64, error) {
	if m.getOrgFunc != nil {
		return m.getOrgFunc(ctx, id)
	}
	return 0, nil
}

func TestDelete(t *testing.T) {
	// Use sc claims so org access always passes — we're testing delete logic, not org access.
	scClaims := &token.Claims{
		UserID: 1,
		Roles:  []string{"sc"},
	}

	tests := []struct {
		name           string
		piezometerID   string
		orgID          int64
		orgErr         error
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "successful deletion",
			piezometerID:   "1",
			orgID:          1,
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
			name:           "piezometer not found on org lookup",
			piezometerID:   "9999",
			orgErr:         storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "piezometer not found on delete",
			piezometerID:   "9999",
			orgID:          1,
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			piezometerID:   "1",
			orgID:          1,
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleterMock := &mockPiezometerDeleter{
				deleteFunc: func(ctx context.Context, id int64) error {
					return tt.mockError
				},
			}
			orgGetterMock := &mockPiezometerOrgGetterForDelete{
				getOrgFunc: func(ctx context.Context, id int64) (int64, error) {
					return tt.orgID, tt.orgErr
				},
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			r := chi.NewRouter()
			verifier := &mockTokenVerifier{claims: scClaims}
			r.Use(mwauth.Authenticator(verifier))
			r.Delete("/filtration/piezometers/{id}", Delete(logger, deleterMock, orgGetterMock))

			req := httptest.NewRequest(http.MethodDelete, "/filtration/piezometers/"+tt.piezometerID, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}
