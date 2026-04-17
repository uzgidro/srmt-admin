package gesreport

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/token"
)

// mockTokenVerifier mirrors the verifier used by reservoir-summary tests so
// the auth middleware can produce claims with whatever roles we want.
type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

// captureGESUpserter records the slice passed to UpsertGESDailyData so tests
// can assert per-item Optional field state.
type captureGESUpserter struct {
	mu      sync.Mutex
	last    []model.UpsertDailyDataRequest
	err     error
	parents map[int64]*int64 // optional: per-org parent overrides for cascade tests
}

func (c *captureGESUpserter) UpsertGESDailyData(_ context.Context, items []model.UpsertDailyDataRequest, _ int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = make([]model.UpsertDailyDataRequest, len(items))
	copy(c.last, items)
	return c.err
}

// GetOrganizationParentID returns the parent_org_id configured for the given
// org via the parents map. With no map configured (sc/rais tests), the lookup
// is never reached.
func (c *captureGESUpserter) GetOrganizationParentID(_ context.Context, orgID int64) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parents == nil {
		return nil, nil
	}
	return c.parents[orgID], nil
}

func newGESTestRouter(upserter *captureGESUpserter) http.Handler {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := UpsertDailyData(log, upserter)
	verifier := &mockTokenVerifier{claims: &token.Claims{
		UserID:         1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Post("/ges/daily-data", handler)
	return r
}

func doGESUpsert(t *testing.T, upserter *captureGESUpserter, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := newGESTestRouter(upserter)
	req := httptest.NewRequest(http.MethodPost, "/ges/daily-data", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestUpsertGESDailyData_AllAbsent(t *testing.T) {
	upserter := &captureGESUpserter{}
	body := `[{"organization_id": 100, "date": "2026-04-13"}]`
	rr := doGESUpsert(t, upserter, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 1 {
		t.Fatalf("captured items: want 1, got %d", len(upserter.last))
	}
	item := upserter.last[0]

	checks := []struct {
		name string
		set  bool
		val  any
	}{
		{"DailyProductionMlnKWh", item.DailyProductionMlnKWh.Set, item.DailyProductionMlnKWh.Value},
		{"WorkingAggregates", item.WorkingAggregates.Set, item.WorkingAggregates.Value},
		{"WaterLevelM", item.WaterLevelM.Set, item.WaterLevelM.Value},
		{"WaterVolumeMlnM3", item.WaterVolumeMlnM3.Set, item.WaterVolumeMlnM3.Value},
		{"WaterHeadM", item.WaterHeadM.Set, item.WaterHeadM.Value},
		{"ReservoirIncomeM3s", item.ReservoirIncomeM3s.Set, item.ReservoirIncomeM3s.Value},
		{"TotalOutflowM3s", item.TotalOutflowM3s.Set, item.TotalOutflowM3s.Value},
		{"GESFlowM3s", item.GESFlowM3s.Set, item.GESFlowM3s.Value},
	}
	for _, c := range checks {
		if c.set {
			t.Errorf("%s.Set = true, want false (absent)", c.name)
		}
		if !isNilPtr(c.val) {
			t.Errorf("%s.Value = %v, want nil", c.name, c.val)
		}
	}
}

func TestUpsertGESDailyData_AllNull(t *testing.T) {
	upserter := &captureGESUpserter{}
	body := `[{
		"organization_id": 100,
		"date": "2026-04-13",
		"daily_production_mln_kwh": null,
		"working_aggregates": null,
		"water_level_m": null,
		"water_volume_mln_m3": null,
		"water_head_m": null,
		"reservoir_income_m3s": null,
		"total_outflow_m3s": null,
		"ges_flow_m3s": null
	}]`
	rr := doGESUpsert(t, upserter, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	item := upserter.last[0]

	checks := []struct {
		name string
		set  bool
		val  any
	}{
		{"DailyProductionMlnKWh", item.DailyProductionMlnKWh.Set, item.DailyProductionMlnKWh.Value},
		{"WorkingAggregates", item.WorkingAggregates.Set, item.WorkingAggregates.Value},
		{"WaterLevelM", item.WaterLevelM.Set, item.WaterLevelM.Value},
		{"WaterVolumeMlnM3", item.WaterVolumeMlnM3.Set, item.WaterVolumeMlnM3.Value},
		{"WaterHeadM", item.WaterHeadM.Set, item.WaterHeadM.Value},
		{"ReservoirIncomeM3s", item.ReservoirIncomeM3s.Set, item.ReservoirIncomeM3s.Value},
		{"TotalOutflowM3s", item.TotalOutflowM3s.Set, item.TotalOutflowM3s.Value},
		{"GESFlowM3s", item.GESFlowM3s.Set, item.GESFlowM3s.Value},
	}
	for _, c := range checks {
		if !c.set {
			t.Errorf("%s.Set = false, want true (explicit null)", c.name)
		}
		if !isNilPtr(c.val) {
			t.Errorf("%s.Value = %v, want nil", c.name, c.val)
		}
	}
}

func TestUpsertGESDailyData_AllNumbers(t *testing.T) {
	upserter := &captureGESUpserter{}
	body := `[{
		"organization_id": 100,
		"date": "2026-04-13",
		"daily_production_mln_kwh": 1.1,
		"working_aggregates": 2,
		"water_level_m": 3.3,
		"water_volume_mln_m3": 4.4,
		"water_head_m": 5.5,
		"reservoir_income_m3s": 6.6,
		"total_outflow_m3s": 7.7,
		"ges_flow_m3s": 8.8
	}]`
	rr := doGESUpsert(t, upserter, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	item := upserter.last[0]

	if !item.DailyProductionMlnKWh.Set || item.DailyProductionMlnKWh.Value == nil || *item.DailyProductionMlnKWh.Value != 1.1 {
		t.Errorf("DailyProductionMlnKWh = %+v, want set with 1.1", item.DailyProductionMlnKWh)
	}
	if !item.WorkingAggregates.Set || item.WorkingAggregates.Value == nil || *item.WorkingAggregates.Value != 2 {
		t.Errorf("WorkingAggregates = %+v, want set with 2", item.WorkingAggregates)
	}
	if !item.WaterLevelM.Set || item.WaterLevelM.Value == nil || *item.WaterLevelM.Value != 3.3 {
		t.Errorf("WaterLevelM = %+v, want set with 3.3", item.WaterLevelM)
	}
	if !item.WaterVolumeMlnM3.Set || item.WaterVolumeMlnM3.Value == nil || *item.WaterVolumeMlnM3.Value != 4.4 {
		t.Errorf("WaterVolumeMlnM3 = %+v, want set with 4.4", item.WaterVolumeMlnM3)
	}
	if !item.WaterHeadM.Set || item.WaterHeadM.Value == nil || *item.WaterHeadM.Value != 5.5 {
		t.Errorf("WaterHeadM = %+v, want set with 5.5", item.WaterHeadM)
	}
	if !item.ReservoirIncomeM3s.Set || item.ReservoirIncomeM3s.Value == nil || *item.ReservoirIncomeM3s.Value != 6.6 {
		t.Errorf("ReservoirIncomeM3s = %+v, want set with 6.6", item.ReservoirIncomeM3s)
	}
	if !item.TotalOutflowM3s.Set || item.TotalOutflowM3s.Value == nil || *item.TotalOutflowM3s.Value != 7.7 {
		t.Errorf("TotalOutflowM3s = %+v, want set with 7.7", item.TotalOutflowM3s)
	}
	if !item.GESFlowM3s.Set || item.GESFlowM3s.Value == nil || *item.GESFlowM3s.Value != 8.8 {
		t.Errorf("GESFlowM3s = %+v, want set with 8.8", item.GESFlowM3s)
	}
}

func TestUpsertGESDailyData_Mixed(t *testing.T) {
	upserter := &captureGESUpserter{}
	body := `[{
		"organization_id": 100,
		"date": "2026-04-13",
		"daily_production_mln_kwh": 8.4,
		"working_aggregates": null,
		"ges_flow_m3s": 0
	}]`
	rr := doGESUpsert(t, upserter, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	item := upserter.last[0]

	// daily_production_mln_kwh: set with 8.4
	if !item.DailyProductionMlnKWh.Set || item.DailyProductionMlnKWh.Value == nil || *item.DailyProductionMlnKWh.Value != 8.4 {
		t.Errorf("DailyProductionMlnKWh = %+v, want set with 8.4", item.DailyProductionMlnKWh)
	}
	// working_aggregates: explicit null → set, value nil
	if !item.WorkingAggregates.Set {
		t.Errorf("WorkingAggregates.Set = false, want true (explicit null)")
	}
	if item.WorkingAggregates.Value != nil {
		t.Errorf("WorkingAggregates.Value = %v, want nil (explicit null)", *item.WorkingAggregates.Value)
	}
	// water_level_m: absent → set false, value nil
	if item.WaterLevelM.Set {
		t.Errorf("WaterLevelM.Set = true, want false (absent)")
	}
	if item.WaterLevelM.Value != nil {
		t.Errorf("WaterLevelM.Value = %v, want nil (absent)", *item.WaterLevelM.Value)
	}
	// ges_flow_m3s: explicit zero → set, value 0 (NOT absent!)
	if !item.GESFlowM3s.Set {
		t.Errorf("GESFlowM3s.Set = false, want true (explicit zero)")
	}
	if item.GESFlowM3s.Value == nil {
		t.Fatalf("GESFlowM3s.Value = nil, want pointer to 0")
	}
	if *item.GESFlowM3s.Value != 0 {
		t.Errorf("*GESFlowM3s.Value = %v, want 0", *item.GESFlowM3s.Value)
	}
}

func TestUpsertGESDailyData_AcceptsArray(t *testing.T) {
	upserter := &captureGESUpserter{}
	body := `[
		{"organization_id": 100, "date": "2026-04-13", "daily_production_mln_kwh": 1.5},
		{"organization_id": 101, "date": "2026-04-13", "daily_production_mln_kwh": 2.5}
	]`
	rr := doGESUpsert(t, upserter, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 2 {
		t.Fatalf("captured items: want 2, got %d", len(upserter.last))
	}
	if upserter.last[0].OrganizationID != 100 {
		t.Errorf("item[0].OrganizationID = %d, want 100", upserter.last[0].OrganizationID)
	}
	if upserter.last[1].OrganizationID != 101 {
		t.Errorf("item[1].OrganizationID = %d, want 101", upserter.last[1].OrganizationID)
	}
	if upserter.last[0].DailyProductionMlnKWh.Value == nil || *upserter.last[0].DailyProductionMlnKWh.Value != 1.5 {
		t.Errorf("item[0].DailyProductionMlnKWh = %+v, want 1.5", upserter.last[0].DailyProductionMlnKWh)
	}
	if upserter.last[1].DailyProductionMlnKWh.Value == nil || *upserter.last[1].DailyProductionMlnKWh.Value != 2.5 {
		t.Errorf("item[1].DailyProductionMlnKWh = %+v, want 2.5", upserter.last[1].DailyProductionMlnKWh)
	}
}

func TestUpsertGESDailyData_EmptyArray(t *testing.T) {
	upserter := &captureGESUpserter{}
	rr := doGESUpsert(t, upserter, `[]`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if upserter.last != nil {
		t.Errorf("upserter should not be called for empty array, got %d items", len(upserter.last))
	}
}

func TestUpsertGESDailyData_ItemIndexInError(t *testing.T) {
	upserter := &captureGESUpserter{}
	body := `[
		{"organization_id": 100, "date": "2026-04-13"},
		{"organization_id": 101, "date": "not-a-date"}
	]`
	rr := doGESUpsert(t, upserter, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, `"item_index":1`) && !strings.Contains(bodyStr, `"item_index": 1`) {
		// Try parsing JSON to check the field robustly.
		var payload map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err == nil {
			if v, ok := payload["item_index"]; ok {
				if n, ok := v.(float64); ok && int(n) == 1 {
					return
				}
			}
		}
		t.Errorf("response body does not contain item_index=1; body: %s", bodyStr)
	}
}

// isNilPtr reports whether the interface holds a nil pointer of any pointer type.
func isNilPtr(v any) bool {
	if v == nil {
		return true
	}
	switch p := v.(type) {
	case *float64:
		return p == nil
	case *int:
		return p == nil
	}
	return false
}
