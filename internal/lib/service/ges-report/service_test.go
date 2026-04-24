package gesreportservice

import (
	"context"
	"io"
	"log/slog"
	"math"
	"testing"
	"time"

	model "srmt-admin/internal/lib/model/ges-report"
)

// discardLogger returns a logger whose output is silently discarded — used in unit tests.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockRepo implements Repository with date-based dispatch for GetGESDailyDataBatch.
type mockRepo struct {
	todayData      []model.RawDailyRow
	yesterdayData  []model.RawDailyRow
	prevYearData   []model.RawDailyRow
	todayDate      string
	yesterdayDate  string
	prevYearDate   string
	aggregations   []model.ProductionAggregation
	plans          []model.PlanRow
	discharges     []model.IdleDischargeRow
	cascadeWeather map[model.CascadeWeatherKey]*model.CascadeWeather
	frozen         map[int64]map[string]float64
}

func (m *mockRepo) GetGESDailyDataBatch(_ context.Context, date string) ([]model.RawDailyRow, error) {
	switch date {
	case m.yesterdayDate:
		return m.yesterdayData, nil
	case m.prevYearDate:
		return m.prevYearData, nil
	default:
		return m.todayData, nil
	}
}

func (m *mockRepo) GetGESProductionAggregations(_ context.Context, _ string) ([]model.ProductionAggregation, error) {
	return m.aggregations, nil
}

func (m *mockRepo) GetGESPlansForReport(_ context.Context, _ int, _ []int) ([]model.PlanRow, error) {
	return m.plans, nil
}

func (m *mockRepo) GetIdleDischargesForDate(_ context.Context, _, _ time.Time) ([]model.IdleDischargeRow, error) {
	return m.discharges, nil
}

func (m *mockRepo) GetCascadeDailyWeatherBatch(_ context.Context, orgIDs []int64, dates []string) (map[model.CascadeWeatherKey]*model.CascadeWeather, error) {
	result := make(map[model.CascadeWeatherKey]*model.CascadeWeather)
	for _, id := range orgIDs {
		for _, d := range dates {
			key := model.CascadeWeatherKey{OrgID: id, Date: d}
			if w, ok := m.cascadeWeather[key]; ok {
				result[key] = w
			}
		}
	}
	return result, nil
}

func (m *mockRepo) GetFrozenDefaults(_ context.Context) (map[int64]map[string]float64, error) {
	if m.frozen == nil {
		return map[int64]map[string]float64{}, nil
	}
	out := make(map[int64]map[string]float64, len(m.frozen))
	for k, v := range m.frozen {
		inner := make(map[string]float64, len(v))
		for kk, vv := range v {
			inner[kk] = vv
		}
		out[k] = inner
	}
	return out, nil
}

// ptr returns a pointer to the given float64.
func ptr(v float64) *float64 { return &v }

// ptrStr returns a pointer to the given string.
func ptrStr(v string) *string { return &v }

// approxEqual checks two float64 values are within epsilon.
func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// mustLoc loads a time.Location and panics on error.
func mustLoc(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// TestBuildReport_SingleStation verifies basic report assembly:
// power computation, MTD/YTD, plan fulfillment percentage, YoY, and grand total.
func TestBuildReport_SingleStation(t *testing.T) {
	cascadeID := int64(1)
	cascadeName := "Cascade A"
	orgID := int64(100)

	// 24 MWh daily → power = 24*1000/24 = 1000 MWt
	// MTD=50, YTD=200, PrevYearYTD=160
	// quarterly plan = 300 (months 1+2+3), monthly = 100
	// fulfillment = 200/300
	// YoY growth = (200/160) - 1 = 0.25
	// YoY diff = 200 - 160 = 40

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Alpha",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 24.0,
				WorkingAggregates:     3,
				InstalledCapacityMWt:  500.0,
				TotalAggregates:       4,
				HasReservoir:          true,
			},
		},
		yesterdayData: nil,
		prevYearData:  nil,
		aggregations: []model.ProductionAggregation{
			{
				OrganizationID: orgID,
				MTD:            50.0,
				YTD:            200.0,
				PrevYearMTD:    45.0,
				PrevYearYTD:    160.0,
			},
		},
		// Quarter for March: months 1,2,3 — plan per month = 100
		plans: []model.PlanRow{
			{OrganizationID: orgID, Year: 2026, Month: 1, PlanMlnKWh: 100.0},
			{OrganizationID: orgID, Year: 2026, Month: 2, PlanMlnKWh: 100.0},
			{OrganizationID: orgID, Year: 2026, Month: 3, PlanMlnKWh: 100.0},
		},
		discharges: nil,
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if report.Date != "2026-03-13" {
		t.Errorf("date: got %q, want %q", report.Date, "2026-03-13")
	}
	if len(report.Cascades) != 1 {
		t.Fatalf("cascades: got %d, want 1", len(report.Cascades))
	}

	cascade := report.Cascades[0]
	if len(cascade.Stations) != 1 {
		t.Fatalf("stations: got %d, want 1", len(cascade.Stations))
	}

	st := cascade.Stations[0]

	// Power = 24 * 1000 / 24 = 1000 MWt
	if !approxEqual(st.Current.PowerMWt, 1000.0) {
		t.Errorf("power: got %.4f, want 1000.0", st.Current.PowerMWt)
	}

	// MTD and YTD
	if !approxEqual(st.Aggregations.MTDProductionMlnKWh, 50.0) {
		t.Errorf("MTD: got %.4f, want 50.0", st.Aggregations.MTDProductionMlnKWh)
	}
	if !approxEqual(st.Aggregations.YTDProductionMlnKWh, 200.0) {
		t.Errorf("YTD: got %.4f, want 200.0", st.Aggregations.YTDProductionMlnKWh)
	}

	// Plan
	if !approxEqual(st.Plan.MonthlyPlanMlnKWh, 100.0) {
		t.Errorf("monthly plan: got %.4f, want 100.0", st.Plan.MonthlyPlanMlnKWh)
	}
	if !approxEqual(st.Plan.QuarterlyPlanMlnKWh, 300.0) {
		t.Errorf("quarterly plan: got %.4f, want 300.0", st.Plan.QuarterlyPlanMlnKWh)
	}
	if st.Plan.FulfillmentPct == nil {
		t.Fatal("fulfillment pct is nil")
	}
	wantFulfillment := 200.0 / 300.0
	if !approxEqual(*st.Plan.FulfillmentPct, wantFulfillment) {
		t.Errorf("fulfillment: got %.6f, want %.6f", *st.Plan.FulfillmentPct, wantFulfillment)
	}
	if !approxEqual(st.Plan.DifferenceMlnKWh, 200.0-300.0) {
		t.Errorf("plan diff: got %.4f, want -100.0", st.Plan.DifferenceMlnKWh)
	}

	// YoY
	if st.YoY.GrowthRate == nil {
		t.Fatal("YoY growth rate is nil")
	}
	wantGrowth := (200.0 / 160.0) - 1.0
	if !approxEqual(*st.YoY.GrowthRate, wantGrowth) {
		t.Errorf("yoy growth: got %.6f, want %.6f", *st.YoY.GrowthRate, wantGrowth)
	}
	if !approxEqual(st.YoY.DifferenceMlnKWh, 40.0) {
		t.Errorf("yoy diff: got %.4f, want 40.0", st.YoY.DifferenceMlnKWh)
	}

	// Grand total must match single station values.
	gt := report.GrandTotal
	if gt == nil {
		t.Fatal("grand total is nil")
	}
	if !approxEqual(gt.DailyProductionMlnKWh, 24.0) {
		t.Errorf("grand total production: got %.4f, want 24.0", gt.DailyProductionMlnKWh)
	}
	if !approxEqual(gt.YTDProductionMlnKWh, 200.0) {
		t.Errorf("grand total YTD: got %.4f, want 200.0", gt.YTDProductionMlnKWh)
	}
	if gt.FulfillmentPct == nil {
		t.Fatal("grand total fulfillment pct is nil")
	}
	if !approxEqual(*gt.FulfillmentPct, wantFulfillment) {
		t.Errorf("grand total fulfillment: got %.6f, want %.6f", *gt.FulfillmentPct, wantFulfillment)
	}
}

