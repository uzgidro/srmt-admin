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
	knownCascades map[int64]bool // org IDs that GetCascadeConfigByOrgID will treat as valid cascades
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

// captureCascadeWeatherGetter satisfies CascadeDailyWeatherGetter.
type captureCascadeWeatherGetter struct {
	result *model.CascadeWeather
	err    error
}

func (c *captureCascadeWeatherGetter) GetCascadeDailyWeather(_ context.Context, _ int64, _ string) (*model.CascadeWeather, error) {
	return c.result, c.err
}

func setupCascadeWeatherPOSTRouter(upserter *captureCascadeWeatherUpserter) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Post("/cascade-daily-data", UpsertCascadeDailyWeather(logger, upserter))
	return r
}

func setupCascadeWeatherGETRouter(getter *captureCascadeWeatherGetter) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}}
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
