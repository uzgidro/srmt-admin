package locations

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/lib/model/filtration"
	"testing"
)

type mockLocationGetter struct {
	getFunc func(ctx context.Context, orgID int64) ([]filtration.Location, error)
}

func (m *mockLocationGetter) GetFiltrationLocationsByOrg(ctx context.Context, orgID int64) ([]filtration.Location, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, orgID)
	}
	return nil, nil
}

func TestGet(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockResult     []filtration.Location
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "missing organization_id",
			url:            "/filtration/locations",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid organization_id",
			url:            "/filtration/locations?organization_id=abc",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "successful get",
			url:            "/filtration/locations?organization_id=1",
			mockResult:     []filtration.Location{{ID: 1, Name: "Location 1"}},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "internal server error",
			url:            "/filtration/locations?organization_id=1",
			mockError:      errors.New("database error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLocationGetter{
				getFunc: func(ctx context.Context, orgID int64) ([]filtration.Location, error) {
					return tt.mockResult, tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Get(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}
