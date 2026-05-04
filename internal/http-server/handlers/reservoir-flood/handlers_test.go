package reservoirflood

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
	model "srmt-admin/internal/lib/model/reservoir-flood"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

// ---------- mock token verifier ----------

type mockTokenVerifier struct{ claims *token.Claims }

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) { return m.claims, nil }

// ---------- mock repo for hourly + config ----------

// captureRepo captures calls and lets tests pre-set responses.
type captureRepo struct {
	mu sync.Mutex

	// UpsertReservoirFloodHourly recording.
	upsertHourlyItems  []model.UpsertHourlyRequest
	upsertHourlyUserID int64
	upsertHourlyErr    error

	// GetReservoirFloodHourlyRange recording.
	hourlyRangeResult []model.HourlyRecord
	hourlyRangeErr    error

	// Config CRUD recording.
	upsertConfigReq    model.UpsertConfigRequest
	upsertConfigErr    error
	configList         []model.Config
	configListErr      error
	deleteConfigOrgID  int64
	deleteConfigErr    error
}

func (c *captureRepo) UpsertReservoirFloodHourly(_ context.Context, items []model.UpsertHourlyRequest, userID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upsertHourlyItems = append(c.upsertHourlyItems, items...)
	c.upsertHourlyUserID = userID
	return c.upsertHourlyErr
}

func (c *captureRepo) GetReservoirFloodHourlyRange(_ context.Context, _ []int64, _, _ time.Time) ([]model.HourlyRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hourlyRangeResult, c.hourlyRangeErr
}

func (c *captureRepo) UpsertReservoirFloodConfig(_ context.Context, req model.UpsertConfigRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upsertConfigReq = req
	return c.upsertConfigErr
}

func (c *captureRepo) GetAllReservoirFloodConfigs(_ context.Context) ([]model.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.configList, c.configListErr
}

func (c *captureRepo) DeleteReservoirFloodConfig(_ context.Context, orgID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleteConfigOrgID = orgID
	return c.deleteConfigErr
}

// ---------- helpers ----------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newRouter(repo *captureRepo, claims *token.Claims) http.Handler {
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(&mockTokenVerifier{claims: claims}))
	log := discardLogger()
	r.Post("/reservoir-flood/hourly", UpsertHourly(log, repo))
	r.Get("/reservoir-flood/hourly", GetHourly(log, repo))
	r.Post("/reservoir-flood/config", UpsertConfig(log, repo))
	r.Get("/reservoir-flood/config", GetConfigs(log, repo))
	r.Delete("/reservoir-flood/config", DeleteConfig(log, repo))
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
func dutyClaims(orgID int64) *token.Claims {
	return &token.Claims{UserID: 10, OrganizationID: orgID, Roles: []string{"reservoir_duty"}}
}

func hourlyBody(orgID int64, recordedAt string) string {
	return `[{
		"organization_id": ` + i2s(orgID) + `,
		"recorded_at": "` + recordedAt + `",
		"water_level_m": 815.4,
		"water_volume_mln_m3": 1234.5,
		"inflow_m3s": 250.0,
		"outflow_m3s": 200.0,
		"ges_flow_m3s": 180.0,
		"filtration_m3s": 5.0,
		"idle_discharge_m3s": 15.0,
		"duty_name": "Иванов И.И., смена 1"
	}]`
}

func i2s(i int64) string {
	bs, _ := json.Marshal(i)
	return string(bs)
}

// ===== Hourly UPSERT tests =====

func TestUpsertHourly_SCUser_AnyOrg_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly",
		hourlyBody(42, "2026-04-27T15:00:00Z"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 1 {
		t.Errorf("repo upsert calls: want 1 item, got %d", len(repo.upsertHourlyItems))
	}
}

