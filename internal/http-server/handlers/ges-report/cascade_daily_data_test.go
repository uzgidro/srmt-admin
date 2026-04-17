package gesreport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

// captureCascadeWeatherUpserter satisfies CascadeDailyWeatherUpserter and
// records the slice passed to UpsertCascadeDailyWeatherBulk.
type captureCascadeWeatherUpserter struct {
	mu            sync.Mutex
	lastItems     []model.UpsertCascadeDailyWeatherRequest
	knownCascades map[int64]bool   // org IDs that GetCascadeConfigByOrgID will treat as valid cascades
	parents       map[int64]*int64 // optional: per-org parent overrides for cascade access tests
	upsertErr     error
}

func (c *captureCascadeWeatherUpserter) GetCascadeConfigByOrgID(_ context.Context, orgID int64) (*model.CascadeConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.knownCascades[orgID] {
		return nil, storage.ErrNotFound
	}
	return &model.CascadeConfig{
		ID:             1,
		OrganizationID: orgID,
		SortOrder:      0,
	}, nil
}

func (c *captureCascadeWeatherUpserter) UpsertCascadeDailyWeatherBulk(_ context.Context, items []model.UpsertCascadeDailyWeatherRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastItems = make([]model.UpsertCascadeDailyWeatherRequest, len(items))
	copy(c.lastItems, items)
	return c.upsertErr
}

// GetOrganizationParentID is required by CheckCascadeStationAccess. With no
// parents map configured (sc/rais tests) the lookup returns nil/nil — matching
// the production behaviour for an org with no parent — but those tests never
// reach this branch because sc/rais get a fast-path inside the auth helper.
func (c *captureCascadeWeatherUpserter) GetOrganizationParentID(_ context.Context, orgID int64) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parents == nil {
		return nil, nil
	}
	return c.parents[orgID], nil
}

// captureCascadeWeatherGetter satisfies CascadeDailyWeatherGetter.
type captureCascadeWeatherGetter struct {
	mu      sync.Mutex
	result  *model.CascadeWeather
	err     error
	parents map[int64]*int64 // optional: per-org parent overrides for cascade access tests
}

func (c *captureCascadeWeatherGetter) GetCascadeDailyWeather(_ context.Context, _ int64, _ string) (*model.CascadeWeather, error) {
	return c.result, c.err
}

func (c *captureCascadeWeatherGetter) GetOrganizationParentID(_ context.Context, orgID int64) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parents == nil {
		return nil, nil
	}
	return c.parents[orgID], nil
}

func setupCascadeWeatherPOSTRouter(upserter *captureCascadeWeatherUpserter) http.Handler {
	return setupCascadeWeatherPOSTRouterWithClaims(upserter, &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	})
}

func setupCascadeWeatherPOSTRouterWithClaims(upserter *captureCascadeWeatherUpserter, claims *token.Claims) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Post("/cascade-daily-data", UpsertCascadeDailyWeather(logger, upserter))
	return r
}

func setupCascadeWeatherGETRouter(getter *captureCascadeWeatherGetter) http.Handler {
	return setupCascadeWeatherGETRouterWithClaims(getter, &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	})
}

func setupCascadeWeatherGETRouterWithClaims(getter *captureCascadeWeatherGetter, claims *token.Claims) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Get("/cascade-daily-data", GetCascadeDailyWeather(logger, getter))
	return r
}

