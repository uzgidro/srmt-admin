package gesreport

import (
	"encoding/json"
	"net/http"
	"testing"

	"srmt-admin/internal/token"
)

// decodeStructuredError parses {"error","code","details":[{...}]} and returns
// all three. Used by the structured-error tests so we can assert each piece
// of the contract independently.
func decodeStructuredError(t *testing.T, body []byte) (errMsg string, code string, details []map[string]any) {
	t.Helper()
	var payload struct {
		Error   string                   `json:"error"`
		Code    string                   `json:"code"`
		Details []map[string]any         `json:"details"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("response is not valid JSON: %v; body=%s", err, body)
	}
	return payload.Error, payload.Code, payload.Details
}

// --- consumption_m3_s save tests ---

// Positive case: a small non-negative value is accepted and forwarded to the
// upsert layer verbatim. This pins the wiring from JSON → Optional[float64].
func TestUpsertDailyData_ConsumptionPersisted(t *testing.T) {
	upserter := &captureGESUpserter{}
	claims := &token.Claims{
		UserID:          1,
		OrganizationIDs: []int64{1},
		Roles:           []string{"sc"},
	}
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"consumption_m3_s": 1.5
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 1 {
		t.Fatalf("upserter should have been called once; got %d", len(upserter.last))
	}
	got := upserter.last[0].ConsumptionM3s
	if !got.Set {
		t.Fatal("ConsumptionM3s.Set must be true when payload includes the field")
	}
	if got.Value == nil {
		t.Fatal("ConsumptionM3s.Value must be non-nil for explicit 1.5")
	}
	if *got.Value != 1.5 {
		t.Errorf("ConsumptionM3s.Value: want 1.5, got %v", *got.Value)
	}
}

// Explicit null: payload {"consumption_m3_s": null} → Set=true, Value=nil.
// Repo CASE WHEN writes NULL to the column. This pins three-state semantics.
func TestUpsertDailyData_ConsumptionNullExplicit(t *testing.T) {
	upserter := &captureGESUpserter{}
	claims := &token.Claims{
		UserID:          1,
		OrganizationIDs: []int64{1},
		Roles:           []string{"sc"},
	}
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"consumption_m3_s": null
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	got := upserter.last[0].ConsumptionM3s
	if !got.Set {
		t.Error("ConsumptionM3s.Set must be true when payload sends null")
	}
	if got.Value != nil {
		t.Errorf("ConsumptionM3s.Value: want nil for explicit null, got %v", *got.Value)
	}
}

// Absent: payload omits the field entirely → Set=false, Value=nil. Repo CASE
// WHEN preserves the existing column value. This pins partial-update semantics.
func TestUpsertDailyData_ConsumptionAbsent_PartialUpdate(t *testing.T) {
	upserter := &captureGESUpserter{}
	claims := &token.Claims{
		UserID:          1,
		OrganizationIDs: []int64{1},
		Roles:           []string{"sc"},
	}
	// No consumption_m3_s key at all.
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"working_aggregates": 1
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	got := upserter.last[0].ConsumptionM3s
	if got.Set {
		t.Error("ConsumptionM3s.Set must be false when payload omits the field")
	}
}

// Negative value → 400 with structured error: code=save.field_negative,
// details carry organization_id, field, value. Frontend keys off the code to
// localize and binds details[].field to highlight the input. Upserter MUST
// NOT be called when validation fails.
func TestUpsertDailyData_ConsumptionNegative_StructuredError(t *testing.T) {
	upserter := &captureGESUpserter{}
	claims := &token.Claims{
		UserID:          1,
		OrganizationIDs: []int64{1},
		Roles:           []string{"sc"},
	}
	body := `[{
		"organization_id": 16,
		"date": "2026-04-13",
		"consumption_m3_s": -1.5
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 0 {
		t.Errorf("upserter must NOT be called when validation fails; got %d items", len(upserter.last))
	}

	errMsg, code, details := decodeStructuredError(t, rr.Body.Bytes())
	if code != "save.field_negative" {
		t.Errorf("code: want %q, got %q (full body: %s)", "save.field_negative", code, rr.Body.String())
	}
	if errMsg == "" {
		t.Error("error message must not be empty (frontend fallback)")
	}
	if len(details) != 1 {
		t.Fatalf("details: want 1 entry, got %d (body: %s)", len(details), rr.Body.String())
	}
	d := details[0]
	if d["field"] != "consumption_m3_s" {
		t.Errorf("details[0].field: want %q, got %v", "consumption_m3_s", d["field"])
	}
	// JSON numbers decode to float64.
	if orgID, _ := d["organization_id"].(float64); orgID != 16 {
		t.Errorf("details[0].organization_id: want 16, got %v", d["organization_id"])
	}
	if val, _ := d["value"].(float64); val != -1.5 {
		t.Errorf("details[0].value: want -1.5, got %v", d["value"])
	}
}

// Aggregate sum violation → 400 with structured code+details. Migrating the
// existing test (which only checked the human-readable error string) onto
// the new format. The plain `error` text is preserved for backwards compat
// (already covered by TestUpsertDailyData_RejectsAggregatesExceedingTotal),
// this one pins the new machine-readable contract.
func TestUpsertDailyData_AggregatesExceedTotal_StructuredError(t *testing.T) {
	const stationOrgID int64 = 10
	upserter := &captureGESUpserter{
		totals: map[int64]int{stationOrgID: 4},
	}
	claims := &token.Claims{
		UserID:          1,
		OrganizationIDs: []int64{1},
		Roles:           []string{"sc"},
	}
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"working_aggregates": 4,
		"repair_aggregates": 1,
		"modernization_aggregates": 0
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	_, code, details := decodeStructuredError(t, rr.Body.Bytes())
	if code != "save.aggregates_exceed_total" {
		t.Errorf("code: want %q, got %q (body: %s)", "save.aggregates_exceed_total", code, rr.Body.String())
	}
	if len(details) != 1 {
		t.Fatalf("details: want 1, got %d", len(details))
	}
	d := details[0]
	if orgID, _ := d["organization_id"].(float64); orgID != 10 {
		t.Errorf("details[0].organization_id: want 10, got %v", d["organization_id"])
	}
	if d["date"] != "2026-04-13" {
		t.Errorf("details[0].date: want 2026-04-13, got %v", d["date"])
	}
	if w, _ := d["working"].(float64); w != 4 {
		t.Errorf("details[0].working: want 4, got %v", d["working"])
	}
	if r, _ := d["repair"].(float64); r != 1 {
		t.Errorf("details[0].repair: want 1, got %v", d["repair"])
	}
	if m, _ := d["modernization"].(float64); m != 0 {
		t.Errorf("details[0].modernization: want 0, got %v", d["modernization"])
	}
	if s, _ := d["sum"].(float64); s != 5 {
		t.Errorf("details[0].sum: want 5, got %v", d["sum"])
	}
	if total, _ := d["total"].(float64); total != 4 {
		t.Errorf("details[0].total: want 4, got %v", d["total"])
	}
}

// Production-cap violation → 400 with structured code save.production_exceeds_max
// and details carrying organization_id + value + max.
func TestUpsertDailyData_ProductionExceedsMax_StructuredError(t *testing.T) {
	const stationOrgID int64 = 1
	upserter := &captureGESUpserter{
		maxProd: map[int64]float64{stationOrgID: 5.0},
	}
	claims := &token.Claims{
		UserID:          1,
		OrganizationIDs: []int64{1},
		Roles:           []string{"sc"},
	}
	body := `[{
		"organization_id": 1,
		"date": "2026-04-13",
		"daily_production_mln_kwh": 10.0
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	_, code, details := decodeStructuredError(t, rr.Body.Bytes())
	if code != "save.production_exceeds_max" {
		t.Errorf("code: want %q, got %q (body: %s)", "save.production_exceeds_max", code, rr.Body.String())
	}
	if len(details) != 1 {
		t.Fatalf("details: want 1, got %d", len(details))
	}
	d := details[0]
	if orgID, _ := d["organization_id"].(float64); orgID != 1 {
		t.Errorf("details[0].organization_id: want 1, got %v", d["organization_id"])
	}
	if v, _ := d["value"].(float64); v != 10.0 {
		t.Errorf("details[0].value: want 10.0, got %v", d["value"])
	}
	if m, _ := d["max"].(float64); m != 5.0 {
		t.Errorf("details[0].max: want 5.0, got %v", d["max"])
	}
}
