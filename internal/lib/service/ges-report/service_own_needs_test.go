package gesreportservice

import (
	"context"
	"testing"

	model "srmt-admin/internal/lib/model/ges-report"
)

// TestBuildOwnNeedsReport_GroupsByCascade verifies the projection groups
// stations by cascade and that totals sum across the cascade's stations.
func TestBuildOwnNeedsReport_GroupsByCascade(t *testing.T) {
	cid := int64(1)
	cname := "Cascade A"

	repo := &mockRepo{
		todayDate:     "2026-04-27",
		yesterdayDate: "2026-04-26",
		prevYearDate:  "2025-04-27",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        100,
				OrganizationName:      "Station One",
				CascadeID:             &cid,
				CascadeName:           &cname,
				Date:                  "2026-04-27",
				DailyProductionMlnKWh: 2.5,
				WorkingAggregates:     2,
				InstalledCapacityMWt:  10.0,
				TotalAggregates:       3,
				HasReservoir:          true,
				HasRowForDate:         true,
				OwnConsumptionKWh:     ptr(500.0),
			},
			{
				OrganizationID:        101,
				OrganizationName:      "Station Two",
				CascadeID:             &cid,
				CascadeName:           &cname,
				Date:                  "2026-04-27",
				DailyProductionMlnKWh: 1.0,
				WorkingAggregates:     1,
				InstalledCapacityMWt:  5.0,
				TotalAggregates:       2,
				HasReservoir:          true,
				HasRowForDate:         true,
				OwnConsumptionKWh:     ptr(200.0),
			},
		},
		aggregations: []model.ProductionAggregation{
			{OrganizationID: 100, MTD: 50, YTD: 200, MTDOwnConsumptionKWh: 12000, YTDOwnConsumptionKWh: 60000},
			{OrganizationID: 101, MTD: 20, YTD: 80, MTDOwnConsumptionKWh: 5000, YTDOwnConsumptionKWh: 25000},
		},
		plans: []model.PlanRow{
			{OrganizationID: 100, Year: 2026, Month: 4, PlanMlnKWh: 6.5},
			{OrganizationID: 101, Year: 2026, Month: 4, PlanMlnKWh: 2.5},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	rep, err := svc.BuildOwnNeedsReport(context.Background(), "2026-04-27")
	if err != nil {
		t.Fatalf("BuildOwnNeedsReport: %v", err)
	}

	if rep.Date != "2026-04-27" {
		t.Errorf("Date: got %q, want 2026-04-27", rep.Date)
	}
	if len(rep.Cascades) != 1 {
		t.Fatalf("Cascades: got %d, want 1", len(rep.Cascades))
	}
	c := rep.Cascades[0]
	if c.CascadeID != 1 || c.CascadeName != "Cascade A" {
		t.Errorf("Cascade meta: got id=%d name=%q, want 1/Cascade A", c.CascadeID, c.CascadeName)
	}
	if len(c.Stations) != 2 {
		t.Fatalf("Stations: got %d, want 2", len(c.Stations))
	}

	// Per-station spot checks.
	if c.Stations[0].OrganizationID != 100 || c.Stations[0].Name != "Station One" {
		t.Errorf("Station[0]: got id=%d name=%q", c.Stations[0].OrganizationID, c.Stations[0].Name)
	}
	if c.Stations[0].OwnConsumptionKWh == nil || *c.Stations[0].OwnConsumptionKWh != 500.0 {
		t.Errorf("Station[0].OwnConsumptionKWh: got %v, want 500", c.Stations[0].OwnConsumptionKWh)
	}
	if !approxEqual(c.Stations[0].MTDOwnConsumptionKWh, 12000) {
		t.Errorf("Station[0].MTDOwn: got %v, want 12000", c.Stations[0].MTDOwnConsumptionKWh)
	}
	if !approxEqual(c.Stations[0].MonthlyPlanMlnKWh, 6.5) {
		t.Errorf("Station[0].MonthlyPlan: got %v, want 6.5", c.Stations[0].MonthlyPlanMlnKWh)
	}

	// Cascade totals = sum across stations.
	if !approxEqual(c.Totals.OwnConsumptionKWh, 700.0) {
		t.Errorf("Cascade.Totals.OwnConsumptionKWh: got %v, want 700", c.Totals.OwnConsumptionKWh)
	}
	if !approxEqual(c.Totals.InstalledCapacityMWt, 15.0) {
		t.Errorf("Cascade.Totals.InstalledCapacityMWt: got %v, want 15", c.Totals.InstalledCapacityMWt)
	}
	if !approxEqual(c.Totals.MTDOwnConsumptionKWh, 17000) {
		t.Errorf("Cascade.Totals.MTDOwn: got %v, want 17000", c.Totals.MTDOwnConsumptionKWh)
	}
	if !approxEqual(c.Totals.MonthlyPlanMlnKWh, 9.0) {
		t.Errorf("Cascade.Totals.MonthlyPlan: got %v, want 9.0", c.Totals.MonthlyPlanMlnKWh)
	}

	// Grand total = single cascade.
	if !approxEqual(rep.GrandTotal.OwnConsumptionKWh, 700.0) {
		t.Errorf("GrandTotal.OwnConsumptionKWh: got %v, want 700", rep.GrandTotal.OwnConsumptionKWh)
	}
	if !approxEqual(rep.GrandTotal.YTDOwnConsumptionKWh, 85000) {
		t.Errorf("GrandTotal.YTDOwn: got %v, want 85000", rep.GrandTotal.YTDOwnConsumptionKWh)
	}
}

