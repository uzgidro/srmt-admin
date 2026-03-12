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

type mockPiezometerCounter struct {
	countFunc func(ctx context.Context, orgID int64) (filtration.PiezometerCounts, error)
}

func (m *mockPiezometerCounter) GetPiezometerCountsByOrg(ctx context.Context, orgID int64) (filtration.PiezometerCounts, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, orgID)
	}
	return filtration.PiezometerCounts{}, nil
}

func TestCounts(t *testing.T) {
	scClaims := &token.Claims{
		UserID: 1,
		Roles:  []string{"sc"},
	}

	tests := []struct {
		name           string
		url            string
		mockResult     filtration.PiezometerCounts
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "missing organization_id",
			url:            "/filtration/piezometers/counts",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid organization_id",
			url:            "/filtration/piezometers/counts?organization_id=abc",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "successful counts",
			url:            "/filtration/piezometers/counts?organization_id=1",
			mockResult:     filtration.PiezometerCounts{Pressure: 3, NonPressure: 5},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "internal server error",
			url:            "/filtration/piezometers/counts?organization_id=1",
			mockError:      errors.New("database error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPiezometerCounter{
				countFunc: func(ctx context.Context, orgID int64) (filtration.PiezometerCounts, error) {
					return tt.mockResult, tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := withAuth(Counts(logger, mock), scClaims)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}
