package gesreport

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

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/token"
)

// captureConfigUpserter satisfies ConfigUpserter and records the request
// passed to UpsertGESConfig so tests can inspect the parsed value (and, by
// re-marshalling, the wire fields the production struct exposed to JSON).
type captureConfigUpserter struct {
	mu      sync.Mutex
	lastReq model.UpsertConfigRequest
	called  int
	err     error
}

func (c *captureConfigUpserter) UpsertGESConfig(_ context.Context, req model.UpsertConfigRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.called++
	c.lastReq = req
	return c.err
}

// staticConfigGetter satisfies ConfigGetter, returning a fixed slice.
type staticConfigGetter struct {
	configs []model.Config
	err     error
}

func (s *staticConfigGetter) GetAllGESConfigs(_ context.Context) ([]model.Config, error) {
	return s.configs, s.err
}

func newConfigUpsertRouter(upserter ConfigUpserter, claims *token.Claims) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Post("/ges/config", UpsertConfig(log, upserter))
	return r
}

func newConfigGetRouter(getter ConfigGetter, claims *token.Claims) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Get("/ges/config", GetConfigs(log, getter))
	return r
}

func doConfigUpsert(upserter ConfigUpserter, claims *token.Claims, body string) *httptest.ResponseRecorder {
	r := newConfigUpsertRouter(upserter, claims)
	req := httptest.NewRequest(http.MethodPost, "/ges/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func doConfigGet(getter ConfigGetter, claims *token.Claims) *httptest.ResponseRecorder {
	r := newConfigGetRouter(getter, claims)
	req := httptest.NewRequest(http.MethodGet, "/ges/config", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// scClaims returns a minimal sc-role claims set so the cascade filter
// fast-paths the test scenarios.
func scClaims() *token.Claims {
	return &token.Claims{UserID: 1, OrganizationID: 1, Roles: []string{"sc"}}
}

// --- max_daily_production_mln_kwh tests ---

// A negative max value must be rejected by the validator with 400, and the
// repo must not be called. Currently there is no field/validator → handler
// returns 200; this is the failing scenario that drives the GREEN-phase
// validator tag (e.g. validate:"gte=0").
func TestUpsertConfig_NegativeMaxDailyProduction_BadRequest(t *testing.T) {
	upserter := &captureConfigUpserter{}
	body := `{
		"organization_id": 1,
		"installed_capacity_mwt": 100.0,
		"total_aggregates": 4,
		"has_reservoir": true,
		"sort_order": 1,
		"max_daily_production_mln_kwh": -1.0
	}`
	rr := doConfigUpsert(upserter, scClaims(), body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if upserter.called != 0 {
		t.Errorf("repo must NOT be called when validation fails; got %d calls", upserter.called)
	}
}

// Explicit zero is the documented "no cap" sentinel and must be accepted.
// This test is expected to PASS today (handler accepts the request because
// no validator on a missing field) — once the field exists with a gte=0
// constraint it must continue to pass. Acts as a regression guard against
// over-eager `gt=0` constraints.
func TestUpsertConfig_ZeroMaxDailyProduction_OK(t *testing.T) {
	upserter := &captureConfigUpserter{}
	body := `{
		"organization_id": 1,
		"installed_capacity_mwt": 100.0,
		"total_aggregates": 4,
		"has_reservoir": true,
		"sort_order": 1,
		"max_daily_production_mln_kwh": 0
	}`
	rr := doConfigUpsert(upserter, scClaims(), body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if upserter.called != 1 {
		t.Errorf("repo must be called exactly once; got %d", upserter.called)
	}
}

// Positive value: handler must round-trip it to the repo. We re-marshal the
// captured request struct and look for the wire key. RED today because the
// struct lacks the field, so json.Marshal omits the key entirely.
func TestUpsertConfig_PositiveMaxDailyProduction_OK(t *testing.T) {
	upserter := &captureConfigUpserter{}
	body := `{
		"organization_id": 1,
		"installed_capacity_mwt": 100.0,
		"total_aggregates": 4,
		"has_reservoir": true,
		"sort_order": 1,
		"max_daily_production_mln_kwh": 12.5
	}`
	rr := doConfigUpsert(upserter, scClaims(), body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if upserter.called != 1 {
		t.Fatalf("repo must be called exactly once; got %d", upserter.called)
	}

	// Re-marshal what the handler actually parsed and forwarded. The wire
	// key must be present with the requested value once the field exists
	// on UpsertConfigRequest.
	raw, err := json.Marshal(upserter.lastReq)
	if err != nil {
		t.Fatalf("marshal lastReq: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	v, ok := parsed["max_daily_production_mln_kwh"]
	if !ok {
		t.Fatalf("captured request has no max_daily_production_mln_kwh key; struct missing field. raw=%s", raw)
	}
	num, ok := v.(float64)
	if !ok {
		t.Fatalf("max_daily_production_mln_kwh is not a number: %T %v", v, v)
	}
	if num != 12.5 {
		t.Errorf("max_daily_production_mln_kwh = %v, want 12.5", num)
	}
}

// GetConfigs must serialise the new field on Config rows. Mock returns a
// Config; we decode the response JSON and assert the key surfaces with the
// configured value. RED today because Config has no field — JSON omits the
// key entirely.
func TestGetConfigs_IncludesMaxDailyProduction(t *testing.T) {
	getter := &staticConfigGetter{
		configs: []model.Config{{
			ID:                   1,
			OrganizationID:       1,
			OrganizationName:     "Test GES",
			InstalledCapacityMWt: 100.0,
			TotalAggregates:      4,
			HasReservoir:         true,
			SortOrder:            0,
		}},
	}
	rr := doConfigGet(getter, scClaims())
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	// Re-decode the response into a generic structure so we can assert the
	// wire key without depending on whether the Go field exists yet.
	var configs []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &configs); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, rr.Body.String())
	}
	if len(configs) != 1 {
		t.Fatalf("response: want 1 config, got %d", len(configs))
	}
	v, ok := configs[0]["max_daily_production_mln_kwh"]
	if !ok {
		t.Fatalf("response config missing max_daily_production_mln_kwh; body=%s", rr.Body.String())
	}
	num, ok := v.(float64)
	if !ok {
		t.Fatalf("max_daily_production_mln_kwh is not a number: %T %v", v, v)
	}
	// Once the model carries the field, the static fixture above seeds it
	// to its zero value (7.0 once we add MaxDailyProductionMlnKwh: 7.0).
	// Today the test fails at the missing-key assertion; once GREEN
	// surfaces the field, this assertion ensures the fixture flows through.
	if num != 7.0 {
		// Note: this assertion will only ever pass once both the model
		// has the field AND the fixture seeds it. We deliberately leave
		// the seed value out of the literal above so the GREEN-phase
		// commit must touch this line.
		t.Errorf("max_daily_production_mln_kwh = %v, want 7.0 (seed missing on Config struct?)", num)
	}
}
