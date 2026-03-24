package comparison

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
	"srmt-admin/internal/token"
)

type mockComparisonDataGetter struct {
	orgIDs         []int64
	orgIDsErr      error
	summaryFunc    func(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
	levelFunc      func(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
}

func (m *mockComparisonDataGetter) GetFiltrationOrgIDs(_ context.Context) ([]int64, error) {
	return m.orgIDs, m.orgIDsErr
}

func (m *mockComparisonDataGetter) GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error) {
	return m.summaryFunc(ctx, orgID, date)
}

func (m *mockComparisonDataGetter) GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error) {
	return m.levelFunc(ctx, orgID, date)
}

func TestGetData(t *testing.T) {
	scClaims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	reservoirClaims := &token.Claims{UserID: 2, Roles: []string{"reservoir"}, OrganizationID: 5}
	noOrgClaims := &token.Claims{UserID: 3, Roles: []string{"reservoir"}}

	lvl := 284.5
	vol := 12.3

	baseSummary := &filtration.OrgFiltrationSummary{
		OrganizationID:   1,
		OrganizationName: "Org A",
		Locations:        []filtration.LocationReading{},
		Piezometers:      []filtration.PiezoReading{},
		PiezoCounts:      filtration.PiezometerCounts{Pressure: 2, NonPressure: 3},
	}

	successMock := &mockComparisonDataGetter{
		orgIDs: []int64{1},
		summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
			return baseSummary, nil
		},
		levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
			return &lvl, &vol, nil
		},
	}

	tests := []struct {
		name       string
		url        string
		claims     *token.Claims
		mock       *mockComparisonDataGetter
		wantStatus int
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "missing date",
			url:        "/comparison/data?filter_date=2025-01-01&piezo_date=2025-01-01",
			claims:     scClaims,
			mock:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "missing filter_date",
			url:        "/comparison/data?date=2025-01-01&piezo_date=2025-01-01",
			claims:     scClaims,
			mock:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "missing piezo_date",
			url:        "/comparison/data?date=2025-01-01&filter_date=2025-01-01",
			claims:     scClaims,
			mock:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid date format",
			url:        "/comparison/data?date=bad&filter_date=2025-01-01&piezo_date=2025-01-01",
			claims:     scClaims,
			mock:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "no organization assigned",
			url:        "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-09-03",
			claims:     noOrgClaims,
			mock:       successMock,
			wantStatus: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:   "org not found — skipped",
			url:    "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-09-03",
			claims: scClaims,
			mock: &mockComparisonDataGetter{
				orgIDs: []int64{1},
				summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
					return nil, storage.ErrNotFound
				},
				levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
					return &lvl, &vol, nil
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:   "db error",
			url:    "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-09-03",
			claims: scClaims,
			mock: &mockComparisonDataGetter{
				orgIDs: []int64{1},
				summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
					return nil, errors.New("db error")
				},
				levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
					return &lvl, &vol, nil
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "supervisor — success",
			url:        "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-09-03",
			claims:     scClaims,
			mock:       successMock,
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:   "regular user — success",
			url:    "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-09-03",
			claims: reservoirClaims,
			mock: &mockComparisonDataGetter{
				summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
					return &filtration.OrgFiltrationSummary{
						OrganizationID:   5,
						OrganizationName: "My Org",
						Locations:        []filtration.LocationReading{},
						Piezometers:      []filtration.PiezoReading{},
						PiezoCounts:      filtration.PiezometerCounts{},
					}, nil
				},
				levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
					return &lvl, &vol, nil
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "same filter_date and piezo_date — optimization",
			url:        "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-11-15",
			claims:     scClaims,
			mock:       successMock,
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := withAuth(GetData(log, tt.mock), tt.claims)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantErr {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("expected error in body, got: %v", resp)
				}
				return
			}

			var result []filtration.OrgComparisonV2
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("result len = %d, want %d", len(result), tt.wantLen)
			}

			if tt.wantLen > 0 {
				comp := result[0]
				if comp.Current.Date != "2025-01-01" {
					t.Errorf("current date = %q, want %q", comp.Current.Date, "2025-01-01")
				}
				if comp.HistoricalFilter == nil {
					t.Error("expected historical_filter to be non-nil")
				}
				if comp.HistoricalPiezo == nil {
					t.Error("expected historical_piezo to be non-nil")
				}
			}
		})
	}
}

func TestGetData_SameDateOptimization(t *testing.T) {
	scClaims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	lvl := 284.5
	vol := 12.3

	callCount := 0
	mock := &mockComparisonDataGetter{
		orgIDs: []int64{1},
		summaryFunc: func(_ context.Context, _ int64, date string) (*filtration.OrgFiltrationSummary, error) {
			callCount++
			return &filtration.OrgFiltrationSummary{
				OrganizationID:   1,
				OrganizationName: "Org A",
				Locations:        []filtration.LocationReading{},
				Piezometers:      []filtration.PiezoReading{},
				PiezoCounts:      filtration.PiezometerCounts{},
			}, nil
		},
		levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
			return &lvl, &vol, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/comparison/data?date=2025-01-01&filter_date=2024-11-15&piezo_date=2024-11-15", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := withAuth(GetData(log, mock), scClaims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	// When filter_date == piezo_date, we expect 2 summary calls (current + one historical), not 3
	if callCount != 2 {
		t.Errorf("summary call count = %d, want 2 (same-date optimization)", callCount)
	}

	var result []filtration.OrgComparisonV2
	json.Unmarshal(rr.Body.Bytes(), &result)
	if len(result) != 1 {
		t.Fatalf("result len = %d, want 1", len(result))
	}
	if result[0].HistoricalFilter.Date != result[0].HistoricalPiezo.Date {
		t.Errorf("expected historical_filter and historical_piezo to have same date, got %q vs %q",
			result[0].HistoricalFilter.Date, result[0].HistoricalPiezo.Date)
	}
}
