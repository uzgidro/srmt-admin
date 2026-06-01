package repo

import (
	"strings"
	"testing"
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
