package piezometers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/token"
	"testing"
)

type mockPiezometerGetter struct {
	getFunc func(ctx context.Context, orgID int64) ([]filtration.Piezometer, error)
}

func (m *mockPiezometerGetter) GetPiezometersByOrg(ctx context.Context, orgID int64) ([]filtration.Piezometer, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, orgID)
	}
	return nil, nil
}

func TestGet(t *testing.T) {
	scClaims := &token.Claims{
		UserID: 1,
		Roles:  []string{"sc"},
	}

	tests := []struct {
		name           string
		url            string
		mockResult     []filtration.Piezometer
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "missing organization_id",
			url:            "/filtration/piezometers",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid organization_id",
			url:            "/filtration/piezometers?organization_id=abc",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "successful get",
			url:            "/filtration/piezometers?organization_id=1",
			mockResult:     []filtration.Piezometer{{ID: 1, Name: "Piezometer 1", Type: filtration.PiezometerTypePressure}},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "internal server error",
			url:            "/filtration/piezometers?organization_id=1",
			mockError:      errors.New("database error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPiezometerGetter{
				getFunc: func(ctx context.Context, orgID int64) ([]filtration.Piezometer, error) {
					return tt.mockResult, tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := withAuth(Get(logger, mock), scClaims)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}