// TestBuildReport_DiffsFromYesterday verifies that diffs are computed correctly
// when yesterday data is available.
func TestBuildReport_DiffsFromYesterday(t *testing.T) {
	cascadeID := int64(2)
	cascadeName := "Cascade B"
	orgID := int64(200)

	// Today: level=100m, production=12 MlnKWh
	// Yesterday: level=99.5m, production=10 MlnKWh
	// Expected: levelChange = (100 - 99.5) * 100 = 50 cm
	// productionChange = 12 - 10 = 2 MlnKWh

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Beta",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 12.0,
				WorkingAggregates:     2,
				WaterLevelM:           ptr(100.0),
				InstalledCapacityMWt:  200.0,
				TotalAggregates:       3,
			},
		},
		yesterdayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Beta",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-12",
				DailyProductionMlnKWh: 10.0,
				WorkingAggregates:     2,
				WaterLevelM:           ptr(99.5),
				InstalledCapacityMWt:  200.0,
				TotalAggregates:       3,
			},
		},
		prevYearData: nil,
		aggregations: []model.ProductionAggregation{
			{OrganizationID: orgID, MTD: 100.0, YTD: 300.0, PrevYearMTD: 90.0, PrevYearYTD: 280.0},
		},
		plans: []model.PlanRow{
			{OrganizationID: orgID, Year: 2026, Month: 1, PlanMlnKWh: 80.0},
			{OrganizationID: orgID, Year: 2026, Month: 2, PlanMlnKWh: 80.0},
			{OrganizationID: orgID, Year: 2026, Month: 3, PlanMlnKWh: 80.0},
		},
		discharges: nil,
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	diffs := st.Diffs

	// Level change in cm.
	if diffs.LevelChangeCm == nil {
		t.Fatal("level change is nil")
	}
	if !approxEqual(*diffs.LevelChangeCm, 50.0) {
		t.Errorf("level change: got %.4f cm, want 50.0 cm", *diffs.LevelChangeCm)
	}

	// Production change.
	if diffs.ProductionChange == nil {
		t.Fatal("production change is nil")
	}
	if !approxEqual(*diffs.ProductionChange, 2.0) {
		t.Errorf("production change: got %.4f, want 2.0", *diffs.ProductionChange)
	}

	// Power change: today power = 12*1000/24 = 500, yesterday = 10*1000/24 ≈ 416.667
	wantPowerChange := 12.0*1000.0/24.0 - 10.0*1000.0/24.0
	if diffs.PowerChangeMWt == nil {
		t.Fatal("power change is nil")
	}
	if !approxEqual(*diffs.PowerChangeMWt, wantPowerChange) {
		t.Errorf("power change: got %.4f, want %.4f", *diffs.PowerChangeMWt, wantPowerChange)
	}
}

// TestBuildReport_IdleDischarge verifies that discharge data from the repo
// is correctly attached to the station report.
func TestBuildReport_IdleDischarge(t *testing.T) {
	cascadeID := int64(3)
	cascadeName := "Cascade C"
	orgID := int64(300)
	reason := "Flood prevention"

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Gamma",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 8.0,
				WorkingAggregates:     1,
				InstalledCapacityMWt:  150.0,
				TotalAggregates:       2,
			},
		},
		yesterdayData: nil,
		prevYearData:  nil,
		aggregations: []model.ProductionAggregation{
			{OrganizationID: orgID, MTD: 60.0, YTD: 150.0},
		},
		plans: []model.PlanRow{
			{OrganizationID: orgID, Year: 2026, Month: 1, PlanMlnKWh: 60.0},
			{OrganizationID: orgID, Year: 2026, Month: 2, PlanMlnKWh: 60.0},
			{OrganizationID: orgID, Year: 2026, Month: 3, PlanMlnKWh: 60.0},
		},
		discharges: []model.IdleDischargeRow{
			{
				OrganizationID: orgID,
				FlowRateM3s:    120.5,
				VolumeMlnM3:    0.432,
				Reason:         &reason,
				IsOngoing:      true,
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	if st.IdleDischarge == nil {
		t.Fatal("IdleDischarge is nil, expected discharge data")
	}

	// FlowRateM3s = VolumeMlnM3 / 0.0864 = 0.432 / 0.0864 = 5.0
	if !approxEqual(st.IdleDischarge.FlowRateM3s, 5.0) {
		t.Errorf("flow rate: got %.4f, want 5.0", st.IdleDischarge.FlowRateM3s)
	}
	if !approxEqual(st.IdleDischarge.VolumeMlnM3, 0.43) {
		t.Errorf("volume: got %.4f, want 0.43 (0.432 rounded to 2 dp)", st.IdleDischarge.VolumeMlnM3)
	}
	if st.IdleDischarge.Reason == nil || *st.IdleDischarge.Reason != reason {
		t.Errorf("reason: got %v, want %q", st.IdleDischarge.Reason, reason)
	}
	if !st.IdleDischarge.IsOngoing {
		t.Error("IsOngoing: got false, want true")
	}

	// Grand total should include idle discharge flow rate.
	gt := report.GrandTotal
	if gt == nil {
		t.Fatal("grand total is nil")
	}
	if !approxEqual(gt.IdleDischargeM3s, 5.0) {
		t.Errorf("grand total idle discharge: got %.4f, want 5.0", gt.IdleDischargeM3s)
	}
}

// TestBuildReport_MultipleDischargesPerOrg verifies summing behaviour when
// there are multiple discharge rows for the same organisation.
func TestBuildReport_MultipleDischargesPerOrg(t *testing.T) {
	cascadeID := int64(4)
	cascadeName := "Cascade D"
	orgID := int64(400)
	reason1 := "High water"
	reason2 := "Maintenance"

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Delta",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 5.0,
				WorkingAggregates:     1,
				InstalledCapacityMWt:  100.0,
				TotalAggregates:       2,
			},
		},
		yesterdayData: nil,
		prevYearData:  nil,
		aggregations:  nil,
		plans:         nil,
		discharges: []model.IdleDischargeRow{
			{OrganizationID: orgID, FlowRateM3s: 50.0, VolumeMlnM3: 0.1, Reason: &reason1, IsOngoing: false},
			{OrganizationID: orgID, FlowRateM3s: 30.0, VolumeMlnM3: 0.2, Reason: &reason2, IsOngoing: true},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	st := report.Cascades[0].Stations[0]
	if st.IdleDischarge == nil {
		t.Fatal("IdleDischarge is nil")
	}

	// Volumes summed: 0.1 + 0.2 = 0.3; FlowRate = 0.3 / 0.0864 ≈ 3.4722 → rounded 3.47.
	if !approxEqual(st.IdleDischarge.FlowRateM3s, 3.47) {
		t.Errorf("derived flow rate: got %.4f, want 3.47 (rounded)", st.IdleDischarge.FlowRateM3s)
	}
	// Volumes summed: 0.1 + 0.2 = 0.3
	if !approxEqual(st.IdleDischarge.VolumeMlnM3, 0.3) {
		t.Errorf("summed volume: got %.4f, want 0.3", st.IdleDischarge.VolumeMlnM3)
	}
	// First reason kept.
	if st.IdleDischarge.Reason == nil || *st.IdleDischarge.Reason != reason1 {
		t.Errorf("reason: got %v, want %q", st.IdleDischarge.Reason, reason1)
	}
	// IsOngoing = true because second row is ongoing.
	if !st.IdleDischarge.IsOngoing {
		t.Error("IsOngoing: got false, want true (second row has IsOngoing=true)")
	}
}

