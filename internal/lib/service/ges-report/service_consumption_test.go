package gesreportservice

import (
	"context"
	"errors"
	"testing"

	model "srmt-admin/internal/lib/model/ges-report"
)

// makeStation builds a minimal RawDailyRow for the consumption-validation
// tests. Caller controls outflow/flow/consumption; everything else is
// constants that don't affect the consumption check.
func makeStation(orgID int64, name string, cascadeID int64, cascadeName string,
	totalOutflow, gesFlow, consumption *float64,
) model.RawDailyRow {
	return model.RawDailyRow{
		OrganizationID:        orgID,
		OrganizationName:      name,
		CascadeID:             &cascadeID,
		CascadeName:           &cascadeName,
		Date:                  "2026-04-22",
		DailyProductionMlnKWh: 24.0,
		TotalOutflowM3s:       totalOutflow,
		GESFlowM3s:            gesFlow,
		ConsumptionM3s:        consumption,
		InstalledCapacityMWt:  500,
		TotalAggregates:       4,
		WorkingAggregates:     1,
	}
}

// Happy path: consumption fits inside idle (outflow=10, gesFlow=5, idle=5,
// consumption=2). The report renders, and CurrentData.IdleDischargeM3s
// reflects the adjusted value (idle - consumption = 3).
func TestBuildDailyReport_ConsumptionAdjustsIdle(t *testing.T) {
	repo := &mockRepo{
		todayDate:     "2026-04-22",
		yesterdayDate: "2026-04-21",
		prevYearDate:  "2025-04-22",
		todayData: []model.RawDailyRow{
			makeStation(100, "Station A", 1, "Cascade X",
				ptr(10.0), ptr(5.0), ptr(2.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())

	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: unexpected err: %v", err)
	}
	if report == nil || len(report.Cascades) != 1 || len(report.Cascades[0].Stations) != 1 {
		t.Fatalf("expected 1 cascade with 1 station, got: %+v", report)
	}
	idle := report.Cascades[0].Stations[0].Current.IdleDischargeM3s
	if idle == nil {
		t.Fatal("CurrentData.IdleDischargeM3s must be set when outflow+ges_flow are present")
	}
	if !approxEqual(*idle, 3.0) {
		t.Errorf("adjusted idle: want 3.0 (10 - 5 - 2), got %v", *idle)
	}
}

// Consumption equal to idle gives 0 (boundary, allowed). Plan rule:
// "consumption <= idle" with strict-inequality only for the violation.
func TestBuildDailyReport_ConsumptionEqualsIdle_OK(t *testing.T) {
	repo := &mockRepo{
		todayDate: "2026-04-22",
		todayData: []model.RawDailyRow{
			makeStation(100, "Station A", 1, "Cascade X",
				ptr(10.0), ptr(5.0), ptr(5.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: unexpected err: %v", err)
	}
	idle := report.Cascades[0].Stations[0].Current.IdleDischargeM3s
	if idle == nil || !approxEqual(*idle, 0.0) {
		t.Errorf("idle: want 0.0 at boundary, got %v", idle)
	}
}

// Consumption > idle for a single station fails the report build with a
// typed *ReportValidationError (code=report.consumption_exceeds_idle, one
// violation describing the offending station).
func TestBuildDailyReport_ConsumptionExceedsIdle_Error(t *testing.T) {
	repo := &mockRepo{
		todayDate: "2026-04-22",
		todayData: []model.RawDailyRow{
			// idle = 10 - 8 = 2; consumption = 5 → violation
			makeStation(16, "ГЭС-1", 1, "Каскад A",
				ptr(10.0), ptr(8.0), ptr(5.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err == nil {
		t.Fatalf("expected ReportValidationError, got nil; report=%+v", report)
	}
	var rve *ReportValidationError
	if !errors.As(err, &rve) {
		t.Fatalf("err type: want *ReportValidationError, got %T (%v)", err, err)
	}
	if rve.Code != "report.consumption_exceeds_idle" {
		t.Errorf("Code: want %q, got %q", "report.consumption_exceeds_idle", rve.Code)
	}
	if len(rve.Violations) != 1 {
		t.Fatalf("Violations: want 1, got %d (%+v)", len(rve.Violations), rve.Violations)
	}
	v := rve.Violations[0]
	if v.OrganizationID != 16 {
		t.Errorf("Violations[0].OrganizationID: want 16, got %d", v.OrganizationID)
	}
	if v.OrganizationName != "ГЭС-1" {
		t.Errorf("Violations[0].OrganizationName: want %q, got %q", "ГЭС-1", v.OrganizationName)
	}
	if !approxEqual(v.IdleM3s, 2.0) {
		t.Errorf("Violations[0].IdleM3s: want 2.0, got %v", v.IdleM3s)
	}
	if !approxEqual(v.ConsumptionM3s, 5.0) {
		t.Errorf("Violations[0].ConsumptionM3s: want 5.0, got %v", v.ConsumptionM3s)
	}
	if v.Date != "2026-04-22" {
		t.Errorf("Violations[0].Date: want %q, got %q", "2026-04-22", v.Date)
	}
}

// Multiple violations across stations are ALL collected (no early-exit) so
// the user sees the full list at once instead of fixing them one-by-one.
func TestBuildDailyReport_MultipleViolations_AllReported(t *testing.T) {
	repo := &mockRepo{
		todayDate: "2026-04-22",
		todayData: []model.RawDailyRow{
			// Station 16: idle=2, consumption=5 → violation
			makeStation(16, "ГЭС-1", 1, "Каскад A", ptr(10.0), ptr(8.0), ptr(5.0)),
			// Station 17: idle=1, consumption=3 → violation
			makeStation(17, "ГЭС-2", 1, "Каскад A", ptr(4.0), ptr(3.0), ptr(3.0)),
			// Station 18: idle=10, consumption=2 → OK (no violation)
			makeStation(18, "ГЭС-3", 1, "Каскад A", ptr(20.0), ptr(10.0), ptr(2.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	_, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	var rve *ReportValidationError
	if !errors.As(err, &rve) {
		t.Fatalf("err type: want *ReportValidationError, got %T", err)
	}
	if len(rve.Violations) != 2 {
		t.Fatalf("Violations: want 2, got %d (%+v)", len(rve.Violations), rve.Violations)
	}
	// Order is the same as input today rows.
	if rve.Violations[0].OrganizationID != 16 || rve.Violations[1].OrganizationID != 17 {
		t.Errorf("Violations order/IDs: %+v", rve.Violations)
	}
}

// When consumption is set but outflow OR ges_flow is nil, idle cannot be
// computed → violation check is skipped (matches existing computeDaySnapshot
// nil-handling semantics). Report renders without error; idle stays nil.
func TestBuildDailyReport_ConsumptionWithoutIdleInputs_NoError(t *testing.T) {
	repo := &mockRepo{
		todayDate: "2026-04-22",
		todayData: []model.RawDailyRow{
			// outflow nil → idle uncomputable
			makeStation(100, "Station A", 1, "Cascade X", nil, ptr(5.0), ptr(2.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport must not fail when idle is uncomputable: %v", err)
	}
	idle := report.Cascades[0].Stations[0].Current.IdleDischargeM3s
	if idle != nil {
		t.Errorf("IdleDischargeM3s: want nil when outflow nil, got %v", *idle)
	}
}

// Consumption nil + idle present → no adjustment, no error. Adjustment only
// applies when consumption is explicitly set.
func TestBuildDailyReport_ConsumptionNil_IdleUnchanged(t *testing.T) {
	repo := &mockRepo{
		todayDate: "2026-04-22",
		todayData: []model.RawDailyRow{
			makeStation(100, "Station A", 1, "Cascade X",
				ptr(10.0), ptr(5.0), nil),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: %v", err)
	}
	idle := report.Cascades[0].Stations[0].Current.IdleDischargeM3s
	if idle == nil || !approxEqual(*idle, 5.0) {
		t.Errorf("idle: want 5.0 (no consumption adjustment), got %v", idle)
	}
}

// Historical violation in yesterday's data must NOT surface as negative idle
// in PreviousDayData. validateConsumptionAgainstIdle only runs over today's
// rows; yesterday's idle is clamped to >= 0 in computeDaySnapshot. This pins
// the contract that a buggy historical row never produces a negative number
// in the rendered report.
func TestBuildDailyReport_YesterdayConsumptionExceedsIdle_Clamped(t *testing.T) {
	repo := &mockRepo{
		todayDate:     "2026-04-22",
		yesterdayDate: "2026-04-21",
		// Today: clean (idle=5, consumption=2 → adjusted=3)
		todayData: []model.RawDailyRow{
			makeStation(100, "Station A", 1, "Cascade X",
				ptr(10.0), ptr(5.0), ptr(2.0)),
		},
		// Yesterday: consumption (10) > idle (2) — historically broken row
		yesterdayData: []model.RawDailyRow{
			makeStation(100, "Station A", 1, "Cascade X",
				ptr(10.0), ptr(8.0), ptr(10.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport must not fail on yesterday's historical violation: %v", err)
	}
	st := report.Cascades[0].Stations[0]
	if st.PreviousDay == nil {
		t.Fatal("PreviousDay must be set when yesterday's row exists")
	}
	if st.PreviousDay.IdleDischargeM3s == nil {
		t.Fatal("PreviousDay.IdleDischargeM3s must be set when outflow+ges_flow present")
	}
	if *st.PreviousDay.IdleDischargeM3s < 0 {
		t.Errorf("PreviousDay.IdleDischargeM3s must be clamped to >= 0, got %v", *st.PreviousDay.IdleDischargeM3s)
	}
	if !approxEqual(*st.PreviousDay.IdleDischargeM3s, 0.0) {
		t.Errorf("PreviousDay.IdleDischargeM3s: want 0.0 (clamped from -8), got %v", *st.PreviousDay.IdleDischargeM3s)
	}
}

// CurrentData carries ConsumptionM3s verbatim so the frontend can display
// the value separately from the (already-adjusted) idle.
func TestBuildDailyReport_ConsumptionPropagatedToSnapshot(t *testing.T) {
	repo := &mockRepo{
		todayDate: "2026-04-22",
		todayData: []model.RawDailyRow{
			makeStation(100, "Station A", 1, "Cascade X",
				ptr(10.0), ptr(5.0), ptr(2.0)),
		},
	}
	svc := NewService(repo, mustLoc("Asia/Tashkent"), discardLogger())
	report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
	if err != nil {
		t.Fatalf("BuildDailyReport: %v", err)
	}
	cons := report.Cascades[0].Stations[0].Current.ConsumptionM3s
	if cons == nil || !approxEqual(*cons, 2.0) {
		t.Errorf("CurrentData.ConsumptionM3s: want 2.0, got %v", cons)
	}
}
