package solar

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/solar"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

// ---------- mock token verifier ----------

type mockTokenVerifier struct{ claims *token.Claims }

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) { return m.claims, nil }

// ---------- mock repo for solar handlers ----------
//
// captureRepo records calls and lets tests pre-set responses. It implements
// the union of all solar handler dependencies — config, daily data, plans —
// so a single mock can drive every handler under test.
type captureRepo struct {
	mu sync.Mutex

	// Daily data upsert recording.
	upsertDailyItems  []model.UpsertDailyDataRequest
	upsertDailyUserID int64
	upsertDailyErr    error

	// Daily data range recording.
	dailyRangeResult []model.DailyData
	dailyRangeErr    error

	// Config CRUD recording.
	upsertConfigReq   model.UpsertConfigRequest
	upsertConfigErr   error
	configList        []model.Config
	configListErr     error
	deleteConfigOrgID int64
	deleteConfigErr   error

	// Plan recording.
	upsertPlanItems  []model.UpsertPlanRequest
	upsertPlanUserID int64
	upsertPlanErr    error
	planListYear     int
	planListResult   []model.ProductionPlan
	planListErr      error
}

func (c *captureRepo) UpsertSolarDailyData(_ context.Context, items []model.UpsertDailyDataRequest, userID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upsertDailyItems = append(c.upsertDailyItems, items...)
	c.upsertDailyUserID = userID
	return c.upsertDailyErr
}

func (c *captureRepo) GetSolarDailyDataRange(_ context.Context, _ []int64, _, _ time.Time) ([]model.DailyData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dailyRangeResult, c.dailyRangeErr
}

func (c *captureRepo) UpsertSolarConfig(_ context.Context, req model.UpsertConfigRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upsertConfigReq = req
	return c.upsertConfigErr
}

func (c *captureRepo) GetAllSolarConfigs(_ context.Context) ([]model.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.configList, c.configListErr
}

func (c *captureRepo) DeleteSolarConfig(_ context.Context, orgID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleteConfigOrgID = orgID
	return c.deleteConfigErr
}

func (c *captureRepo) BulkUpsertSolarPlan(_ context.Context, plans []model.UpsertPlanRequest, userID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upsertPlanItems = append(c.upsertPlanItems, plans...)
	c.upsertPlanUserID = userID
	return c.upsertPlanErr
}

func (c *captureRepo) GetSolarPlans(_ context.Context, year int) ([]model.ProductionPlan, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.planListYear = year
	return c.planListResult, c.planListErr
}

// ---------- helpers ----------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newRouter wires every solar handler under one chi router using the same
// auth middleware as production. Route-level RequireAnyRole guards are NOT
// installed here because the handler under test must apply its own
// defence-in-depth check (Tier 2 routes still call the handler in this
// test setup, so the handler MUST refuse cascade callers itself).
func newRouter(repo *captureRepo, claims *token.Claims) http.Handler {
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(&mockTokenVerifier{claims: claims}))
	log := discardLogger()
	r.Post("/solar/daily-data", UpsertDailyData(log, repo))
	r.Get("/solar/daily-data", GetDailyData(log, repo))
	r.Post("/solar/config", UpsertConfig(log, repo))
	r.Get("/solar/config", GetConfigs(log, repo))
	r.Delete("/solar/config", DeleteConfig(log, repo))
	r.Post("/solar/plans", BulkUpsertPlan(log, repo))
	r.Get("/solar/plans", GetPlans(log, repo))
	return r
}

func doRequest(t *testing.T, repo *captureRepo, claims *token.Claims, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := newRouter(repo, claims)
	var bodyReader *bytes.Buffer
	if body == "" {
		bodyReader = bytes.NewBuffer(nil)
	} else {
		bodyReader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, target, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func scClaims() *token.Claims {
	return &token.Claims{UserID: 1, OrganizationID: 1, Roles: []string{"sc"}}
}
func raisClaims() *token.Claims {
	return &token.Claims{UserID: 2, OrganizationID: 1, Roles: []string{"rais"}}
}
func cascadeClaims(orgID int64) *token.Claims {
	return &token.Claims{UserID: 10, OrganizationID: orgID, Roles: []string{"cascade"}}
}

func dailyBody(orgID int64, date string) string {
	return `[{
		"organization_id": ` + i2s(orgID) + `,
		"date": "` + date + `",
		"generation_kwh": 620.5,
		"grid_export_kwh": 540.0
	}]`
}

func i2s(i int64) string {
	bs, _ := json.Marshal(i)
	return string(bs)
}

// ===== UpsertSolarDailyData tests =====

func TestUpsertSolarDailyData_SCAnyOrg_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/daily-data", dailyBody(42, "2026-04-28"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertDailyItems) != 1 {
		t.Errorf("repo upsert: want 1 item, got %d", len(repo.upsertDailyItems))
	}
}