func TestUpsertHourly_RaisUser_AnyOrg_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, raisClaims(), http.MethodPost, "/reservoir-flood/hourly",
		hourlyBody(99, "2026-04-27T15:00:00Z"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestUpsertHourly_DutyOwnOrg_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, dutyClaims(42), http.MethodPost, "/reservoir-flood/hourly",
		hourlyBody(42, "2026-04-27T15:00:00Z"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 1 {
		t.Errorf("repo upsert: want 1 item, got %d", len(repo.upsertHourlyItems))
	}
	if repo.upsertHourlyUserID != 10 {
		t.Errorf("user id propagated: want 10, got %d", repo.upsertHourlyUserID)
	}
}

func TestUpsertHourly_DutyForeignOrg_Forbidden(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, dutyClaims(42), http.MethodPost, "/reservoir-flood/hourly",
		hourlyBody(99, "2026-04-27T15:00:00Z"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 0 {
		t.Errorf("repo MUST NOT be called for foreign org; got %d items", len(repo.upsertHourlyItems))
	}
}

// Bulk batch with a mix of own + foreign orgs MUST be rejected wholesale.
// Atomicity: no partial writes.
func TestUpsertHourly_DutyMixedBatch_AllForbidden(t *testing.T) {
	repo := &captureRepo{}
	body := `[
		{"organization_id": 42, "recorded_at": "2026-04-27T15:00:00Z", "water_level_m": 815.4},
		{"organization_id": 99, "recorded_at": "2026-04-27T15:00:00Z", "water_level_m": 814.0}
	]`
	rr := doRequest(t, repo, dutyClaims(42), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 0 {
		t.Errorf("repo MUST NOT be called when any item is foreign-org; got %d items", len(repo.upsertHourlyItems))
	}
}

func TestUpsertHourly_InvalidTimeFormat_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "recorded_at": "not-a-time", "water_level_m": 815.4}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 0 {
		t.Errorf("repo MUST NOT be called on invalid time; got %d items", len(repo.upsertHourlyItems))
	}
}

func TestUpsertHourly_NegativeValue_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "recorded_at": "2026-04-27T15:00:00Z", "inflow_m3s": -1.0}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// recorded_at "2026-04-27T15:42:18Z" must be normalized to "2026-04-27T15:00:00Z"
// before being written to the DB. The handler is responsible for parsing the
// raw string into time.Time and truncating to the hour.
func TestUpsertHourly_TimeNormalization(t *testing.T) {
	repo := &captureRepo{}
	body := `[{
		"organization_id": 42,
		"recorded_at": "2026-04-27T15:42:18Z",
		"water_level_m": 815.4
	}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 1 {
		t.Fatalf("repo upsert: want 1 item, got %d", len(repo.upsertHourlyItems))
	}
	got := repo.upsertHourlyItems[0].RecordedAt
	want := "2026-04-27T15:00:00Z"
	if got != want {
		t.Errorf("RecordedAt: want %q (truncated to hour), got %q", want, got)
	}
}

// ===== Hourly GET tests =====

func TestGetHourly_SCSeesAll(t *testing.T) {
	repo := &captureRepo{
		hourlyRangeResult: []model.HourlyRecord{
			{ID: 1, OrganizationID: 42, RecordedAt: time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC)},
			{ID: 2, OrganizationID: 99, RecordedAt: time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)},
		},
	}
	rr := doRequest(t, repo, scClaims(), http.MethodGet, "/reservoir-flood/hourly?date=2026-04-27", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.HourlyRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("sc must see all records; got %d, want 2", len(got))
	}
}

func TestGetHourly_DutySeesOwnOrgOnly(t *testing.T) {
	repo := &captureRepo{
		hourlyRangeResult: []model.HourlyRecord{
			{ID: 1, OrganizationID: 42, RecordedAt: time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC)},
			{ID: 2, OrganizationID: 99, RecordedAt: time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)},
		},
	}
	rr := doRequest(t, repo, dutyClaims(42), http.MethodGet, "/reservoir-flood/hourly?date=2026-04-27", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.HourlyRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].OrganizationID != 42 {
		t.Errorf("reservoir_duty must see only own org; got %+v", got)
	}
}

// ===== Config tests =====

func TestUpsertConfig_SC_OK(t *testing.T) {
	repo := &captureRepo{}
	body := `{"organization_id": 42, "sort_order": 1, "is_active": true}`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/config", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertConfigReq.OrganizationID != 42 {
		t.Errorf("upsert payload: want orgID=42, got %d", repo.upsertConfigReq.OrganizationID)
	}
}

// reservoir_duty MUST be rejected at the route level (Tier 2 = sc/rais only).
// In this test the router is built without the route-level RequireAnyRole gate
// (we wire all routes for handler-test convenience), so the handler MUST itself
// reject the duty role with 403. This keeps defence-in-depth even if the route
// gate is misconfigured at registration time.
func TestUpsertConfig_DutyForbidden(t *testing.T) {
	repo := &captureRepo{}
	body := `{"organization_id": 42, "sort_order": 1, "is_active": true}`
	rr := doRequest(t, repo, dutyClaims(42), http.MethodPost, "/reservoir-flood/config", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertConfigReq.OrganizationID != 0 {
		t.Errorf("repo must NOT be called for duty role; got orgID=%d", repo.upsertConfigReq.OrganizationID)
	}
}

func TestDeleteConfig_OK(t *testing.T) {
	repo := &captureRepo{}
	rr := doRequest(t, repo, scClaims(), http.MethodDelete, "/reservoir-flood/config?organization_id=42", "")
	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Fatalf("status: want 204 or 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if repo.deleteConfigOrgID != 42 {
		t.Errorf("delete: want orgID=42, got %d", repo.deleteConfigOrgID)
	}
}

func TestDeleteConfig_NotFound_NotFound(t *testing.T) {
	repo := &captureRepo{deleteConfigErr: storage.ErrNotFound}
	rr := doRequest(t, repo, scClaims(), http.MethodDelete, "/reservoir-flood/config?organization_id=999", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetConfigs_OK(t *testing.T) {
	repo := &captureRepo{
		configList: []model.Config{
			{ID: 1, OrganizationID: 42, OrganizationName: "Charvak", SortOrder: 1, IsActive: true},
		},
	}
	rr := doRequest(t, repo, scClaims(), http.MethodGet, "/reservoir-flood/config", "")
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

// ===== Regression tests for review findings =====

// Water level negative must be rejected (regression for security review RISK-1).
// water_level_m uses Baltic height datum and is always positive in operational
// contexts. The handler MUST reject negative values like every other metric.
func TestUpsertHourly_NegativeWaterLevel_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "recorded_at": "2026-04-27T15:00:00Z", "water_level_m": -1.0}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 0 {
		t.Errorf("repo MUST NOT be called on negative water_level_m; got %d items", len(repo.upsertHourlyItems))
	}
}

// Broken-account state (non-admin without org) on GET /hourly must yield 403,
// not silently empty 200. Regression for security review RISK-2.
func TestGetHourly_DutyNoOrgID_Forbidden(t *testing.T) {
	repo := &captureRepo{
		hourlyRangeResult: []model.HourlyRecord{
			{ID: 1, OrganizationID: 42, RecordedAt: time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC)},
		},
	}
	// duty user with OrganizationID = 0 (broken DB setup).
	claims := &token.Claims{UserID: 10, OrganizationID: 0, Roles: []string{"reservoir_duty"}}
	rr := doRequest(t, repo, claims, http.MethodGet, "/reservoir-flood/hourly?date=2026-04-27", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// Same regression but on GET /config.
func TestGetConfigs_DutyNoOrgID_Forbidden(t *testing.T) {
	repo := &captureRepo{
		configList: []model.Config{
			{ID: 1, OrganizationID: 42, OrganizationName: "Charvak", SortOrder: 1, IsActive: true},
		},
	}
	claims := &token.Claims{UserID: 10, OrganizationID: 0, Roles: []string{"reservoir_duty"}}
	rr := doRequest(t, repo, claims, http.MethodGet, "/reservoir-flood/config", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// ===== Wave 2: capacity_mwt / weather_condition / temperature_c =====

// Round-trip: POST /hourly with the three new fields, then verify the captured
// upsert items carry them through to repo.
func TestUpsertHourly_NewMetrics_OK(t *testing.T) {
	repo := &captureRepo{}
	body := `[{
		"organization_id": 42,
		"recorded_at": "2026-04-27T15:00:00Z",
		"water_level_m": 815.4,
		"capacity_mwt": 100.5,
		"weather_condition": "ясно",
		"temperature_c": -3.5
	}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 1 {
		t.Fatalf("repo upsert: want 1 item, got %d", len(repo.upsertHourlyItems))
	}
	it := repo.upsertHourlyItems[0]
	if it.CapacityMwt.Value == nil || *it.CapacityMwt.Value != 100.5 {
		t.Errorf("capacity_mwt: want 100.5 set, got Value=%v Set=%v", it.CapacityMwt.Value, it.CapacityMwt.Set)
	}
	if it.WeatherCondition.Value == nil || *it.WeatherCondition.Value != "ясно" {
		t.Errorf("weather_condition: want \"ясно\" set, got Value=%v Set=%v", it.WeatherCondition.Value, it.WeatherCondition.Set)
	}
	if it.TemperatureC.Value == nil || *it.TemperatureC.Value != -3.5 {
		t.Errorf("temperature_c: want -3.5 set (negative valid), got Value=%v Set=%v", it.TemperatureC.Value, it.TemperatureC.Set)
	}
}

// capacity_mwt < 0 must be rejected (mirrors all other m³/s metrics).
func TestUpsertHourly_NegativeCapacity_BadRequest(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "recorded_at": "2026-04-27T15:00:00Z", "capacity_mwt": -5.0}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if len(repo.upsertHourlyItems) != 0 {
		t.Errorf("repo MUST NOT be called on negative capacity_mwt; got %d items", len(repo.upsertHourlyItems))
	}
}

