package reservoirsummary

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"srmt-admin/internal/lib/dto"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
)

type mockSummaryGetter struct {
	summaries []*reservoirsummary.ResponseModel
	err       error
}

func (m *mockSummaryGetter) GetReservoirSummary(_ context.Context, _ string) ([]*reservoirsummary.ResponseModel, error) {
	return m.summaries, m.err
}

type mockStaticDataFetcher struct {
	data map[int64]*dto.OrganizationWithData
	err  error
}

func (m *mockStaticDataFetcher) FetchDataAtDayBegin(_ context.Context, _ string) (map[int64]*dto.OrganizationWithData, error) {
	return m.data, m.err
}

func ptrFloat(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64     { return &v }

func TestGet_NoAvg_ShouldNotFallbackToCurrentValues(t *testing.T) {
	orgID := int64(1)
	income := 100.0
	release := 50.0

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Income:         reservoirsummary.ValueResponse{Current: 0},
				Release:        reservoirsummary.ValueResponse{Current: 0},
			},
		},
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Income:     &income,
					Release:    &release,
					AvgIncome:  nil, // no avg
					AvgRelease: nil, // no avg
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}

	// Without averages, Income and Release should remain 0 (no fallback to current values)
	if result[0].Income.Current != 0 {
		t.Errorf("Income.Current: expected 0 (no fallback), got %f", result[0].Income.Current)
	}
	if result[0].Release.Current != 0 {
		t.Errorf("Release.Current: expected 0 (no fallback), got %f", result[0].Release.Current)
	}
	if result[0].Income.IsEdited != nil {
		t.Errorf("Income.IsEdited: expected nil, got %v", *result[0].Income.IsEdited)
	}
	if result[0].Release.IsEdited != nil {
		t.Errorf("Release.IsEdited: expected nil, got %v", *result[0].Release.IsEdited)
	}
}

func TestGet_WithAvg_ShouldUseAvgValues(t *testing.T) {
	orgID := int64(1)
	avgIncome := 80.0
	avgRelease := 40.0

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Income:         reservoirsummary.ValueResponse{Current: 0},
				Release:        reservoirsummary.ValueResponse{Current: 0},
			},
		},
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Income:     ptrFloat(100.0),
					Release:    ptrFloat(50.0),
					AvgIncome:  &avgIncome,
					AvgRelease: &avgRelease,
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result[0].Income.Current != avgIncome {
		t.Errorf("Income.Current: expected %f (avg), got %f", avgIncome, result[0].Income.Current)
	}
	if result[0].Release.Current != avgRelease {
		t.Errorf("Release.Current: expected %f (avg), got %f", avgRelease, result[0].Release.Current)
	}
}

func TestGet_AlreadyHasValues_ShouldNotOverwrite(t *testing.T) {
	orgID := int64(1)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Income:         reservoirsummary.ValueResponse{Current: 200},
				Release:        reservoirsummary.ValueResponse{Current: 150},
			},
		},
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					AvgIncome:  ptrFloat(80.0),
					AvgRelease: ptrFloat(40.0),
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var result []*reservoirsummary.ResponseModel
	json.NewDecoder(rr.Body).Decode(&result)

	// Existing values should not be overwritten
	if result[0].Income.Current != 200 {
		t.Errorf("Income.Current: expected 200 (unchanged), got %f", result[0].Income.Current)
	}
	if result[0].Release.Current != 150 {
		t.Errorf("Release.Current: expected 150 (unchanged), got %f", result[0].Release.Current)
	}
}