func TestUpsertSolarDailyData_RaisAnyOrg_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, raisClaims(), http.MethodPost, "/solar/daily-data", dailyBody(99, "2026-04-28"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestUpsertSolarDailyData_CascadeOwnOrg_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodPost, "/solar/daily-data", dailyBody(42, "2026-04-28"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertDailyUserID != 10 {
		t.Errorf("user id propagated: want 10, got %d", repo.upsertDailyUserID)
	}
}

func TestUpsertSolarDailyData_CascadeForeignOrg_Forbidden(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodPost, "/solar/daily-data", dailyBody(99, "2026-04-28"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertDailyItems) != 0 {
		t.Errorf("repo MUST NOT be called for foreign org; got %d items", len(repo.upsertDailyItems))
	}
}

// Bulk batch with mix of own + foreign orgs MUST be rejected wholesale —
// no partial writes. Critical IDOR protection.
func TestUpsertSolarDailyData_CascadeMixedBatch_AllForbidden(t *testing.T) {
	repo := &captureRepo{}
	body := `[
		{"organization_id": 42, "date": "2026-04-28", "generation_kwh": 600.0},
		{"organization_id": 99, "date": "2026-04-28", "generation_kwh": 500.0}
	]`
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodPost, "/solar/daily-data", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertDailyItems) != 0 {
		t.Errorf("repo MUST NOT be called when any item is foreign-org; got %d items", len(repo.upsertDailyItems))
	}
}

// Broken-account state: cascade user without OrganizationID claim → 403,
// not 500 and not silent 200. Regression for known security issue.
func TestUpsertSolarDailyData_CascadeNoOrgID_Forbidden(t *testing.T) {
	repo := &captureRepo{}
	claims := &token.Claims{UserID: 10, OrganizationID: 0, Roles: []string{"cascade"}}
	rr := doRequest(t, repo, claims, http.MethodPost, "/solar/daily-data", dailyBody(42, "2026-04-28"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertDailyItems) != 0 {
		t.Errorf("repo MUST NOT be called when caller has no org; got %d items", len(repo.upsertDailyItems))
	}
}

func TestUpsertSolarDailyData_NegativeGeneration_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "date": "2026-04-28", "generation_kwh": -1.0}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/daily-data", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertDailyItems) != 0 {
		t.Errorf("repo MUST NOT be called on negative generation; got %d items", len(repo.upsertDailyItems))
	}
}

func TestUpsertSolarDailyData_NegativeGridExport_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "date": "2026-04-28", "grid_export_kwh": -50.0}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/daily-data", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestUpsertSolarDailyData_InvalidDate_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "date": "not-a-date", "generation_kwh": 100.0}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/daily-data", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertDailyItems) != 0 {
		t.Errorf("repo MUST NOT be called on invalid date; got %d items", len(repo.upsertDailyItems))
	}
}

// ===== GetSolarDailyData tests =====

func TestGetSolarDailyData_SCSeesAll(t *testing.T) {
	repo := &captureRepo{
		dailyRangeResult: []model.DailyData{
			{ID: 1, OrganizationID: 42, Date: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)},
			{ID: 2, OrganizationID: 99, Date: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)},
		},
	}
	rr := doRequest(t, repo, scClaims(), http.MethodGet, "/solar/daily-data?date=2026-04-28", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.DailyData
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("sc must see all records; got %d, want 2", len(got))
	}
}

func TestGetSolarDailyData_CascadeSeesOwnOrgOnly(t *testing.T) {
	repo := &captureRepo{
		dailyRangeResult: []model.DailyData{
			{ID: 1, OrganizationID: 42, Date: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)},
			{ID: 2, OrganizationID: 99, Date: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)},
		},
	}
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodGet, "/solar/daily-data?date=2026-04-28", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.DailyData
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].OrganizationID != 42 {
		t.Errorf("cascade must see only own org; got %+v", got)
	}
}

// Cross-org info disclosure regression: cascade user passing
// `?organization_id=99` (a foreign org) must NOT receive that data even
// though they may have legitimate access to other orgs. The query param
// is a hint, not a source of truth.
func TestGetSolarDailyData_CascadeForeignOrgQueryParam_FilteredOut(t *testing.T) {
	repo := &captureRepo{
		dailyRangeResult: []model.DailyData{
			{ID: 2, OrganizationID: 99, Date: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)},
		},
	}
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodGet, "/solar/daily-data?date=2026-04-28&organization_id=99", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.DailyData
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("cascade requesting foreign org via query param must get no records; got %+v", got)
	}
}