// TestBuildReport_CascadeWeather verifies that per-cascade weather is loaded
// via GetCascadeDailyWeatherBatch and attached to CascadeReport.Weather, with
// both current-day and previous-year temperatures populated correctly.
func TestBuildReport_CascadeWeather(t *testing.T) {
	cascadeID := int64(10)
	cascadeName := "Cascade W"
	orgID := int64(1000)

	repo := &mockRepo{
		todayDate:     "2026-04-13",
		yesterdayDate: "2026-04-12",
		prevYearDate:  "2025-04-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station W",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-04-13",
				DailyProductionMlnKWh: 10.0,
				WorkingAggregates:     2,
				InstalledCapacityMWt:  200.0,
				TotalAggregates:       3,
			},
		},
		yesterdayData: nil,
		prevYearData:  nil,
		aggregations: []model.ProductionAggregation{
			{OrganizationID: orgID, MTD: 40.0, YTD: 120.0},
		},
		plans: []model.PlanRow{
			{OrganizationID: orgID, Year: 2026, Month: 4, PlanMlnKWh: 60.0},
			{OrganizationID: orgID, Year: 2026, Month: 5, PlanMlnKWh: 60.0},
			{OrganizationID: orgID, Year: 2026, Month: 6, PlanMlnKWh: 60.0},
		},
		discharges: nil,
		cascadeWeather: map[model.CascadeWeatherKey]*model.CascadeWeather{
			{OrgID: cascadeID, Date: "2026-04-13"}: {
				Temperature: ptr(22.5),
				Condition:   ptrStr("01d"),
			},
			{OrgID: cascadeID, Date: "2025-04-13"}: {
				Temperature: ptr(18.0),
				Condition:   ptrStr("02d"),
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-04-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if len(report.Cascades) != 1 {
		t.Fatalf("cascades: got %d, want 1", len(report.Cascades))
	}

	cascade := report.Cascades[0]
	if cascade.Weather == nil {
		t.Fatal("cascade.Weather is nil, expected weather data")
	}
	if cascade.Weather.Temperature == nil {
		t.Fatal("cascade.Weather.Temperature is nil")
	}
	if !approxEqual(*cascade.Weather.Temperature, 22.5) {
		t.Errorf("Weather.Temperature: got %.4f, want 22.5", *cascade.Weather.Temperature)
	}
	if cascade.Weather.Condition == nil {
		t.Fatal("cascade.Weather.Condition is nil")
	}
	if *cascade.Weather.Condition != "01d" {
		t.Errorf("Weather.Condition: got %q, want %q", *cascade.Weather.Condition, "01d")
	}
	if cascade.Weather.PrevYearTemperature == nil {
		t.Fatal("cascade.Weather.PrevYearTemperature is nil")
	}
	if !approxEqual(*cascade.Weather.PrevYearTemperature, 18.0) {
		t.Errorf("Weather.PrevYearTemperature: got %.4f, want 18.0", *cascade.Weather.PrevYearTemperature)
	}
	if cascade.Weather.PrevYearCondition == nil {
		t.Fatal("cascade.Weather.PrevYearCondition is nil")
	}
	if *cascade.Weather.PrevYearCondition != "02d" {
		t.Errorf("Weather.PrevYearCondition: got %q, want %q", *cascade.Weather.PrevYearCondition, "02d")
	}
}

// twoCascadeRepo builds a mockRepo with two cascades, each with a single station.
// Cascade A (id=1) → station 100, daily=24, YTD=200
// Cascade B (id=2) → station 200, daily=12, YTD=100
func twoCascadeRepo() *mockRepo {
	cascadeAID := int64(1)
	cascadeAName := "Cascade A"
	cascadeBID := int64(2)
	cascadeBName := "Cascade B"

	return &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        100,
				OrganizationName:      "Station Alpha",
				CascadeID:             &cascadeAID,
				CascadeName:           &cascadeAName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 24.0,
				WorkingAggregates:     3,
				InstalledCapacityMWt:  500.0,
				TotalAggregates:       4,
			},
			{
				OrganizationID:        200,
				OrganizationName:      "Station Beta",
				CascadeID:             &cascadeBID,
				CascadeName:           &cascadeBName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 12.0,
				WorkingAggregates:     2,
				InstalledCapacityMWt:  200.0,
				TotalAggregates:       3,
			},
		},
		aggregations: []model.ProductionAggregation{
			{OrganizationID: 100, MTD: 50.0, YTD: 200.0, PrevYearYTD: 160.0},
			{OrganizationID: 200, MTD: 30.0, YTD: 100.0, PrevYearYTD: 80.0},
		},
		plans: []model.PlanRow{
			{OrganizationID: 100, Year: 2026, Month: 1, PlanMlnKWh: 100.0},
			{OrganizationID: 100, Year: 2026, Month: 2, PlanMlnKWh: 100.0},
			{OrganizationID: 100, Year: 2026, Month: 3, PlanMlnKWh: 100.0},
			{OrganizationID: 200, Year: 2026, Month: 1, PlanMlnKWh: 50.0},
			{OrganizationID: 200, Year: 2026, Month: 2, PlanMlnKWh: 50.0},
			{OrganizationID: 200, Year: 2026, Month: 3, PlanMlnKWh: 50.0},
		},
	}
}

