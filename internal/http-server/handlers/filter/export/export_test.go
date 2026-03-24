package export

import (
	"context"
	"errors"
	"testing"

	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"
)

type mockFiltrationGetter struct {
	orgIDs      []int64
	orgIDsErr   error
	summaryFunc func(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
	levelFunc   func(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
}

func (m *mockFiltrationGetter) GetFiltrationOrgIDs(_ context.Context) ([]int64, error) {
	return m.orgIDs, m.orgIDsErr
}

func (m *mockFiltrationGetter) GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error) {
	return m.summaryFunc(ctx, orgID, date)
}

func (m *mockFiltrationGetter) GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error) {
	return m.levelFunc(ctx, orgID, date)
}

func ptr(v float64) *float64 { return &v }

func TestBuildComparisons(t *testing.T) {
	baseSummary := &filtration.OrgFiltrationSummary{
		OrganizationID:   1,
		OrganizationName: "Org A",
		Locations:        []filtration.LocationReading{},
		Piezometers:      []filtration.PiezoReading{},
		PiezoCounts:      filtration.PiezometerCounts{Pressure: 2, NonPressure: 3},
	}

	mock := &mockFiltrationGetter{
		orgIDs: []int64{1},
		summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
			return baseSummary, nil
		},
		levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
			return ptr(284.5), ptr(12.3), nil
		},
	}

	t.Run("with both dates", func(t *testing.T) {
		result, err := buildComparisons(context.Background(), mock, "2026-02-02", "2026-01-10", "2026-01-15")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 comparison, got %d", len(result))
		}
		comp := result[0]
		if comp.Current.Date != "2026-02-02" {
			t.Errorf("current date = %q, want %q", comp.Current.Date, "2026-02-02")
		}
		if comp.HistoricalFilter == nil {
			t.Fatal("expected historical_filter to be non-nil")
		}
		if comp.HistoricalPiezo == nil {
			t.Fatal("expected historical_piezo to be non-nil")
		}
		if comp.HistoricalFilter.Date != "2026-01-10" {
			t.Errorf("filter date = %q, want %q", comp.HistoricalFilter.Date, "2026-01-10")
		}
		if comp.HistoricalPiezo.Date != "2026-01-15" {
			t.Errorf("piezo date = %q, want %q", comp.HistoricalPiezo.Date, "2026-01-15")
		}
	})

	t.Run("with same dates — optimization", func(t *testing.T) {
		callCount := 0
		countMock := &mockFiltrationGetter{
			orgIDs: []int64{1},
			summaryFunc: func(_ context.Context, _ int64, date string) (*filtration.OrgFiltrationSummary, error) {
				callCount++
				return baseSummary, nil
			},
			levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
				return ptr(284.5), ptr(12.3), nil
			},
		}

		result, err := buildComparisons(context.Background(), countMock, "2026-02-02", "2026-01-10", "2026-01-10")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 comparison, got %d", len(result))
		}
		// 2 summary calls: current + 1 historical (same date optimization)
		if callCount != 2 {
			t.Errorf("summary call count = %d, want 2 (same-date optimization)", callCount)
		}
	})

	t.Run("with empty dates — no historical", func(t *testing.T) {
		result, err := buildComparisons(context.Background(), mock, "2026-02-02", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 comparison, got %d", len(result))
		}
		if result[0].HistoricalFilter != nil {
			t.Error("expected historical_filter to be nil when filter_date is empty")
		}
		if result[0].HistoricalPiezo != nil {
			t.Error("expected historical_piezo to be nil when piezo_date is empty")
		}
	})

	t.Run("org not found — skipped", func(t *testing.T) {
		notFoundMock := &mockFiltrationGetter{
			orgIDs: []int64{1, 2},
			summaryFunc: func(_ context.Context, orgID int64, _ string) (*filtration.OrgFiltrationSummary, error) {
				if orgID == 1 {
					return nil, storage.ErrNotFound
				}
				return baseSummary, nil
			},
			levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
				return ptr(284.5), ptr(12.3), nil
			},
		}

		result, err := buildComparisons(context.Background(), notFoundMock, "2026-02-02", "2026-01-10", "2026-01-10")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 comparison (org 1 skipped), got %d", len(result))
		}
	})

	t.Run("db error propagates", func(t *testing.T) {
		errMock := &mockFiltrationGetter{
			orgIDs: []int64{1},
			summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
				return nil, errors.New("db error")
			},
			levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
				return ptr(284.5), ptr(12.3), nil
			},
		}

		_, err := buildComparisons(context.Background(), errMock, "2026-02-02", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestBuildExportSnapshot(t *testing.T) {
	summary := &filtration.OrgFiltrationSummary{
		OrganizationID:   1,
		OrganizationName: "Org A",
		Locations: []filtration.LocationReading{
			{Location: filtration.Location{ID: 10, Name: "Loc1"}, FlowRate: ptr(0.5)},
		},
		Piezometers: []filtration.PiezoReading{
			{Piezometer: filtration.Piezometer{ID: 20, Name: "PZ1"}, Level: ptr(118.5)},
		},
		PiezoCounts: filtration.PiezometerCounts{Pressure: 5, NonPressure: 3},
	}

	mock := &mockFiltrationGetter{
		summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
			return summary, nil
		},
		levelFunc: func(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
			return ptr(284.5), ptr(12.3), nil
		},
	}

	snap, err := buildExportSnapshot(context.Background(), mock, 1, "2026-01-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Date != "2026-01-10" {
		t.Errorf("date = %q, want %q", snap.Date, "2026-01-10")
	}
	if *snap.Level != 284.5 {
		t.Errorf("level = %v, want 284.5", *snap.Level)
	}
	if len(snap.Locations) != 1 {
		t.Errorf("locations count = %d, want 1", len(snap.Locations))
	}
	if len(snap.Piezometers) != 1 {
		t.Errorf("piezometers count = %d, want 1", len(snap.Piezometers))
	}

	t.Run("not found returns error", func(t *testing.T) {
		notFoundMock := &mockFiltrationGetter{
			summaryFunc: func(_ context.Context, _ int64, _ string) (*filtration.OrgFiltrationSummary, error) {
				return nil, storage.ErrNotFound
			},
		}
		_, err := buildExportSnapshot(context.Background(), notFoundMock, 1, "2026-01-10")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
