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
	if !approxEqual(st.IdleDischarge.VolumeMlnM3, 0.432) {
		t.Errorf("volume: got %.4f, want 0.432", st.IdleDischarge.VolumeMlnM3)
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

	// Volumes summed: 0.1 + 0.2 = 0.3; FlowRate = 0.3 / 0.0864 ≈ 3.4722
	wantFlow := 0.3 / 0.0864
	if !approxEqual(st.IdleDischarge.FlowRateM3s, wantFlow) {
		t.Errorf("derived flow rate: got %.4f, want %.4f", st.IdleDischarge.FlowRateM3s, wantFlow)
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
