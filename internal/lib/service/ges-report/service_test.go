package gesreportservice

import (
	"context"
	"math"
	"testing"
	"time"

	model "srmt-admin/internal/lib/model/ges-report"
)

// mockRepo implements Repository with date-based dispatch for GetGESDailyDataBatch.
type mockRepo struct {
	todayData     []model.RawDailyRow
	yesterdayData []model.RawDailyRow
	prevYearData  []model.RawDailyRow
	todayDate     string
	yesterdayDate string
	prevYearDate  string
	aggregations  []model.ProductionAggregation
	plans         []model.PlanRow
	discharges    []model.IdleDischargeRow
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

// ptr returns a pointer to the given float64.
func ptr(v float64) *float64 { return &v }

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
	svc := NewService(repo, loc)

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13")
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
	svc := NewService(repo, loc)

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13")
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
	svc := NewService(repo, loc)

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13")
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

	if !approxEqual(st.IdleDischarge.FlowRateM3s, 120.5) {
		t.Errorf("flow rate: got %.4f, want 120.5", st.IdleDischarge.FlowRateM3s)
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
	if !approxEqual(gt.IdleDischargeM3s, 120.5) {
		t.Errorf("grand total idle discharge: got %.4f, want 120.5", gt.IdleDischargeM3s)
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
	svc := NewService(repo, loc)

	report, err := svc.BuildDailyReport(context.Background(), "2026-03-13")
	if err != nil {
		t.Fatalf("BuildDailyReport returned error: %v", err)
	}

	st := report.Cascades[0].Stations[0]
	if st.IdleDischarge == nil {
		t.Fatal("IdleDischarge is nil")
	}

	// Flow rates summed: 50 + 30 = 80
	if !approxEqual(st.IdleDischarge.FlowRateM3s, 80.0) {
		t.Errorf("summed flow rate: got %.4f, want 80.0", st.IdleDischarge.FlowRateM3s)
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
