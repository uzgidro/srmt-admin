package repo

import (
	"strings"
	"testing"
	"time"

	dvmodel "srmt-admin/internal/lib/model/duty-violations"
)

// Structural tests for the duty-violations SQL. Behaviour against a real
// DB lives in the dev smoke loop; these lock in the shape so accidental
// edits surface in CI.

func TestBuildDutyViolationsListQuery_NoFilters(t *testing.T) {
	q, args := buildDutyViolationsListQuery(dvmodel.ListFilter{})

	if len(args) != 0 {
		t.Errorf("no-filter query must not bind args, got %d", len(args))
	}
	if strings.Contains(q, "WHERE") {
		t.Errorf("no-filter query must not have WHERE clause:\n%s", q)
	}
	if !strings.Contains(q, "ORDER BY dv.start_time DESC") {
		t.Errorf("query must order by start_time DESC:\n%s", q)
	}
}

func TestBuildDutyViolationsListQuery_FiltersByOrg(t *testing.T) {
	orgID := int64(42)
	q, args := buildDutyViolationsListQuery(dvmodel.ListFilter{OrganizationID: &orgID})

	if !strings.Contains(q, "dv.organization_id = $1") {
		t.Errorf("org filter missing or wrong placeholder:\n%s", q)
	}
	if len(args) != 1 || args[0] != orgID {
		t.Errorf("org filter args wrong: %v", args)
	}
}

// A non-nil Day expands into a half-open `[Day, Day+24h)` window: both
// sides bound to start_time, the upper bound strict `<` so a record on
// the cutoff belongs to the NEXT op-day. Two args, one each.
func TestBuildDutyViolationsListQuery_FiltersByDay(t *testing.T) {
	day := time.Date(2026, 6, 8, 5, 0, 0, 0, time.UTC)
	q, args := buildDutyViolationsListQuery(dvmodel.ListFilter{Day: &day})

	if !strings.Contains(q, "dv.start_time >= $1") {
		t.Errorf("lower bound missing: %s", q)
	}
	if !strings.Contains(q, "dv.start_time < $2") {
		t.Errorf("upper bound must use half-open `< $2`, got: %s", q)
	}
	if strings.Contains(q, "dv.start_time <= $") {
		t.Errorf("legacy inclusive `<=` filter must not appear: %s", q)
	}
	if len(args) != 2 {
		t.Errorf("want 2 args (start + end), got %d", len(args))
	}
	if got := args[0].(time.Time); !got.Equal(day) {
		t.Errorf("lower bound arg: want %v, got %v", day, got)
	}
	if got := args[1].(time.Time); !got.Equal(day.Add(24 * time.Hour)) {
		t.Errorf("upper bound arg: want %v, got %v", day.Add(24*time.Hour), got)
	}
}

func TestBuildDutyViolationsListQuery_CombinesFiltersWithAnd(t *testing.T) {
	orgID := int64(42)
	day := time.Date(2026, 6, 8, 5, 0, 0, 0, time.UTC)
	q, _ := buildDutyViolationsListQuery(dvmodel.ListFilter{
		OrganizationID: &orgID, Day: &day,
	})

	if !strings.Contains(q, " AND ") {
		t.Errorf("multiple filters must be joined with AND:\n%s", q)
	}
}

// The single-row query must JOIN organizations so the response includes
// org name without an extra round-trip — frontends render the org label
// inline with the violation row.
func TestSelectDutyViolationFields_JoinsOrgName(t *testing.T) {
	combined := selectDutyViolationFields + fromDutyViolationJoins
	if !strings.Contains(combined, "LEFT JOIN organizations") {
		t.Errorf("missing organizations JOIN:\n%s", combined)
	}
	if !strings.Contains(combined, "organization_name") {
		t.Errorf("missing organization_name column:\n%s", combined)
	}
}

// The single-row SELECT must list exactly the columns the row scanner
// reads — 10 fields. If the constant drifts (column added/removed without
// updating scanDutyViolationRow), this test surfaces it before runtime.
// Each expected column appears as a token in the SELECT list.
func TestSelectDutyViolationFields_AllScannerColumnsPresent(t *testing.T) {
	must := []string{
		"dv.id",
		"dv.organization_id",
		"organization_name", // aliased from COALESCE(o.name, '')
		"dv.start_time",
		"dv.end_time",
		"dv.duty_officer_name",
		"dv.reason",
		"dv.created_at",
		"dv.created_by_user_id",
		"dv.updated_at",
	}
	for _, col := range must {
		if !strings.Contains(selectDutyViolationFields, col) {
			t.Errorf("SELECT missing column %q\nSQL:\n%s", col, selectDutyViolationFields)
		}
	}
}