// TestBuildOwnNeedsReport_DeltaPositiveAndNegative verifies own_consumption
// delta is computed as today - yesterday, both signs.
func TestBuildOwnNeedsReport_DeltaPositiveAndNegative(t *testing.T) {
	cid := int64(1)
	cname := "Cascade A"

	repo := &mockRepo{
		todayDate:     "2026-04-27",
		yesterdayDate: "2026-04-26",
		prevYearDate:  "2025-04-27",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        100, OrganizationName: "Up",
				CascadeID: &cid, CascadeName: &cname,
				Date: "2026-04-27", DailyProductionMlnKWh: 3.0,
				InstalledCapacityMWt: 10.0, TotalAggregates: 3,
				HasRowForDate: true, OwnConsumptionKWh: ptr(100.0),
			},
			{
				OrganizationID:        101, OrganizationName: "Down",
				CascadeID: &cid, CascadeName: &cname,
				Date: "2026-04-27", DailyProductionMlnKWh: 1.0,
				InstalledCapacityMWt: 5.0, TotalAggregates: 2,
				HasRowForDate: true, OwnConsumptionKWh: ptr(40.0),
			},
		},
		yesterdayData: []model.RawDailyRow{
			{
				OrganizationID: 100, OrganizationName: "Up",
				CascadeID: &cid, CascadeName: &cname,
				Date: "2026-04-26", DailyProductionMlnKWh: 2.5,
				InstalledCapacityMWt: 10.0, TotalAggregates: 3,
				HasRowForDate: true, OwnConsumptionKWh: ptr(80.0),
			},
			{
				OrganizationID: 101, OrganizationName: "Down",
				CascadeID: &cid, CascadeName: &cname,
				Date: "2026-04-26", DailyProductionMlnKWh: 1.5,
				InstalledCapacityMWt: 5.0, TotalAggregates: 2,
				HasRowForDate: true, OwnConsumptionKWh: ptr(60.0),
			},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	rep, err := svc.BuildOwnNeedsReport(context.Background(), "2026-04-27")
	if err != nil {
		t.Fatalf("BuildOwnNeedsReport: %v", err)
	}

	st0 := rep.Cascades[0].Stations[0]
	st1 := rep.Cascades[0].Stations[1]

	if st0.OwnConsumptionDelta == nil || !approxEqual(*st0.OwnConsumptionDelta, 20.0) {
		t.Errorf("Up.OwnConsumptionDelta: got %v, want 20", st0.OwnConsumptionDelta)
	}
	if st1.OwnConsumptionDelta == nil || !approxEqual(*st1.OwnConsumptionDelta, -20.0) {
		t.Errorf("Down.OwnConsumptionDelta: got %v, want -20", st1.OwnConsumptionDelta)
	}
	if st0.DailyProductionDelta == nil || !approxEqual(*st0.DailyProductionDelta, 0.5) {
		t.Errorf("Up.ProductionDelta: got %v, want 0.5", st0.DailyProductionDelta)
	}
	if st1.DailyProductionDelta == nil || !approxEqual(*st1.DailyProductionDelta, -0.5) {
		t.Errorf("Down.ProductionDelta: got %v, want -0.5", st1.DailyProductionDelta)
	}
}

// TestBuildOwnNeedsReport_NoYesterday_DeltaNil verifies that when no
// yesterday row exists, the deltas are nil (UI renders these as empty/0).
func TestBuildOwnNeedsReport_NoYesterday_DeltaNil(t *testing.T) {
	cid := int64(1)
	cname := "Cascade A"

	repo := &mockRepo{
		todayDate:     "2026-04-27",
		yesterdayDate: "2026-04-26",
		prevYearDate:  "2025-04-27",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        100, OrganizationName: "Lonely",
				CascadeID: &cid, CascadeName: &cname,
				Date: "2026-04-27", DailyProductionMlnKWh: 3.0,
				InstalledCapacityMWt: 10.0, TotalAggregates: 3,
				HasRowForDate: true, OwnConsumptionKWh: ptr(100.0),
			},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	rep, err := svc.BuildOwnNeedsReport(context.Background(), "2026-04-27")
	if err != nil {
		t.Fatalf("BuildOwnNeedsReport: %v", err)
	}

	st := rep.Cascades[0].Stations[0]
	if st.OwnConsumptionDelta != nil {
		t.Errorf("OwnConsumptionDelta: got %v, want nil", *st.OwnConsumptionDelta)
	}
	if st.DailyProductionDelta != nil {
		t.Errorf("DailyProductionDelta: got %v, want nil", *st.DailyProductionDelta)
	}
}