func doCascadeWeatherPOST(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/cascade-daily-data", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer faketoken")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func doCascadeWeatherGET(t *testing.T, h http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/cascade-daily-data?"+query, nil)
	req.Header.Set("Authorization", "Bearer faketoken")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// --- POST tests ---

func TestUpsertCascadeDailyWeather_AllNumbers(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[{"organization_id":10,"date":"2026-04-13","temperature":22.5,"weather_condition":"01d"}]`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if len(upserter.lastItems) != 1 {
		t.Fatalf("items count: got %d, want 1", len(upserter.lastItems))
	}
	item := upserter.lastItems[0]
	if !item.Temperature.Set || item.Temperature.Value == nil || *item.Temperature.Value != 22.5 {
		t.Errorf("temperature: got Set=%v Value=%v", item.Temperature.Set, item.Temperature.Value)
	}
	if !item.WeatherCondition.Set || item.WeatherCondition.Value == nil || *item.WeatherCondition.Value != "01d" {
		t.Errorf("weather_condition: got Set=%v Value=%v", item.WeatherCondition.Set, item.WeatherCondition.Value)
	}
}

func TestUpsertCascadeDailyWeather_PartialTempOnly(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[{"organization_id":10,"date":"2026-04-13","temperature":15.0}]`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	item := upserter.lastItems[0]
	if !item.Temperature.Set || item.Temperature.Value == nil || *item.Temperature.Value != 15.0 {
		t.Errorf("temperature should be set to 15.0")
	}
	if item.WeatherCondition.Set {
		t.Errorf("weather_condition should NOT be set (absent from JSON)")
	}
}

func TestUpsertCascadeDailyWeather_PartialConditionOnly(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[{"organization_id":10,"date":"2026-04-13","weather_condition":"10n"}]`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	item := upserter.lastItems[0]
	if item.Temperature.Set {
		t.Errorf("temperature should NOT be set (absent)")
	}
	if !item.WeatherCondition.Set || item.WeatherCondition.Value == nil || *item.WeatherCondition.Value != "10n" {
		t.Errorf("weather_condition should be set to 10n")
	}
}

func TestUpsertCascadeDailyWeather_ExplicitNulls(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[{"organization_id":10,"date":"2026-04-13","temperature":null,"weather_condition":null}]`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	item := upserter.lastItems[0]
	if !item.Temperature.Set || item.Temperature.Value != nil {
		t.Errorf("temperature: expected Set=true Value=nil")
	}
	if !item.WeatherCondition.Set || item.WeatherCondition.Value != nil {
		t.Errorf("weather_condition: expected Set=true Value=nil")
	}
}

func TestUpsertCascadeDailyWeather_AcceptsArray(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true, 20: true}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[
		{"organization_id":10,"date":"2026-04-13","temperature":23.0,"weather_condition":"01d"},
		{"organization_id":20,"date":"2026-04-13","temperature":18.5,"weather_condition":"10d"}
	]`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if len(upserter.lastItems) != 2 {
		t.Fatalf("items: got %d, want 2", len(upserter.lastItems))
	}
}

func TestUpsertCascadeDailyWeather_EmptyArray(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[]`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rr.Code)
	}
}