// temperature_c < 0 is valid (winter). MUST NOT be rejected.
func TestUpsertHourly_NegativeTemperature_OK(t *testing.T) {
	repo := &captureRepo{}
	body := `[{"organization_id": 42, "recorded_at": "2026-04-27T15:00:00Z", "temperature_c": -10.5}]`
	rr := doRequest(t, repo, scClaims(), http.MethodPost, "/reservoir-flood/hourly", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200 (negative temperature is valid), got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// GET /hourly must surface the three new fields in the JSON response.
func TestGetHourly_ReturnsNewMetrics(t *testing.T) {
	cap := 250.0
	weather := "облачно"
	temp := 18.5
	repo := &captureRepo{
		hourlyRangeResult: []model.HourlyRecord{
			{
				ID:               1,
				OrganizationID:   42,
				RecordedAt:       time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC),
				CapacityMwt:      &cap,
				WeatherCondition: &weather,
				TemperatureC:     &temp,
			},
		},
	}
	rr := doRequest(t, repo, scClaims(), http.MethodGet, "/reservoir-flood/hourly?date=2026-04-27", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.HourlyRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 record, got %d", len(got))
	}
	if got[0].CapacityMwt == nil || *got[0].CapacityMwt != 250.0 {
		t.Errorf("capacity_mwt: want 250.0, got %v", got[0].CapacityMwt)
	}
	if got[0].WeatherCondition == nil || *got[0].WeatherCondition != "облачно" {
		t.Errorf("weather_condition: want \"облачно\", got %v", got[0].WeatherCondition)
	}
	if got[0].TemperatureC == nil || *got[0].TemperatureC != 18.5 {
		t.Errorf("temperature_c: want 18.5, got %v", got[0].TemperatureC)
	}
}