// TestBuildDailyReport_NoFilter verifies that passing nil cascadeOrgID returns
// every cascade and aggregates GrandTotal across all of them.
func TestBuildDailyReport_NoFilter(t *testing.T) {
	repo := twoCascadeRepo()
	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if len(report.Cascades) != 2 {
		t.Fatalf("cascades: got %d, want 2", len(report.Cascades))
	}
	if report.GrandTotal == nil {
		t.Fatal("grand total is nil")
	}
	// 24 + 12 = 36
	if !approxEqual(report.GrandTotal.DailyProductionMlnKWh, 36.0) {
		t.Errorf("grand total daily production: got %.4f, want 36.0", report.GrandTotal.DailyProductionMlnKWh)
	}
	// 200 + 100 = 300
	if !approxEqual(report.GrandTotal.YTDProductionMlnKWh, 300.0) {
		t.Errorf("grand total YTD: got %.4f, want 300.0", report.GrandTotal.YTDProductionMlnKWh)
	}
}

// TestBuildDailyReport_FilterByCascade verifies that passing a known cascade ID
// restricts the report to that single cascade and recomputes GrandTotal as
// equal to that cascade's summary.
func TestBuildDailyReport_FilterByCascade(t *testing.T) {
	repo := twoCascadeRepo()
	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	cascadeBID := int64(2)
	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", &cascadeBID)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if len(report.Cascades) != 1 {
		t.Fatalf("cascades: got %d, want 1", len(report.Cascades))
	}
	if report.Cascades[0].CascadeID != cascadeBID {
		t.Errorf("cascade id: got %d, want %d", report.Cascades[0].CascadeID, cascadeBID)
	}
	if report.GrandTotal == nil {
		t.Fatal("grand total is nil")
	}
	// GrandTotal must equal Cascade B's summary: daily=12, YTD=100.
	if !approxEqual(report.GrandTotal.DailyProductionMlnKWh, 12.0) {
		t.Errorf("grand total daily production: got %.4f, want 12.0", report.GrandTotal.DailyProductionMlnKWh)
	}
	if !approxEqual(report.GrandTotal.YTDProductionMlnKWh, 100.0) {
		t.Errorf("grand total YTD: got %.4f, want 100.0", report.GrandTotal.YTDProductionMlnKWh)
	}
	// Sanity: GrandTotal matches the lone cascade's Summary.
	if report.Cascades[0].Summary == nil {
		t.Fatal("cascade summary is nil")
	}
	if !approxEqual(report.GrandTotal.DailyProductionMlnKWh, report.Cascades[0].Summary.DailyProductionMlnKWh) {
		t.Errorf("grand total != cascade summary (daily): %.4f vs %.4f",
			report.GrandTotal.DailyProductionMlnKWh, report.Cascades[0].Summary.DailyProductionMlnKWh)
	}
	if !approxEqual(report.GrandTotal.YTDProductionMlnKWh, report.Cascades[0].Summary.YTDProductionMlnKWh) {
		t.Errorf("grand total != cascade summary (YTD): %.4f vs %.4f",
			report.GrandTotal.YTDProductionMlnKWh, report.Cascades[0].Summary.YTDProductionMlnKWh)
	}
}

// TestComputeStation_ReserveCalculation verifies that reserve aggregates
// are computed as total - working - repair - modernization for a healthy
// configuration where the sum does not exceed total.
func TestComputeStation_ReserveCalculation(t *testing.T) {
	cascadeID := int64(7)
	cascadeName := "Cascade R"
	orgID := int64(700)

	// total=10, working=4, repair=2, modernization=1 → reserve = 10-4-2-1 = 3.
	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:          orgID,
				OrganizationName:        "Station Reserve",
				CascadeID:               &cascadeID,
				CascadeName:             &cascadeName,
				Date:                    "2026-03-13",
				DailyProductionMlnKWh:   24.0,
				WorkingAggregates:       4,
				RepairAggregates:        2,
				ModernizationAggregates: 1,
				InstalledCapacityMWt:    500.0,
				TotalAggregates:         10,
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}
	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	if st.Current.RepairAggregates != 2 {
		t.Errorf("RepairAggregates: got %d, want 2", st.Current.RepairAggregates)
	}
	if st.Current.ModernizationAggregates != 1 {
		t.Errorf("ModernizationAggregates: got %d, want 1", st.Current.ModernizationAggregates)
	}
	if st.Current.ReserveAggregates != 3 {
		t.Errorf("ReserveAggregates: got %d, want 3 (10-4-2-1)", st.Current.ReserveAggregates)
	}
}

// TestComputeStation_ReserveClampsAtZero verifies that when working+repair+mod
// exceed total (a "sick" data state), reserve is clamped to zero rather than
// going negative — the trigger should normally prevent this, but if config
// shrank or data was inserted before the trigger, we degrade gracefully.
func TestComputeStation_ReserveClampsAtZero(t *testing.T) {
	cascadeID := int64(8)
	cascadeName := "Cascade S"
	orgID := int64(800)

	// total=5, working=4, repair=2, modernization=1 → 4+2+1=7 > 5 → reserve clamped to 0.
	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:          orgID,
				OrganizationName:        "Station Sick",
				CascadeID:               &cascadeID,
				CascadeName:             &cascadeName,
				Date:                    "2026-03-13",
				DailyProductionMlnKWh:   24.0,
				WorkingAggregates:       4,
				RepairAggregates:        2,
				ModernizationAggregates: 1,
				InstalledCapacityMWt:    500.0,
				TotalAggregates:         5,
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}
	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	if st.Current.ReserveAggregates != 0 {
		t.Errorf("ReserveAggregates: got %d, want 0 (clamped from -2)", st.Current.ReserveAggregates)
	}
}