func TestGetSolarDailyData_CascadeNoOrgID_Forbidden(t *testing.T) {
	repo := &captureRepo{
		dailyRangeResult: []model.DailyData{
			{ID: 1, OrganizationID: 42, Date: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)},
		},
	}
	claims := &token.Claims{UserID: 10, OrganizationID: 0, Roles: []string{"cascade"}}
	rr := doRequest(t, repo, claims, http.MethodGet, "/solar/daily-data?date=2026-04-28", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// ===== Config tests =====

func TestUpsertSolarConfig_SC_OK(t *testing.T) {
	repo := &captureRepo{}
	body := `{"organization_id": 42, "installed_capacity_kw": 150.0, "sort_order": 1}`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/config", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertConfigReq.OrganizationID != 42 {
		t.Errorf("upsert payload: want orgID=42, got %d", repo.upsertConfigReq.OrganizationID)
	}
	if repo.upsertConfigReq.InstalledCapacityKW != 150.0 {
		t.Errorf("upsert payload: want capacity=150, got %v", repo.upsertConfigReq.InstalledCapacityKW)
	}
}

// Defence-in-depth: cascade role MUST be rejected by the handler even when
// the route-level RequireAnyRole is misconfigured (production would normally
// stop them at the router; this test ensures the handler doesn't blindly
// trust the route gate).
func TestUpsertSolarConfig_CascadeForbidden_RouteLevel(t *testing.T) {
	repo := &captureRepo{}
	body := `{"organization_id": 42, "installed_capacity_kw": 150.0, "sort_order": 1}`
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodPost, "/solar/config", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertConfigReq.OrganizationID != 0 {
		t.Errorf("repo must NOT be called for cascade; got orgID=%d", repo.upsertConfigReq.OrganizationID)
	}
}

func TestGetSolarConfigs_OK(t *testing.T) {
	repo := &captureRepo{
		configList: []model.Config{
			{ID: 1, OrganizationID: 42, OrganizationName: "Solar A", InstalledCapacityKW: 150.0, SortOrder: 1},
		},
	}
	rr := doRequest(t, repo, scClaims(), http.MethodGet, "/solar/config", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.Config
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("want 1 config, got %d", len(got))
	}
}

func TestDeleteSolarConfig_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, scClaims(), http.MethodDelete, "/solar/config?organization_id=42", "")
	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Fatalf("status: want 204 or 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.deleteConfigOrgID != 42 {
		t.Errorf("delete: want orgID=42, got %d", repo.deleteConfigOrgID)
	}
}

func TestDeleteSolarConfig_NotFound_404(t *testing.T) {
	repo := &captureRepo{deleteConfigErr: storage.ErrNotFound}
	rr := doRequest(t, repo, scClaims(), http.MethodDelete, "/solar/config?organization_id=999", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteSolarConfig_CascadeForbidden_RouteLevel(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodDelete, "/solar/config?organization_id=42", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.deleteConfigOrgID != 0 {
		t.Errorf("repo must NOT be called for cascade; got orgID=%d", repo.deleteConfigOrgID)
	}
}

// ===== Plan tests =====

func TestBulkUpsertSolarPlan_SC_OK(t *testing.T) {
	repo := &captureRepo{}
	body := `{"plans": [{"organization_id": 42, "year": 2026, "month": 4, "plan_thousand_kwh": 18.5}]}`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/plans", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertPlanItems) != 1 {
		t.Fatalf("plans: want 1, got %d", len(repo.upsertPlanItems))
	}
	if repo.upsertPlanItems[0].PlanThousandKWh != 18.5 {
		t.Errorf("plan_thousand_kwh: want 18.5, got %v", repo.upsertPlanItems[0].PlanThousandKWh)
	}
}

func TestBulkUpsertSolarPlan_NegativePlan_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `{"plans": [{"organization_id": 42, "year": 2026, "month": 4, "plan_thousand_kwh": -5.0}]}`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/solar/plans", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertPlanItems) != 0 {
		t.Errorf("repo MUST NOT be called on negative plan; got %d items", len(repo.upsertPlanItems))
	}
}

// Defence-in-depth: cascade role MUST be rejected on plan write.
func TestBulkUpsertSolarPlan_CascadeForbidden_RouteLevel(t *testing.T) {
	repo := &captureRepo{}
	body := `{"plans": [{"organization_id": 42, "year": 2026, "month": 4, "plan_thousand_kwh": 18.5}]}`
	rr := doRequest(t, repo, cascadeClaims(42), http.MethodPost, "/solar/plans", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertPlanItems) != 0 {
		t.Errorf("repo MUST NOT be called for cascade; got %d items", len(repo.upsertPlanItems))
	}
}

func TestGetSolarPlans_FilterByYear(t *testing.T) {
	repo := &captureRepo{
		planListResult: []model.ProductionPlan{
			{ID: 1, OrganizationID: 42, Year: 2026, Month: 4, PlanThousandKWh: 18.5},
		},
	}
	rr := doRequest(t, repo, scClaims(), http.MethodGet, "/solar/plans?year=2026", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.planListYear != 2026 {
		t.Errorf("repo received year: want 2026, got %d", repo.planListYear)
	}
	var got []model.ProductionPlan
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Year != 2026 {
		t.Errorf("want 1 plan with year=2026; got %+v", got)
	}
}
