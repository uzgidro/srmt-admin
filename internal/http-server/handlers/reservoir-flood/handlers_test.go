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
	hourlyRangeResult    []model.HourlyRecord
	hourlyRangeStart     time.Time // last [start, end) window passed by handler
	hourlyRangeEnd       time.Time
	hourlyRangeCallCount int   // total calls to GetReservoirFloodHourlyRange
	hourlyRangeErr       error

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

// GetReservoirFloodHourlyRange records the [start, end) window the handler
// passed and returns ONLY the records that fall inside it. This keeps the
// regression for the timezone-window bug honest: if the handler computes a
// wrong window, the mock will silently drop records and the assertion will
// fail. Tests can still inject any fixture data via hourlyRangeResult.
func (c *captureRepo) GetReservoirFloodHourlyRange(_ context.Context, _ []int64, start, end time.Time) ([]model.HourlyRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hourlyRangeCallCount++
	c.hourlyRangeStart = start
	c.hourlyRangeEnd = end
	if c.hourlyRangeErr != nil {
		return nil, c.hourlyRangeErr
	}
	out := make([]model.HourlyRecord, 0, len(c.hourlyRangeResult))
	for _, rec := range c.hourlyRangeResult {
		if !rec.RecordedAt.Before(start) && rec.RecordedAt.Before(end) {
			out = append(out, rec)
		}
	}
	return out, nil
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

// tashkentLoc is the timezone fixture for new tests that exercise the
// local-day window. Using FixedZone avoids any tzdata-on-CI flakiness.
var tashkentLoc = time.FixedZone("Asia/Tashkent", 5*3600)

func newRouterInLoc(repo *captureRepo, claims *token.Claims, loc *time.Location) http.Handler {
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(&mockTokenVerifier{claims: claims}))
	log := discardLogger()
	r.Post("/reservoir-flood/hourly", UpsertHourly(log, repo))
	r.Get("/reservoir-flood/hourly", GetHourly(log, repo, loc))
	r.Post("/reservoir-flood/config", UpsertConfig(log, repo))
	r.Get("/reservoir-flood/config", GetConfigs(log, repo))
	r.Delete("/reservoir-flood/config", DeleteConfig(log, repo))
	return r
}

// newRouter is the historical wrapper for tests that don't care about TZ.
// Passing time.UTC keeps their behavior identical to the pre-fix code
// (date string → midnight UTC window).
func newRouter(repo *captureRepo, claims *token.Claims) http.Handler {
	return newRouterInLoc(repo, claims, time.UTC)
}