// TestComputeSummary_AggregatesSumAcrossStations verifies that repair,
// modernization, and reserve aggregates are summed correctly across stations
// in a cascade, with reserve recomputed (and clamped) from cascade totals.
func TestComputeSummary_AggregatesSumAcrossStations(t *testing.T) {
	cascadeID := int64(9)
	cascadeName := "Cascade T"

	// Station 1: total=10, working=3, repair=1, mod=1 → reserve = 5
	// Station 2: total=8,  working=4, repair=2, mod=0 → reserve = 2
	// Cascade summary: total=18, working=7, repair=3, mod=1 → reserve = 7
	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:          910,
				OrganizationName:        "Station One",
				CascadeID:               &cascadeID,
				CascadeName:             &cascadeName,
				Date:                    "2026-03-13",
				DailyProductionMlnKWh:   24.0,
				WorkingAggregates:       3,
				RepairAggregates:        1,
				ModernizationAggregates: 1,
				InstalledCapacityMWt:    500.0,
				TotalAggregates:         10,
			},
			{
				OrganizationID:          920,
				OrganizationName:        "Station Two",
				CascadeID:               &cascadeID,
				CascadeName:             &cascadeName,
				Date:                    "2026-03-13",
				DailyProductionMlnKWh:   12.0,
				WorkingAggregates:       4,
				RepairAggregates:        2,
				ModernizationAggregates: 0,
				InstalledCapacityMWt:    200.0,
				TotalAggregates:         8,
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}
	if len(report.Cascades) != 1 {
		t.Fatalf("cascades: got %d, want 1", len(report.Cascades))
	}

	cascade := report.Cascades[0]
	// Per-station reserves.
	if cascade.Stations[0].Current.ReserveAggregates != 5 {
		t.Errorf("station[0].Reserve: got %d, want 5", cascade.Stations[0].Current.ReserveAggregates)
	}
	if cascade.Stations[1].Current.ReserveAggregates != 2 {
		t.Errorf("station[1].Reserve: got %d, want 2", cascade.Stations[1].Current.ReserveAggregates)
	}

	// Cascade summary sums.
	sum := cascade.Summary
	if sum == nil {
		t.Fatal("cascade summary is nil")
	}
	if sum.TotalAggregates != 18 {
		t.Errorf("summary.Total: got %d, want 18", sum.TotalAggregates)
	}
	if sum.WorkingAggregates != 7 {
		t.Errorf("summary.Working: got %d, want 7", sum.WorkingAggregates)
	}
	if sum.RepairAggregates != 3 {
		t.Errorf("summary.Repair: got %d, want 3", sum.RepairAggregates)
	}
	if sum.ModernizationAggregates != 1 {
		t.Errorf("summary.Modernization: got %d, want 1", sum.ModernizationAggregates)
	}
	if sum.ReserveAggregates != 7 {
		t.Errorf("summary.Reserve: got %d, want 7 (18-7-3-1)", sum.ReserveAggregates)
	}

	// Grand total mirrors the single-cascade summary in this test.
	gt := report.GrandTotal
	if gt == nil {
		t.Fatal("grand total is nil")
	}
	if gt.TotalAggregates != 18 {
		t.Errorf("grandTotal.Total: got %d, want 18", gt.TotalAggregates)
	}
	if gt.RepairAggregates != 3 {
		t.Errorf("grandTotal.Repair: got %d, want 3", gt.RepairAggregates)
	}
	if gt.ModernizationAggregates != 1 {
		t.Errorf("grandTotal.Modernization: got %d, want 1", gt.ModernizationAggregates)
	}
	if gt.ReserveAggregates != 7 {
		t.Errorf("grandTotal.Reserve: got %d, want 7", gt.ReserveAggregates)
	}
}

// TestBuildReport_PreviousDaySnapshot verifies that when yesterday data is
// present in the batch, the station's PreviousDay pointer is populated with a
// full snapshot symmetric to CurrentData: daily production, derived power
// (production * 1000 / 24), aggregate counts with reserve recomputed, the
// water/flow scalars, and the derived idle discharge (outflow - ges_flow).
func TestBuildReport_PreviousDaySnapshot(t *testing.T) {
	cascadeID := int64(11)
	cascadeName := "Cascade P"
	orgID := int64(1100)

	// Yesterday: production=18, working=2, repair=1, mod=1, total=5 → reserve=1.
	// Power_yest = 18 * 1000 / 24 = 750.
	// Idle yest = totalOutflow(100) - gesFlow(80) = 20.
	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station P",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 24.0,
				WorkingAggregates:     3,
				InstalledCapacityMWt:  500.0,
				TotalAggregates:       5,
			},
		},
		yesterdayData: []model.RawDailyRow{
			{
				OrganizationID:          orgID,
				OrganizationName:        "Station P",
				CascadeID:               &cascadeID,
				CascadeName:             &cascadeName,
				Date:                    "2026-03-12",
				DailyProductionMlnKWh:   18.0,
				WorkingAggregates:       2,
				RepairAggregates:        1,
				ModernizationAggregates: 1,
				WaterLevelM:             ptr(99.5),
				WaterVolumeMlnM3:        ptr(500.0),
				WaterHeadM:              ptr(40.0),
				ReservoirIncomeM3s:      ptr(120.0),
				TotalOutflowM3s:         ptr(100.0),
				GESFlowM3s:              ptr(80.0),
				InstalledCapacityMWt:    500.0,
				TotalAggregates:         5,
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}
	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	if st.PreviousDay == nil {
		t.Fatal("expected non-nil PreviousDay when yesterday row exists")
	}
	pd := st.PreviousDay
	if !approxEqual(pd.DailyProductionMlnKWh, 18.0) {
		t.Errorf("PreviousDay.DailyProductionMlnKWh: got %.4f, want 18.0", pd.DailyProductionMlnKWh)
	}
	if !approxEqual(pd.PowerMWt, 18.0*1000.0/24.0) {
		t.Errorf("PreviousDay.PowerMWt: got %.4f, want %.4f", pd.PowerMWt, 18.0*1000.0/24.0)
	}
	if pd.WorkingAggregates != 2 {
		t.Errorf("PreviousDay.WorkingAggregates: got %d, want 2", pd.WorkingAggregates)
	}
	if pd.RepairAggregates != 1 {
		t.Errorf("PreviousDay.RepairAggregates: got %d, want 1", pd.RepairAggregates)
	}
	if pd.ModernizationAggregates != 1 {
		t.Errorf("PreviousDay.ModernizationAggregates: got %d, want 1", pd.ModernizationAggregates)
	}
	if pd.ReserveAggregates != 1 {
		t.Errorf("PreviousDay.ReserveAggregates: got %d, want 1 (5-2-1-1)", pd.ReserveAggregates)
	}
	if pd.WaterLevelM == nil || !approxEqual(*pd.WaterLevelM, 99.5) {
		t.Errorf("PreviousDay.WaterLevelM: got %v, want 99.5", pd.WaterLevelM)
	}
	if pd.WaterVolumeMlnM3 == nil || !approxEqual(*pd.WaterVolumeMlnM3, 500.0) {
		t.Errorf("PreviousDay.WaterVolumeMlnM3: got %v, want 500.0", pd.WaterVolumeMlnM3)
	}
	if pd.WaterHeadM == nil || !approxEqual(*pd.WaterHeadM, 40.0) {
		t.Errorf("PreviousDay.WaterHeadM: got %v, want 40.0", pd.WaterHeadM)
	}
	if pd.ReservoirIncomeM3s == nil || !approxEqual(*pd.ReservoirIncomeM3s, 120.0) {
		t.Errorf("PreviousDay.ReservoirIncomeM3s: got %v, want 120.0", pd.ReservoirIncomeM3s)
	}
	if pd.TotalOutflowM3s == nil || !approxEqual(*pd.TotalOutflowM3s, 100.0) {
		t.Errorf("PreviousDay.TotalOutflowM3s: got %v, want 100.0", pd.TotalOutflowM3s)
	}
	if pd.GESFlowM3s == nil || !approxEqual(*pd.GESFlowM3s, 80.0) {
		t.Errorf("PreviousDay.GESFlowM3s: got %v, want 80.0", pd.GESFlowM3s)
	}
	if pd.IdleDischargeM3s == nil || !approxEqual(*pd.IdleDischargeM3s, 20.0) {
		t.Errorf("PreviousDay.IdleDischargeM3s: got %v, want 20.0 (100-80)", pd.IdleDischargeM3s)
	}
}