// TestBuildOwnNeedsReport_NilOwnConsumption verifies that a station with
// no own_consumption_kwh recorded surfaces as nil in the report.
func TestBuildOwnNeedsReport_NilOwnConsumption(t *testing.T) {
	cid := int64(1)
	cname := "Cascade A"

	repo := &mockRepo{
		todayDate:     "2026-04-27",
		yesterdayDate: "2026-04-26",
		prevYearDate:  "2025-04-27",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        100, OrganizationName: "NoOwn",
				CascadeID: &cid, CascadeName: &cname,
				Date: "2026-04-27", DailyProductionMlnKWh: 1.0,
				InstalledCapacityMWt: 5.0, TotalAggregates: 1,
				HasRowForDate: true,
				// OwnConsumptionKWh nil
			},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	rep, err := svc.BuildOwnNeedsReport(context.Background(), "2026-04-27")
	if err != nil {
		t.Fatalf("BuildOwnNeedsReport: %v", err)
	}

	st := rep.Cascades[0].Stations[0]
	if st.OwnConsumptionKWh != nil {
		t.Errorf("OwnConsumptionKWh: got %v, want nil", *st.OwnConsumptionKWh)
	}
	// Cascade total: a nil station contributes 0.
	if !approxEqual(rep.Cascades[0].Totals.OwnConsumptionKWh, 0.0) {
		t.Errorf("Cascade.Totals.OwnConsumptionKWh: got %v, want 0", rep.Cascades[0].Totals.OwnConsumptionKWh)
	}
}

// TestBuildOwnNeedsReport_GrandTotalSumsAcrossCascades verifies the
// grand total aggregates across multiple cascades.
func TestBuildOwnNeedsReport_GrandTotalSumsAcrossCascades(t *testing.T) {
	cid1 := int64(1)
	cn1 := "Cascade A"
	cid2 := int64(2)
	cn2 := "Cascade B"

	repo := &mockRepo{
		todayDate:     "2026-04-27",
		yesterdayDate: "2026-04-26",
		prevYearDate:  "2025-04-27",
		todayData: []model.RawDailyRow{
			{
				OrganizationID:        100, OrganizationName: "A1",
				CascadeID: &cid1, CascadeName: &cn1,
				Date: "2026-04-27", DailyProductionMlnKWh: 2.0,
				InstalledCapacityMWt: 10.0, TotalAggregates: 2,
				HasRowForDate: true, OwnConsumptionKWh: ptr(100.0),
			},
			{
				OrganizationID:        200, OrganizationName: "B1",
				CascadeID: &cid2, CascadeName: &cn2,
				Date: "2026-04-27", DailyProductionMlnKWh: 1.0,
				InstalledCapacityMWt: 5.0, TotalAggregates: 1,
				HasRowForDate: true, OwnConsumptionKWh: ptr(50.0),
			},
		},
		aggregations: []model.ProductionAggregation{
			{OrganizationID: 100, MTD: 30, YTD: 100, MTDOwnConsumptionKWh: 1500, YTDOwnConsumptionKWh: 7500},
			{OrganizationID: 200, MTD: 15, YTD: 50, MTDOwnConsumptionKWh: 800, YTDOwnConsumptionKWh: 3500},
		},
	}

	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	rep, err := svc.BuildOwnNeedsReport(context.Background(), "2026-04-27")
	if err != nil {
		t.Fatalf("BuildOwnNeedsReport: %v", err)
	}

	if len(rep.Cascades) != 2 {
		t.Fatalf("Cascades: got %d, want 2", len(rep.Cascades))
	}
	if !approxEqual(rep.GrandTotal.OwnConsumptionKWh, 150.0) {
		t.Errorf("GrandTotal.OwnConsumptionKWh: got %v, want 150", rep.GrandTotal.OwnConsumptionKWh)
	}
	if !approxEqual(rep.GrandTotal.MTDOwnConsumptionKWh, 2300) {
		t.Errorf("GrandTotal.MTDOwn: got %v, want 2300", rep.GrandTotal.MTDOwnConsumptionKWh)
	}
	if !approxEqual(rep.GrandTotal.YTDProductionMlnKWh, 150) {
		t.Errorf("GrandTotal.YTDProd: got %v, want 150", rep.GrandTotal.YTDProductionMlnKWh)
	}
	if !approxEqual(rep.GrandTotal.InstalledCapacityMWt, 15.0) {
		t.Errorf("GrandTotal.InstalledCapacity: got %v, want 15", rep.GrandTotal.InstalledCapacityMWt)
	}
}