func doRequestInLoc(t *testing.T, repo *captureRepo, claims *token.Claims, loc *time.Location, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := newRouterInLoc(repo, claims, loc)
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

func doRequest(t *testing.T, repo *captureRepo, claims *token.Claims, method, target, body string) *httptest.ResponseRecorder {
	return doRequestInLoc(t, repo, claims, time.UTC, method, target, body)
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

// Regression: a record stored at local midnight in Tashkent (UTC+5) must be
// returned when the client requests THAT local day, not the day before.
// Pre-fix the GET handler interpreted "?date=YYYY-MM-DD" as a UTC midnight
// window, so a record at 2026-05-11T19:00:00Z (= 2026-05-12 00:00 local)
// fell outside the [2026-05-12 00:00 UTC, 2026-05-13 00:00 UTC) window and
// was silently dropped — users saw "saved successfully" then an empty list.
func TestGetHourly_LocalDateBoundary(t *testing.T) {
	// 2026-05-12 00:00:00 Asia/Tashkent == 2026-05-11 19:00:00 UTC.
	midnightLocalUTC := time.Date(2026, 5, 11, 19, 0, 0, 0, time.UTC)

	repo := &captureRepo{
		hourlyRangeResult: []model.HourlyRecord{
			{ID: 1, OrganizationID: 42, RecordedAt: midnightLocalUTC},
		},
	}

	// Asking for the day THE RECORD BELONGS TO (12-th in local TZ): the
	// handler MUST translate that to a UTC window that contains the record.
	rr := doRequestInLoc(t, repo, scClaims(), tashkentLoc, http.MethodGet,
		"/reservoir-flood/hourly?date=2026-05-12", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.HourlyRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("want 1 record for the local day the midnight belongs to, got %d", len(got))
	}

	// Verify the window the handler actually computed — defence against a
	// regression where the handler returns the right rows but for the wrong
	// reason (e.g. the mock filter happens to admit them by accident).
	wantStart := time.Date(2026, 5, 11, 19, 0, 0, 0, time.UTC) // 2026-05-12 00:00 +05:00
	wantEnd := time.Date(2026, 5, 12, 19, 0, 0, 0, time.UTC)   // 2026-05-13 00:00 +05:00
	if !repo.hourlyRangeStart.Equal(wantStart) {
		t.Errorf("window start: want %s, got %s", wantStart, repo.hourlyRangeStart)
	}
	if !repo.hourlyRangeEnd.Equal(wantEnd) {
		t.Errorf("window end: want %s, got %s", wantEnd, repo.hourlyRangeEnd)
	}

	// Counter-check: the previous local day (11-th) MUST NOT include this
	// record — its end boundary is the half-open exclusive 19:00 UTC, exactly
	// the record's timestamp.
	repo.hourlyRangeStart, repo.hourlyRangeEnd = time.Time{}, time.Time{}
	rr = doRequestInLoc(t, repo, scClaims(), tashkentLoc, http.MethodGet,
		"/reservoir-flood/hourly?date=2026-05-11", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}
	var prev []model.HourlyRecord
	_ = json.Unmarshal(rr.Body.Bytes(), &prev)
	if len(prev) != 0 {
		t.Errorf("previous local day must NOT include the midnight record, got %d", len(prev))
	}
}

// Optional ?hour= narrows the window to one local hour. With three records
// scattered across 2026-05-12 local (00:00, 08:00, 15:00), each hour query
// must match exactly the corresponding record (and no others), and the
// no-hour case must still return all three. A fresh captureRepo is built per
// subtest so window-tracking fields can't leak between iterations — keeps the
// table robust to reordering or future t.Parallel().
func TestGetHourly_HourFilter(t *testing.T) {
	hour00LocalUTC := time.Date(2026, 5, 11, 19, 0, 0, 0, time.UTC) // 12-th 00:00 local
	hour08LocalUTC := time.Date(2026, 5, 12, 3, 0, 0, 0, time.UTC)  // 12-th 08:00 local
	hour15LocalUTC := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC) // 12-th 15:00 local

	fixture := []model.HourlyRecord{
		{ID: 1, OrganizationID: 42, RecordedAt: hour00LocalUTC},
		{ID: 2, OrganizationID: 42, RecordedAt: hour08LocalUTC},
		{ID: 3, OrganizationID: 42, RecordedAt: hour15LocalUTC},
	}

	cases := []struct {
		name      string
		query     string
		wantLen   int
		wantID    int64
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name: "hour 0 picks midnight", query: "?date=2026-05-12&hour=0",
			wantLen: 1, wantID: 1,
			wantStart: hour00LocalUTC, wantEnd: hour00LocalUTC.Add(time.Hour),
		},
		{
			name: "hour 8 picks the morning record", query: "?date=2026-05-12&hour=8",
			wantLen: 1, wantID: 2,
			wantStart: hour08LocalUTC, wantEnd: hour08LocalUTC.Add(time.Hour),
		},
		{
			name: "hour 12 returns nothing", query: "?date=2026-05-12&hour=12",
			wantLen:   0,
			wantStart: time.Date(2026, 5, 12, 7, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 12, 8, 0, 0, 0, time.UTC),
		},
		{
			name: "no hour returns the full local day", query: "?date=2026-05-12",
			wantLen:   3,
			wantStart: time.Date(2026, 5, 11, 19, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 12, 19, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &captureRepo{hourlyRangeResult: fixture}
			rr := doRequestInLoc(t, repo, scClaims(), tashkentLoc, http.MethodGet,
				"/reservoir-flood/hourly"+tc.query, "")
			if rr.Code != http.StatusOK {
				t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
			}
			var got []model.HourlyRecord
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len: want %d, got %d", tc.wantLen, len(got))
			}
			if tc.wantLen == 1 && len(got) == 1 && got[0].ID != tc.wantID {
				t.Errorf("ID: want %d, got %d", tc.wantID, got[0].ID)
			}
			if !repo.hourlyRangeStart.Equal(tc.wantStart) {
				t.Errorf("window start: want %s, got %s", tc.wantStart, repo.hourlyRangeStart)
			}
			if !repo.hourlyRangeEnd.Equal(tc.wantEnd) {
				t.Errorf("window end: want %s, got %s", tc.wantEnd, repo.hourlyRangeEnd)
			}
		})
	}
}

// Bad ?hour= values must yield 400 BEFORE the repo is touched. Asserting on
// hourlyRangeCallCount (not on hourlyRangeStart.IsZero) catches a hypothetical
// regression where the handler calls the repo and then ignores the error —
// the start field would still be populated by the mock, but the call count
// would be 1 instead of 0.
func TestGetHourly_BadHour(t *testing.T) {
	cases := []struct {
		name, query string
	}{
		{"hour 24 out of range", "?date=2026-05-12&hour=24"},
		{"hour -1 out of range", "?date=2026-05-12&hour=-1"},
		{"hour abc not an integer", "?date=2026-05-12&hour=abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &captureRepo{}
			rr := doRequestInLoc(t, repo, scClaims(), tashkentLoc, http.MethodGet,
				"/reservoir-flood/hourly"+tc.query, "")
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
			}
			if repo.hourlyRangeCallCount != 0 {
				t.Errorf("repo MUST NOT be called on bad hour; got %d calls", repo.hourlyRangeCallCount)
			}
		})
	}
}

// Sanity: strconv.Atoi accepts leading zeros, so ?hour=08 is equivalent to
// ?hour=8. Frontend may send either depending on whether it formats local
// hour-of-day as zero-padded.
func TestGetHourly_HourWithLeadingZero(t *testing.T) {
	rec := time.Date(2026, 5, 12, 3, 0, 0, 0, time.UTC) // 12-th 08:00 local
	repo := &captureRepo{
		hourlyRangeResult: []model.HourlyRecord{
			{ID: 99, OrganizationID: 42, RecordedAt: rec},
		},
	}
	rr := doRequestInLoc(t, repo, scClaims(), tashkentLoc, http.MethodGet,
		"/reservoir-flood/hourly?date=2026-05-12&hour=08", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var got []model.HourlyRecord
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if len(got) != 1 || got[0].ID != 99 {
		t.Errorf("hour=08 should match the 08:00-local record; got %+v", got)
	}
}