// TestBuildReport_PreviousDayNilWhenNoYesterdayRow verifies that when no
// yesterday row exists for a station, the PreviousDay pointer stays nil.
func TestBuildReport_PreviousDayNilWhenNoYesterdayRow(t *testing.T) {
	cascadeID := int64(12)
	cascadeName := "Cascade Q"
	orgID := int64(1200)

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Q",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 10.0,
				WorkingAggregates:     1,
				InstalledCapacityMWt:  100.0,
				TotalAggregates:       2,
			},
		},
		yesterdayData: nil,
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}
	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	if st.PreviousDay != nil {
		t.Errorf("expected nil PreviousDay when no yesterday row, got %+v", st.PreviousDay)
	}
}

// TestBuildReport_PreviousDayReserveClamp verifies that when the yesterday row
// has working+repair+modernization > total (a "sick" state), the reserve
// computed for the PreviousDay snapshot is clamped to zero (same rule as
// CurrentData).
func TestBuildReport_PreviousDayReserveClamp(t *testing.T) {
	cascadeID := int64(13)
	cascadeName := "Cascade Z"
	orgID := int64(1300)

	// Yesterday: total=3, working=2, repair=2, mod=1 → 2+2+1=5 > 3 → clamp to 0.
	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Z",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 12.0,
				WorkingAggregates:     1,
				InstalledCapacityMWt:  100.0,
				TotalAggregates:       3,
			},
		},
		yesterdayData: []model.RawDailyRow{
			{
				OrganizationID:          orgID,
				OrganizationName:        "Station Z",
				CascadeID:               &cascadeID,
				CascadeName:             &cascadeName,
				Date:                    "2026-03-12",
				DailyProductionMlnKWh:   10.0,
				WorkingAggregates:       2,
				RepairAggregates:        2,
				ModernizationAggregates: 1,
				InstalledCapacityMWt:    100.0,
				TotalAggregates:         3,
			},
		},
	}

	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}
	if len(report.Cascades) == 0 || len(report.Cascades[0].Stations) == 0 {
		t.Fatal("expected at least one station in report")
	}

	st := report.Cascades[0].Stations[0]
	if st.PreviousDay == nil {
		t.Fatal("expected non-nil PreviousDay when yesterday row exists")
	}
	if st.PreviousDay.ReserveAggregates != 0 {
		t.Errorf("PreviousDay.ReserveAggregates: got %d, want 0 (clamped)", st.PreviousDay.ReserveAggregates)
	}
}

// TestBuildDailyReport_FilterNonExistent verifies that passing an unknown
// cascade ID returns an empty cascades slice and a zeroed GrandTotal.
func TestBuildDailyReport_FilterNonExistent(t *testing.T) {
	repo := twoCascadeRepo()
	loc := mustLoc("Asia/Tashkent")
	svc := NewService(repo, loc, discardLogger())

	unknown := int64(9999)
	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", &unknown)
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	if len(report.Cascades) != 0 {
		t.Fatalf("cascades: got %d, want 0", len(report.Cascades))
	}
	if report.GrandTotal == nil {
		t.Fatal("grand total is nil (expected zeroed SummaryBlock)")
	}
	if !approxEqual(report.GrandTotal.DailyProductionMlnKWh, 0.0) {
		t.Errorf("grand total daily production: got %.4f, want 0.0", report.GrandTotal.DailyProductionMlnKWh)
	}
	if !approxEqual(report.GrandTotal.YTDProductionMlnKWh, 0.0) {
		t.Errorf("grand total YTD: got %.4f, want 0.0", report.GrandTotal.YTDProductionMlnKWh)
	}
}

// --- idle discharge rounding (5 fields × half-away-from-zero, 2 decimal places) ---

// roundTo2 helper unit test: confirms half-away-from-zero behaviour for the
// expected edge cases. Production code uses the same formula in service.go.
func TestRoundTo2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.0, 0.0},
		{1.234, 1.23},
		{1.235, 1.24},  // half-up (>= 0.5 rounds away from zero)
		{1.245, 1.25},  // common pitfall — math.Round rounds half away from zero
		{1.249, 1.25},
		{-1.245, -1.25}, // away-from-zero on negatives too
		{5.16667, 5.17}, // simulate TotalOutflow - GESFlow = 5.5 - 0.3333…
		{0.001, 0.0},
		{99.999, 100.0},
	}
	for _, c := range cases {
		got := roundTo2(c.in)
		if !approxEqual(got, c.want) {
			t.Errorf("roundTo2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestBuildReport_RoundsCurrentIdleDischarge: when TotalOutflow - GESFlow has
// long-tail decimals, the resulting current.idle_discharge_m3s in the JSON
// payload (and Excel input) must come back rounded to 2 dp.
func TestBuildReport_RoundsCurrentIdleDischarge(t *testing.T) {
	cascadeID := int64(1)
	cascadeName := "Cascade A"
	orgID := int64(100)

	totalOutflow := 5.5
	gesFlow := 0.3333 // produces 5.1667 raw → expect 5.17

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        orgID,
				OrganizationName:      "Station Alpha",
				CascadeID:             &cascadeID,
				CascadeName:           &cascadeName,
				Date:                  "2026-03-13",
				DailyProductionMlnKWh: 24.0,
				WorkingAggregates:     3,
				InstalledCapacityMWt:  500.0,
				TotalAggregates:       4,
				HasReservoir:          true,
				TotalOutflowM3s:       &totalOutflow,
				GESFlowM3s:            &gesFlow,
			},
		},
		aggregations: nil,
		plans:        nil,
		discharges:   nil,
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: %v", err)
	}

	st := report.Cascades[0].Stations[0]
	if st.Current.IdleDischargeM3s == nil {
		t.Fatal("current.idle_discharge_m3s is nil")
	}
	if !approxEqual(*st.Current.IdleDischargeM3s, 5.17) {
		t.Errorf("current.idle_discharge_m3s: got %v, want 5.17 (rounded from 5.5 - 0.3333)", *st.Current.IdleDischargeM3s)
	}
}

