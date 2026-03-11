package summary

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"
)

type mockSummaryGetter struct {
	getFunc func(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
}

func (m *mockSummaryGetter) GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error) {
	return m.getFunc(ctx, orgID, date)
}

func TestGet(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockReturn     *filtration.OrgFiltrationSummary
		mockErr        error
		wantStatusCode int
		wantErrInBody  bool
	}{
		{
			name:           "missing organization_id",
			url:            "/filtration/summary?date=2025-01-01",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name:           "invalid organization_id",
			url:            "/filtration/summary?organization_id=abc&date=2025-01-01",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name:           "missing date",
			url:            "/filtration/summary?organization_id=1",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name:    "not found",
			url:     "/filtration/summary?organization_id=1&date=2025-01-01",
			mockErr: storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
			wantErrInBody:  true,
		},
		{
			name: "successful get",
			url:  "/filtration/summary?organization_id=1&date=2025-01-01",
			mockReturn: &filtration.OrgFiltrationSummary{
				OrganizationID:   1,
				OrganizationName: "Test Org",
				Locations:        []filtration.LocationReading{},
				Piezometers:      []filtration.PiezoReading{},
				PiezoCounts:      filtration.PiezometerCounts{Pressure: 2, NonPressure: 3},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "internal error",
			url:            "/filtration/summary?organization_id=1&date=2025-01-01",
			mockErr:        errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSummaryGetter{
				getFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
					return tt.mockReturn, tt.mockErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Get(log, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatusCode)
			}

			if tt.wantErrInBody {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("expected error in body, got: %v", resp)
				}
				return
			}

			var result filtration.OrgFiltrationSummary
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if result.OrganizationID != tt.mockReturn.OrganizationID {
				t.Errorf("organization_id = %d, want %d", result.OrganizationID, tt.mockReturn.OrganizationID)
			}
			if result.OrganizationName != tt.mockReturn.OrganizationName {
				t.Errorf("organization_name = %q, want %q", result.OrganizationName, tt.mockReturn.OrganizationName)
			}
		})
	}
}
