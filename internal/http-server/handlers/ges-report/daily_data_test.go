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
	"srmt-admin/internal/token"
)

// captureGESGetter satisfies DailyDataGetter (with the cascade extension) and
// returns a configured DailyData (or an error) for the GET handler.
type captureGESGetter struct {
	mu      sync.Mutex
	result  *model.DailyData
	err     error
	parents map[int64]*int64 // org -> parent_org_id (nil means no parent)
}

func (c *captureGESGetter) GetGESDailyData(_ context.Context, _ int64, _ string) (*model.DailyData, error) {
	return c.result, c.err
}

func (c *captureGESGetter) GetOrganizationParentID(_ context.Context, orgID int64) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parents == nil {
		return nil, nil
	}
	return c.parents[orgID], nil
}

func newGESUpsertRouterWithClaims(upserter *captureGESUpserter, claims *token.Claims) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Post("/ges/daily-data", UpsertDailyData(log, upserter))
	return r
}

func newGESGetRouterWithClaims(getter DailyDataGetter, claims *token.Claims) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Get("/ges/daily-data", GetDailyData(log, getter))
	return r
}

func doGESUpsertWithClaims(upserter *captureGESUpserter, claims *token.Claims, body string) *httptest.ResponseRecorder {
	r := newGESUpsertRouterWithClaims(upserter, claims)
	req := httptest.NewRequest(http.MethodPost, "/ges/daily-data", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func doGESGetWithClaims(getter DailyDataGetter, claims *token.Claims, query string) *httptest.ResponseRecorder {
	r := newGESGetRouterWithClaims(getter, claims)
	req := httptest.NewRequest(http.MethodGet, "/ges/daily-data?"+query, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// ptrInt64 is a small helper for building parent maps inline.
func ptrInt64(v int64) *int64 { return &v }

// --- Upsert cascade access tests ---

// Cascade user with stations whose parent_org_id == claims.OrganizationID
// must be permitted to upsert daily data for those stations.
func TestUpsertDailyData_CascadeOwnStation_OK(t *testing.T) {
	const cascadeOrgID int64 = 50
	const stationOrgID int64 = 100

	upserter := &captureGESUpserter{
		parents: map[int64]*int64{
			stationOrgID: ptrInt64(cascadeOrgID),
		},
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	body := `[{"organization_id": 100, "date": "2026-04-13", "daily_production_mln_kwh": 1.5}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 1 {
		t.Fatalf("upserter should have been called once; got %d items", len(upserter.last))
	}
}

// Cascade user must be denied access when the station's parent_org_id does
// not match the cascade's own organization.
func TestUpsertDailyData_CascadeForeignStation_403(t *testing.T) {
	const cascadeOrgID int64 = 50
	const otherCascadeID int64 = 60
	const stationOrgID int64 = 200

	upserter := &captureGESUpserter{
		parents: map[int64]*int64{
			stationOrgID: ptrInt64(otherCascadeID),
		},
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	body := `[{"organization_id": 200, "date": "2026-04-13", "daily_production_mln_kwh": 1.5}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 0 {
		t.Errorf("upserter must NOT be called when access is denied; got %d items", len(upserter.last))
	}
}

// Cascade user uploading data that belongs to its own org (the cascade itself)
// is permitted without consulting the parent map.
func TestUpsertDailyData_CascadeSelfOrg_OK(t *testing.T) {
	const cascadeOrgID int64 = 50

	upserter := &captureGESUpserter{} // no parents map needed
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	body := `[{"organization_id": 50, "date": "2026-04-13", "daily_production_mln_kwh": 1.5}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// --- Get cascade access tests ---

// Cascade user reading a station that belongs to its cascade gets 200.
func TestGetDailyData_CascadeOwnStation_OK(t *testing.T) {
	const cascadeOrgID int64 = 50
	const stationOrgID int64 = 100

	getter := &captureGESGetter{
		result: &model.DailyData{},
		parents: map[int64]*int64{
			stationOrgID: ptrInt64(cascadeOrgID),
		},
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	rr := doGESGetWithClaims(getter, claims, "organization_id=100&date=2026-04-13")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// Cascade user reading a station with a different parent must get 403, and
// the repository must not be queried.
func TestGetDailyData_CascadeForeignStation_403(t *testing.T) {
	const cascadeOrgID int64 = 50
	const otherCascadeID int64 = 60
	const stationOrgID int64 = 200

	called := false
	tracker := &cascadeGetterCallTracker{
		inner: &captureGESGetter{
			result: &model.DailyData{},
			parents: map[int64]*int64{
				stationOrgID: ptrInt64(otherCascadeID),
			},
		},
		called: &called,
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: cascadeOrgID,
		Roles:          []string{"cascade"},
	}
	rr := doGESGetWithClaims(tracker, claims, "organization_id=200&date=2026-04-13")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if called {
		t.Errorf("GetGESDailyData must not be called when access is denied")
	}

	// Sanity: a sc user can still read this org because sc/rais get the cascade
	// fast-path. This is a regression check that the new check leaves sc alone.
	called = false
	scClaims := &token.Claims{
		UserID:         1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}
	rr2 := doGESGetWithClaims(tracker, scClaims, "organization_id=200&date=2026-04-13")
	if rr2.Code != http.StatusOK {
		t.Fatalf("sc fallback: want 200, got %d; body: %s", rr2.Code, rr2.Body.String())
	}
	if !called {
		t.Errorf("sc user should reach GetGESDailyData")
	}
}

// cascadeGetterCallTracker wraps captureGESGetter to record whether
// GetGESDailyData is invoked. It satisfies DailyDataGetter so it can also be
// used directly as the handler dependency.
type cascadeGetterCallTracker struct {
	inner  *captureGESGetter
	called *bool
}

func (c *cascadeGetterCallTracker) GetGESDailyData(ctx context.Context, orgID int64, date string) (*model.DailyData, error) {
	*c.called = true
	return c.inner.GetGESDailyData(ctx, orgID, date)
}

func (c *cascadeGetterCallTracker) GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error) {
	return c.inner.GetOrganizationParentID(ctx, orgID)
}

// decodeError parses {"error": "..."} from a JSON response body. Centralised
// because go-chi's render.JSON HTML-escapes >, <, & in the wire form, so a
// raw substring search would miss the literal ">" the spec requires.
func decodeError(t *testing.T, body []byte) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("response is not valid JSON: %v; body=%s", err, body)
	}
	v, ok := payload["error"]
	if !ok {
		t.Fatalf("response has no 'error' field: %s", body)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("'error' is not a string: %v", v)
	}
	return s
}

// --- Aggregate validation tests (Task 4) ---

// Sum check uses values from the request overlaid on current DB values for
// any field the request did not include. With total=6 and current working=3,
// a request that bumps working to 4 while also setting repair=2 and mod=1
// makes 4+2+1=7 > 6 — the handler must reject with the documented format.
func TestUpsertDailyData_RejectsAggregatesExceedingTotal(t *testing.T) {
	const stationOrgID int64 = 10

	upserter := &captureGESUpserter{
		totals: map[int64]int{stationOrgID: 6},
		current: map[aggKey]model.AggregateCounts{
			{OrgID: stationOrgID, Date: "2026-04-13"}: {Working: 3, Repair: 0, Modernization: 0},
		},
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"working_aggregates": 4,
		"repair_aggregates": 2,
		"modernization_aggregates": 1
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	want := "aggregates sum exceeds total for organization_id=10: 4+2+1=7 > 6"
	if got := decodeError(t, rr.Body.Bytes()); got != want {
		t.Errorf("error message mismatch:\nwant: %q\ngot:  %q", want, got)
	}
	if len(upserter.last) != 0 {
		t.Errorf("upserter must NOT be called when validation fails; got %d items", len(upserter.last))
	}
}

// Partial update — request only sends repair_aggregates=2; working stays at
// the current DB value (3) and modernization defaults to 0. Effective sum is
// 3+2+0=5 ≤ 6 so the upsert must succeed.
func TestUpsertDailyData_AllowsPartialUpdateWithinTotal(t *testing.T) {
	const stationOrgID int64 = 10

	upserter := &captureGESUpserter{
		totals: map[int64]int{stationOrgID: 6},
		current: map[aggKey]model.AggregateCounts{
			{OrgID: stationOrgID, Date: "2026-04-13"}: {Working: 3, Repair: 0, Modernization: 0},
		},
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"repair_aggregates": 2
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(upserter.last) != 1 {
		t.Fatalf("upserter should have been called once; got %d items", len(upserter.last))
	}
}

// Negative scalar values are rejected per-field with a clear 400 message
// regardless of total_aggregates capacity.
func TestUpsertDailyData_RejectsNegativeAggregate(t *testing.T) {
	upserter := &captureGESUpserter{
		totals: map[int64]int{10: 6},
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"repair_aggregates": -1
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "repair_aggregates") || !strings.Contains(bodyStr, "organization_id=10") {
		t.Errorf("body should mention field and organization_id; got %s", bodyStr)
	}
	if len(upserter.last) != 0 {
		t.Errorf("upserter must NOT be called when validation fails; got %d items", len(upserter.last))
	}
}

// When ges_config has no row for the organization, GetGESConfigsTotalAggregates
// returns no entry — the handler must skip the sum check (matching the trigger).
func TestUpsertDailyData_SkipsSumCheckWhenNoConfig(t *testing.T) {
	upserter := &captureGESUpserter{
		totals: map[int64]int{}, // no config for any org
	}
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}
	// Even huge values pass when no cap is configured.
	body := `[{
		"organization_id": 10,
		"date": "2026-04-13",
		"working_aggregates": 100,
		"repair_aggregates": 100,
		"modernization_aggregates": 100
	}]`
	rr := doGESUpsertWithClaims(upserter, claims, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