// TestBuildReport_RoundsPreviousDayIdleDischarge: same long-tail check but
// for the previous-day snapshot (computeDaySnapshot path).
func TestBuildReport_RoundsPreviousDayIdleDischarge(t *testing.T) {
	cascadeID := int64(1)
	cascadeName := "Cascade A"
	orgID := int64(100)

	yTotalOutflow := 7.7
	yGesFlow := 0.6666 // 7.0334 → expect 7.03

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID: orgID, OrganizationName: "Station Alpha",
				CascadeID: &cascadeID, CascadeName: &cascadeName, Date: "2026-03-13",
				DailyProductionMlnKWh: 24.0, WorkingAggregates: 3,
				InstalledCapacityMWt: 500.0, TotalAggregates: 4, HasReservoir: true,
			},
		},
		yesterdayData: []model.RawDailyRow{
			{
				OrganizationID: orgID, OrganizationName: "Station Alpha",
				CascadeID: &cascadeID, CascadeName: &cascadeName, Date: "2026-03-12",
				DailyProductionMlnKWh: 20.0, WorkingAggregates: 3,
				InstalledCapacityMWt: 500.0, TotalAggregates: 4, HasReservoir: true,
				TotalOutflowM3s: &yTotalOutflow, GESFlowM3s: &yGesFlow,
			},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: %v", err)
	}

	st := report.Cascades[0].Stations[0]
	if st.PreviousDay == nil {
		t.Fatal("previous_day is nil")
	}
	if st.PreviousDay.IdleDischargeM3s == nil {
		t.Fatal("previous_day.idle_discharge_m3s is nil")
	}
	if !approxEqual(*st.PreviousDay.IdleDischargeM3s, 7.03) {
		t.Errorf("previous_day.idle_discharge_m3s: got %v, want 7.03", *st.PreviousDay.IdleDischargeM3s)
	}
}

// TestBuildReport_RoundsIdleDischargeEventBlock: stations[].idle_discharge.{flow_rate_m3s,volume_mln_m3}
// and the cascade/grand total summary.idle_discharge_total_m3s must all round.
func TestBuildReport_RoundsIdleDischargeEventBlock(t *testing.T) {
	cascadeID := int64(1)
	cascadeName := "Cascade A"
	orgID := int64(100)
	reason := "test"

	// 0.12345 mln m³ → flow = 0.12345 / 0.0864 = 1.42882… → expect 1.43
	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData: []model.RawDailyRow{
			{
				OrganizationID: orgID, OrganizationName: "Station Alpha",
				CascadeID: &cascadeID, CascadeName: &cascadeName, Date: "2026-03-13",
				DailyProductionMlnKWh: 24.0, WorkingAggregates: 3,
				InstalledCapacityMWt: 500.0, TotalAggregates: 4, HasReservoir: true,
			},
		},
		discharges: []model.IdleDischargeRow{
			{OrganizationID: orgID, VolumeMlnM3: 0.12345, Reason: &reason, IsOngoing: false},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: %v", err)
	}

	st := report.Cascades[0].Stations[0]
	if st.IdleDischarge == nil {
		t.Fatal("station IdleDischarge is nil")
	}
	if !approxEqual(st.IdleDischarge.VolumeMlnM3, 0.12) {
		t.Errorf("station idle_discharge.volume_mln_m3: got %v, want 0.12 (rounded from 0.12345)", st.IdleDischarge.VolumeMlnM3)
	}
	if !approxEqual(st.IdleDischarge.FlowRateM3s, 1.43) {
		t.Errorf("station idle_discharge.flow_rate_m3s: got %v, want 1.43 (rounded from 0.12345/0.0864)", st.IdleDischarge.FlowRateM3s)
	}

	// Cascade summary aggregates flow rates from stations — must also be rounded.
	if !approxEqual(report.Cascades[0].Summary.IdleDischargeM3s, 1.43) {
		t.Errorf("cascade summary idle_discharge_total_m3s: got %v, want 1.43", report.Cascades[0].Summary.IdleDischargeM3s)
	}
	// Grand total sums cascades — single cascade here, so same value.
	if !approxEqual(report.GrandTotal.IdleDischargeM3s, 1.43) {
		t.Errorf("grand total idle_discharge_total_m3s: got %v, want 1.43", report.GrandTotal.IdleDischargeM3s)
	}
}

// === Frozen defaults (sticky carry-forward) tests ===

// frozenStationRow builds a baseline RawDailyRow with HasRowForDate=true.
// Tests override HasRowForDate / nullable fields to drive the cases.
func frozenStationRow(orgID, cascadeID int64, cascadeName, date string) model.RawDailyRow {
	cid := cascadeID
	cn := cascadeName
	return model.RawDailyRow{
		OrganizationID:        orgID,
		OrganizationName:      "Station Frozen",
		CascadeID:             &cid,
		CascadeName:           &cn,
		Date:                  date,
		DailyProductionMlnKWh: 24.0,
		WorkingAggregates:     3,
		InstalledCapacityMWt:  500.0,
		TotalAggregates:       4,
		HasReservoir:          true,
		HasRowForDate:         true,
	}
}

