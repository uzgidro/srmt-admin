package repo

import (
	"context"
	"strings"
	"testing"

	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
)

// Structural tests for getReservoirSummaryQuery. Behavior tests against a
// real DB live in the dev smoke loop; here we lock in the SQL invariants
// that the JSON / Excel callers depend on so accidental edits surface in CI.

func TestGetReservoirSummaryQuery_WhitelistFromConfig(t *testing.T) {
	q := getReservoirSummaryQuery()

	// The org_data CTE must source organizations from
	// reservoir_summary_config, not "SELECT DISTINCT FROM reservoir_data".
	// If this assertion ever flips, the report silently starts including
	// every reservoir that has any data, breaking the whitelist contract.
	if !strings.Contains(q, "FROM reservoir_summary_config") {
		t.Errorf("query missing whitelist source `FROM reservoir_summary_config`:\n%s", q)
	}
	if strings.Contains(q, "SELECT DISTINCT organization_id\n    FROM reservoir_data") {
		t.Errorf("query still uses old `SELECT DISTINCT FROM reservoir_data` for org_data")
	}
}

func TestGetReservoirSummaryQuery_ItogFiltersOnIncludeInTotal(t *testing.T) {
	q := getReservoirSummaryQuery()

	// The ИТОГО row must sum only `include_in_total = TRUE` orgs. The old
	// design used 16 EXISTS clauses against organization_type_links.type_id = 8
	// — none of those should remain.
	if !strings.Contains(q, "rsc.include_in_total") {
		t.Errorf("ИТОГО row does not filter on rsc.include_in_total:\n%s", q)
	}
	if strings.Contains(q, "AND otl.type_id = 8") {
		t.Errorf("query still uses legacy organization_type_links.type_id = 8 filter")
	}
}

func TestGetReservoirSummaryQuery_IncomingVolumeRespectsConfig(t *testing.T) {
	q := getReservoirSummaryQuery()

	// incoming_volume_mln_m3 in the ИТОГО row historically summed across
	// ALL orgs without the include filter — a long-standing inconsistency.
	// The rewrite aligns it with the other metrics. Two checks:
	//   1. negative — the unfiltered SUM is gone
	//   2. positive — the new ИТОГО expression filters via include_in_total
	unfiltered := "SUM(COALESCE(iv.incoming_volume_mln_m3_current_year, 0))) AS incoming_volume_mln_m3"
	if strings.Contains(q, unfiltered) {
		t.Errorf("incoming_volume_mln_m3 in ИТОГО is unfiltered (legacy behavior); should respect include_in_total")
	}
	filtered := "CASE WHEN rsc.include_in_total THEN COALESCE(iv.incoming_volume_mln_m3_current_year, 0) ELSE 0 END"
	if !strings.Contains(q, filtered) {
		t.Errorf("ИТОГО incoming_volume expression is missing the include_in_total guard:\nexpected substring %q", filtered)
	}
}

// Structural tests for the reservoir_summary_config CRUD SQL. The actual
// behaviour against a real DB is exercised via the dev smoke loop; here we
// pin the column lists so accidental edits surface in CI.

// reservoirSummaryConfigQueryCapturer is a thin shim that captures SQL
// passed to ExecContext / QueryContext / QueryRowContext so we can assert
// on the literal text without standing up a real DB. It satisfies just
// enough of the *sql.DB surface that the repo methods we test exercise.

// TestUpsertConfigSQL_IncludesModsnowEnabled locks in that the upsert SQL
// writes the modsnow_enabled flag. If the column is dropped from the
// INSERT/UPDATE list, the field would silently never persist and Sardoba
// would re-show modsnow in Excel on the next deploy.
func TestUpsertConfigSQL_IncludesModsnowEnabled(t *testing.T) {
	got := upsertReservoirSummaryConfigQuery()
	if !strings.Contains(got, "modsnow_enabled") {
		t.Errorf("UpsertReservoirSummaryConfig SQL is missing `modsnow_enabled` column:\n%s", got)
	}
}

// TestSelectConfigSQL_IncludesModsnowEnabled locks in that the select SQL
// returns modsnow_enabled. Without this column the generator/JSON layer
// would always see the zero-value (false) and silently hide modsnow for
// every reservoir.
func TestSelectConfigSQL_IncludesModsnowEnabled(t *testing.T) {
	for name, q := range map[string]string{
		"GetAllReservoirSummaryConfigs":  getAllReservoirSummaryConfigsQuery(),
		"GetReservoirSummaryConfigByOrgID": getReservoirSummaryConfigByOrgIDQuery(),
	} {
		if !strings.Contains(q, "modsnow_enabled") {
			t.Errorf("%s SQL is missing `modsnow_enabled` column:\n%s", name, q)
		}
	}
}

// Compile-time sanity that the model carries the flag — keeps the test
// honest if someone deletes the column from the SQL AND the field at the
// same time. Not a real assertion, just a structural pin.
var _ = func() reservoirsummary.ReservoirSummaryConfig {
	_ = context.Background()
	return reservoirsummary.ReservoirSummaryConfig{ModsnowEnabled: true}
}

func TestGetReservoirSummaryQuery_OrderBySortPosition(t *testing.T) {
	q := getReservoirSummaryQuery()

	// Final ORDER BY must use the computed sort_position column (per-org
	// rows: sort_order*10; ИТОГО: max(sort_order WHERE include_in_total)*10+5).
	// Anything else (organization_id, organization_name, etc.) means the
	// configured ordering is being ignored.
	if !strings.Contains(q, "ORDER BY sort_position") {
		t.Errorf("query does not ORDER BY sort_position:\n%s", q)
	}
	if !strings.Contains(q, "itog_position") {
		t.Errorf("query missing itog_position CTE that interleaves ИТОГО with per-org rows")
	}
}

// volume_source migration (000086) adds a column controlling how Volume.Current
// is resolved when the daily snapshot is missing or stale. The CRUD SQL must
// round-trip the column, or the handler-layer strategy switch silently reads
// zero values from the DB.
func TestUpsertConfigSQL_IncludesVolumeSource(t *testing.T) {
	q := upsertReservoirSummaryConfigQuery()

	if !strings.Contains(q, "volume_source") {
		t.Errorf("upsert query does not write volume_source:\n%s", q)
	}
}

func TestSelectAllConfigsSQL_IncludesVolumeSource(t *testing.T) {
	q := getAllReservoirSummaryConfigsQuery()

	if !strings.Contains(q, "volume_source") {
		t.Errorf("GetAllReservoirSummaryConfigs query does not select volume_source:\n%s", q)
	}
}

func TestSelectConfigByOrgIDSQL_IncludesVolumeSource(t *testing.T) {
	q := getReservoirSummaryConfigByOrgIDQuery()

	if !strings.Contains(q, "volume_source") {
		t.Errorf("GetReservoirSummaryConfigByOrgID query does not select volume_source:\n%s", q)
	}
}