func TestUpsertCascadeDailyWeather_NotACascade(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true}} // only org 10 is a cascade
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[{"organization_id":999,"date":"2026-04-13","temperature":20.0}]`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "item_index") {
		t.Errorf("body should mention item_index: %s", rr.Body.String())
	}
	if len(upserter.lastItems) != 0 {
		t.Errorf("upsert should not be called when validation fails")
	}
}

func TestUpsertCascadeDailyWeather_ItemIndexInError(t *testing.T) {
	upserter := &captureCascadeWeatherUpserter{knownCascades: map[int64]bool{10: true, 20: true}}
	h := setupCascadeWeatherPOSTRouter(upserter)
	rr := doCascadeWeatherPOST(t, h, `[
		{"organization_id":10,"date":"2026-04-13","temperature":22.0},
		{"organization_id":20,"date":"not-a-date","temperature":18.0}
	]`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	idx, ok := body["item_index"]
	if !ok {
		t.Fatal("body missing item_index")
	}
	// JSON numbers decode to float64
	if idx.(float64) != 1 {
		t.Errorf("item_index: got %v, want 1", idx)
	}
}

// --- GET tests ---

func TestGetCascadeDailyWeather_Found(t *testing.T) {
	temp := 22.5
	cond := "01d"
	getter := &captureCascadeWeatherGetter{
		result: &model.CascadeWeather{Temperature: &temp, Condition: &cond},
	}
	h := setupCascadeWeatherGETRouter(getter)
	rr := doCascadeWeatherGET(t, h, "organization_id=1&date=2026-04-13")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v. body=%s", err, rr.Body.String())
	}
	if body["temperature"].(float64) != 22.5 {
		t.Errorf("temperature: got %v, want 22.5", body["temperature"])
	}
	if body["weather_condition"].(string) != "01d" {
		t.Errorf("weather_condition: got %v, want 01d", body["weather_condition"])
	}
}

func TestGetCascadeDailyWeather_NotFound(t *testing.T) {
	getter := &captureCascadeWeatherGetter{err: storage.ErrNotFound}
	h := setupCascadeWeatherGETRouter(getter)
	rr := doCascadeWeatherGET(t, h, "organization_id=1&date=2026-04-13")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (null body). body=%s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "null" {
		t.Errorf("body: got %q, want \"null\"", rr.Body.String())
	}
}

// --- cascade-role access tests ---

// A cascade user editing weather for its OWN cascade org (orgID == claims.OrganizationID)
// must succeed: the cascade self-org case is handled inside CheckCascadeStationAccess
// without consulting the parent map.
func TestUpsertCascadeDailyWeather_CascadeOwnCascade_OK(t *testing.T) {
	const cascadeOrgID int64 = 50

	upserter := &captureCascadeWeatherUpserter{
		knownCascades: map[int64]bool{cascadeOrgID: true},
	}
	claims := &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	h := setupCascadeWeatherPOSTRouterWithClaims(upserter, claims)
	body := `[{"organization_id":50,"date":"2026-04-13","temperature":22.5,"weather_condition":"01d"}]`
	rr := doCascadeWeatherPOST(t, h, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if len(upserter.lastItems) != 1 {
		t.Fatalf("upserter should be called once; got %d items", len(upserter.lastItems))
	}
}

// A cascade user must NOT be able to edit weather for a foreign cascade org
// (one whose ID differs from claims.OrganizationID and whose parent_org_id is
// also not the cascade's own). The repo write must not be invoked.
func TestUpsertCascadeDailyWeather_CascadeForeignCascade_403(t *testing.T) {
	const cascadeOrgID int64 = 50
	const foreignCascadeID int64 = 60

	upserter := &captureCascadeWeatherUpserter{
		knownCascades: map[int64]bool{foreignCascadeID: true},
		parents: map[int64]*int64{
			// foreign cascade is its own root: parent is nil → no access.
			foreignCascadeID: nil,
		},
	}
	claims := &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	h := setupCascadeWeatherPOSTRouterWithClaims(upserter, claims)
	body := `[{"organization_id":60,"date":"2026-04-13","temperature":22.5}]`
	rr := doCascadeWeatherPOST(t, h, body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403. body=%s", rr.Code, rr.Body.String())
	}
	if len(upserter.lastItems) != 0 {
		t.Errorf("upserter must NOT be called when access is denied; got %d items", len(upserter.lastItems))
	}
}

// A cascade user reading weather for its OWN cascade org succeeds.
func TestGetCascadeDailyWeather_CascadeOwn_OK(t *testing.T) {
	const cascadeOrgID int64 = 50

	temp := 19.0
	getter := &captureCascadeWeatherGetter{
		result: &model.CascadeWeather{Temperature: &temp},
	}
	claims := &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	h := setupCascadeWeatherGETRouterWithClaims(getter, claims)
	rr := doCascadeWeatherGET(t, h, "organization_id=50&date=2026-04-13")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
}

// A cascade user reading weather for a foreign cascade org is forbidden.
func TestGetCascadeDailyWeather_CascadeForeign_403(t *testing.T) {
	const cascadeOrgID int64 = 50
	const foreignCascadeID int64 = 60

	temp := 19.0
	getter := &captureCascadeWeatherGetter{
		result: &model.CascadeWeather{Temperature: &temp},
		parents: map[int64]*int64{
			foreignCascadeID: nil, // foreign cascade has no parent — definitely not cascadeOrgID
		},
	}
	claims := &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	h := setupCascadeWeatherGETRouterWithClaims(getter, claims)
	rr := doCascadeWeatherGET(t, h, "organization_id=60&date=2026-04-13")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403. body=%s", rr.Code, rr.Body.String())
	}
}