// TestBuildReport_FrozenWaterHeadAppliedWhenRowMissing — нет daily_data на дату
// (HasRowForDate=false), все nullable поля nil, frozen water_head_m=45.0 →
// в отчёте current.water_head_m=45.0.
func TestBuildReport_FrozenWaterHeadAppliedWhenRowMissing(t *testing.T) {
	row := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	row.HasRowForDate = false
	row.DailyProductionMlnKWh = 0 // no row → COALESCE
	row.WorkingAggregates = 0

	repo := &mockRepo{
		todayDate:     "2026-03-13",
		yesterdayDate: "2026-03-12",
		prevYearDate:  "2025-03-13",
		todayData:     []model.RawDailyRow{row},
		frozen: map[int64]map[string]float64{
			100: {"water_head_m": 45.0},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: %v", err)
	}
	st := report.Cascades[0].Stations[0]
	if st.Current.WaterHeadM == nil {
		t.Fatal("current.water_head_m is nil — frozen value not applied")
	}
	if !approxEqual(*st.Current.WaterHeadM, 45.0) {
		t.Errorf("current.water_head_m: got %v, want 45.0 (frozen)", *st.Current.WaterHeadM)
	}
}

// TestBuildReport_FrozenWaterHeadAppliedWhenFieldNull — daily_data строка есть,
// но nullable water_head_m=NULL, frozen=45.0 → в отчёте 45.0.
func TestBuildReport_FrozenWaterHeadAppliedWhenFieldNull(t *testing.T) {
	row := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	row.WaterHeadM = nil // explicitly NULL despite HasRowForDate=true

	repo := &mockRepo{
		todayDate: "2026-03-13", yesterdayDate: "2026-03-12", prevYearDate: "2025-03-13",
		todayData: []model.RawDailyRow{row},
		frozen:    map[int64]map[string]float64{100: {"water_head_m": 45.0}},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, _ := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	st := report.Cascades[0].Stations[0]
	if st.Current.WaterHeadM == nil || !approxEqual(*st.Current.WaterHeadM, 45.0) {
		t.Errorf("current.water_head_m: got %v, want 45.0 (frozen fills NULL)", st.Current.WaterHeadM)
	}
}

// TestBuildReport_FrozenDoesNotOverrideExplicitValue — daily_data.water_head_m=40.0,
// frozen=45.0 → в отчёте 40.0 (явное значение побеждает).
func TestBuildReport_FrozenDoesNotOverrideExplicitValue(t *testing.T) {
	row := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	v := 40.0
	row.WaterHeadM = &v

	repo := &mockRepo{
		todayDate: "2026-03-13", yesterdayDate: "2026-03-12", prevYearDate: "2025-03-13",
		todayData: []model.RawDailyRow{row},
		frozen:    map[int64]map[string]float64{100: {"water_head_m": 45.0}},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, _ := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	st := report.Cascades[0].Stations[0]
	if st.Current.WaterHeadM == nil || !approxEqual(*st.Current.WaterHeadM, 40.0) {
		t.Errorf("current.water_head_m: got %v, want 40.0 (explicit, frozen ignored)", st.Current.WaterHeadM)
	}
}

// TestBuildReport_FrozenWorkingAggregatesAppliedWhenNoRow — !HasRowForDate,
// working_aggregates=0 (COALESCE), frozen=3 → в отчёте 3.
func TestBuildReport_FrozenWorkingAggregatesAppliedWhenNoRow(t *testing.T) {
	row := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	row.HasRowForDate = false
	row.WorkingAggregates = 0
	row.DailyProductionMlnKWh = 0

	repo := &mockRepo{
		todayDate: "2026-03-13", yesterdayDate: "2026-03-12", prevYearDate: "2025-03-13",
		todayData: []model.RawDailyRow{row},
		frozen:    map[int64]map[string]float64{100: {"working_aggregates": 3.0}},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, _ := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	st := report.Cascades[0].Stations[0]
	if st.Current.WorkingAggregates != 3 {
		t.Errorf("current.working_aggregates: got %d, want 3 (frozen, no row)", st.Current.WorkingAggregates)
	}
}

// TestBuildReport_FrozenWorkingAggregatesNotAppliedWhenRowExists — HasRowForDate=true,
// working_aggregates=0 в БД, frozen=3 → в отчёте 0 (явный 0, frozen НЕ применяется).
// Регрессия для §2.7: для NOT NULL полей frozen применяется только когда строки нет.
func TestBuildReport_FrozenWorkingAggregatesNotAppliedWhenRowExists(t *testing.T) {
	row := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	row.WorkingAggregates = 0 // explicit 0 with HasRowForDate=true

	repo := &mockRepo{
		todayDate: "2026-03-13", yesterdayDate: "2026-03-12", prevYearDate: "2025-03-13",
		todayData: []model.RawDailyRow{row},
		frozen:    map[int64]map[string]float64{100: {"working_aggregates": 3.0}},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, _ := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	st := report.Cascades[0].Stations[0]
	if st.Current.WorkingAggregates != 0 {
		t.Errorf("current.working_aggregates: got %d, want 0 (explicit 0; frozen MUST be ignored when HasRowForDate=true)", st.Current.WorkingAggregates)
	}
}

// TestBuildReport_FrozenAppliedToPreviousDay — previous_day snapshot тоже видит frozen.
// Без этого diff'ы будут сравнивать сегодня=frozen vs вчера=nil/0 — мусор.
func TestBuildReport_FrozenAppliedToPreviousDay(t *testing.T) {
	today := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	yesterdayRow := frozenStationRow(100, 1, "Cascade A", "2026-03-12")
	yesterdayRow.HasRowForDate = false
	yesterdayRow.WaterHeadM = nil

	repo := &mockRepo{
		todayDate: "2026-03-13", yesterdayDate: "2026-03-12", prevYearDate: "2025-03-13",
		todayData:     []model.RawDailyRow{today},
		yesterdayData: []model.RawDailyRow{yesterdayRow},
		frozen:        map[int64]map[string]float64{100: {"water_head_m": 45.0}},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, _ := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	st := report.Cascades[0].Stations[0]
	if st.PreviousDay == nil {
		t.Fatal("previous_day is nil")
	}
	if st.PreviousDay.WaterHeadM == nil || !approxEqual(*st.PreviousDay.WaterHeadM, 45.0) {
		t.Errorf("previous_day.water_head_m: got %v, want 45.0 (frozen on yesterday)", st.PreviousDay.WaterHeadM)
	}
}

// TestBuildReport_NoFrozenForField_Unchanged — org has frozen entry for water_head_m
// but NOT for water_level_m → water_level_m stays nil (не магически 0).
func TestBuildReport_NoFrozenForField_Unchanged(t *testing.T) {
	row := frozenStationRow(100, 1, "Cascade A", "2026-03-13")
	row.WaterHeadM = nil
	row.WaterLevelM = nil

	repo := &mockRepo{
		todayDate: "2026-03-13", yesterdayDate: "2026-03-12", prevYearDate: "2025-03-13",
		todayData: []model.RawDailyRow{row},
		frozen:    map[int64]map[string]float64{100: {"water_head_m": 45.0}},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, _ := svc.BuildDailyReport(context.Background(), "2026-03-13", nil)
	st := report.Cascades[0].Stations[0]
	if st.Current.WaterLevelM != nil {
		t.Errorf("current.water_level_m: got %v, want nil (no frozen for this field)", st.Current.WaterLevelM)
	}
	if st.Current.WaterHeadM == nil || !approxEqual(*st.Current.WaterHeadM, 45.0) {
		t.Errorf("current.water_head_m: got %v, want 45.0 (frozen)", st.Current.WaterHeadM)
	}
}
