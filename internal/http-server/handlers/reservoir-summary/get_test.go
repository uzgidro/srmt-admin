package reservoirsummary

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/dto"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

type mockSummaryGetter struct {
	summaries []*reservoirsummary.ResponseModel
	err       error

	// Curve lookup for level → volume recomputation. Optional: if curveErr/curveVolume
	// stay zero-valued, behaves as "no curve configured".
	curveCalls  []curveCall
	curveVolume float64
	curveErr    error

	// Reservoir-summary config — drives both modsnow_enabled masking and the
	// volume_source strategy switch in applyStaticFallbacks. Default empty
	// slice = no config row for any org (legacy behaviour: Modsnow untouched,
	// volume resolution falls back to the "static" path). configsErr lets a
	// test exercise the degraded-on-fetch-failure path explicitly.
	configs    []reservoirsummary.ReservoirSummaryConfig
	configsErr error
}

func (m *mockSummaryGetter) GetReservoirSummary(_ context.Context, _ string) ([]*reservoirsummary.ResponseModel, error) {
	return m.summaries, m.err
}

func (m *mockSummaryGetter) GetVolumeByLevelByOrg(_ context.Context, orgID int64, level float64) (float64, error) {
	m.curveCalls = append(m.curveCalls, curveCall{orgID: orgID, level: level})
	return m.curveVolume, m.curveErr
}

func (m *mockSummaryGetter) GetAllReservoirSummaryConfigs(_ context.Context) ([]reservoirsummary.ReservoirSummaryConfig, error) {
	return m.configs, m.configsErr
}

type mockStaticDataFetcher struct {
	data map[int64]*dto.OrganizationWithData
	err  error
}

func (m *mockStaticDataFetcher) FetchDataAtDayBegin(_ context.Context, _ string) (map[int64]*dto.OrganizationWithData, error) {
	return m.data, m.err
}

func ptrFloat(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64     { return &v }

func TestGet_NoAvg_ShouldNotFallbackToCurrentValues(t *testing.T) {
	orgID := int64(1)
	income := 100.0
	release := 50.0

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Income:         reservoirsummary.ValueResponse{Current: 0},
				Release:        reservoirsummary.ValueResponse{Current: 0},
			},
		},
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Income:     &income,
					Release:    &release,
					AvgIncome:  nil, // no avg
					AvgRelease: nil, // no avg
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}

	// Without averages, Income and Release should remain 0 (no fallback to current values)
	if result[0].Income.Current != 0 {
		t.Errorf("Income.Current: expected 0 (no fallback), got %f", result[0].Income.Current)
	}
	if result[0].Release.Current != 0 {
		t.Errorf("Release.Current: expected 0 (no fallback), got %f", result[0].Release.Current)
	}
	if result[0].Income.IsEdited != nil {
		t.Errorf("Income.IsEdited: expected nil, got %v", *result[0].Income.IsEdited)
	}
	if result[0].Release.IsEdited != nil {
		t.Errorf("Release.IsEdited: expected nil, got %v", *result[0].Release.IsEdited)
	}
}

func TestGet_WithAvg_ShouldUseAvgValues(t *testing.T) {
	orgID := int64(1)
	avgIncome := 80.0
	avgRelease := 40.0

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Income:         reservoirsummary.ValueResponse{Current: 0},
				Release:        reservoirsummary.ValueResponse{Current: 0},
			},
		},
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Income:     ptrFloat(100.0),
					Release:    ptrFloat(50.0),
					AvgIncome:  &avgIncome,
					AvgRelease: &avgRelease,
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result[0].Income.Current != avgIncome {
		t.Errorf("Income.Current: expected %f (avg), got %f", avgIncome, result[0].Income.Current)
	}
	if result[0].Release.Current != avgRelease {
		t.Errorf("Release.Current: expected %f (avg), got %f", avgRelease, result[0].Release.Current)
	}
}

func TestGet_VolumeRecomputedFromLevel(t *testing.T) {
	orgID := int64(96)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Level:          reservoirsummary.ValueResponse{Current: 200},
				Volume:         reservoirsummary.ValueResponse{Current: 0},
			},
		},
		curveVolume: 120.0,
	}

	fetcher := &mockStaticDataFetcher{data: map[int64]*dto.OrganizationWithData{}}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result[0].Volume.Current != 120.0 {
		t.Errorf("Volume.Current: expected 120 (computed from level), got %f", result[0].Volume.Current)
	}
	if result[0].Volume.IsEdited == nil || !*result[0].Volume.IsEdited {
		t.Errorf("Volume.IsEdited: expected true, got %v", result[0].Volume.IsEdited)
	}
	if len(getter.curveCalls) != 1 || getter.curveCalls[0].orgID != orgID || getter.curveCalls[0].level != 200 {
		t.Errorf("expected one curve call with orgID=%d, level=200; got %+v", orgID, getter.curveCalls)
	}
}

func TestGet_VolumeRecomputedFromStaticLevel(t *testing.T) {
	orgID := int64(96)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Level:          reservoirsummary.ValueResponse{Current: 0},
				Volume:         reservoirsummary.ValueResponse{Current: 0},
			},
		},
		curveVolume: 125.0,
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Level:  ptrFloat(205.0),
					Volume: ptrFloat(80.0), // would be the old fallback; should be ignored when curve succeeds
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result[0].Level.Current != 205.0 {
		t.Errorf("Level.Current: expected 205 (from static), got %f", result[0].Level.Current)
	}
	if result[0].Level.IsEdited == nil || !*result[0].Level.IsEdited {
		t.Errorf("Level.IsEdited: expected true, got %v", result[0].Level.IsEdited)
	}
	if result[0].Volume.Current != 125.0 {
		t.Errorf("Volume.Current: expected 125 (computed from static level), got %f", result[0].Volume.Current)
	}
	if result[0].Volume.IsEdited == nil || !*result[0].Volume.IsEdited {
		t.Errorf("Volume.IsEdited: expected true, got %v", result[0].Volume.IsEdited)
	}
	if len(getter.curveCalls) != 1 || getter.curveCalls[0].level != 205 {
		t.Errorf("expected one curve call with level=205; got %+v", getter.curveCalls)
	}
}

func TestGet_FallbackToStaticVolumeWhenCurveNotConfigured(t *testing.T) {
	orgID := int64(99)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Level:          reservoirsummary.ValueResponse{Current: 200},
				Volume:         reservoirsummary.ValueResponse{Current: 0},
			},
		},
		curveErr: storage.ErrLevelVolumeNotConfigured,
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Volume: ptrFloat(80.0),
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result[0].Volume.Current != 80.0 {
		t.Errorf("Volume.Current: expected 80 (static fallback), got %f", result[0].Volume.Current)
	}
	if result[0].Volume.IsEdited == nil || !*result[0].Volume.IsEdited {
		t.Errorf("Volume.IsEdited: expected true, got %v", result[0].Volume.IsEdited)
	}
}

func TestGet_NoFallbackWhenDBVolumeNonZero(t *testing.T) {
	orgID := int64(96)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Level:          reservoirsummary.ValueResponse{Current: 200},
				Volume:         reservoirsummary.ValueResponse{Current: 200},
			},
		},
		curveVolume: 999.0, // would be applied if curve were called — but it shouldn't be
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					Volume: ptrFloat(80.0),
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result[0].Volume.Current != 200 {
		t.Errorf("Volume.Current: expected 200 (unchanged), got %f", result[0].Volume.Current)
	}
	if result[0].Volume.IsEdited != nil {
		t.Errorf("Volume.IsEdited: expected nil (untouched), got %v", *result[0].Volume.IsEdited)
	}
	if len(getter.curveCalls) != 0 {
		t.Errorf("expected no curve calls when DB Volume non-zero; got %+v", getter.curveCalls)
	}
}

func TestGet_AlreadyHasValues_ShouldNotOverwrite(t *testing.T) {
	orgID := int64(1)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Income:         reservoirsummary.ValueResponse{Current: 200},
				Release:        reservoirsummary.ValueResponse{Current: 150},
			},
		},
	}

	fetcher := &mockStaticDataFetcher{
		data: map[int64]*dto.OrganizationWithData{
			orgID: {
				OrganizationID: orgID,
				Data: &dto.ReservoirData{
					AvgIncome:  ptrFloat(80.0),
					AvgRelease: ptrFloat(40.0),
				},
			},
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var result []*reservoirsummary.ResponseModel
	json.NewDecoder(rr.Body).Decode(&result)

	// Existing values should not be overwritten
	if result[0].Income.Current != 200 {
		t.Errorf("Income.Current: expected 200 (unchanged), got %f", result[0].Income.Current)
	}
	if result[0].Release.Current != 150 {
		t.Errorf("Release.Current: expected 150 (unchanged), got %f", result[0].Release.Current)
	}
}

// --- Role-based filtering of the response array ---

// makeGetRequest builds a GET request with claims injected via the auth
// test helper. Lives only in this file because all role-filter tests need
// the same plumbing.
func makeGetRequest(claims *token.Claims) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2026-06-01", nil)
	if claims != nil {
		req = req.WithContext(mwauth.ContextWithClaims(req.Context(), claims))
	}
	return req
}

// roleFilterFixture: 3 orgs + the ИТОГО summary row, in the order the repo
// would return after the config-driven ORDER BY.
func roleFilterFixture() []*reservoirsummary.ResponseModel {
	return []*reservoirsummary.ResponseModel{
		{OrganizationID: ptrInt64(41), OrganizationName: "Org 41"},
		{OrganizationID: ptrInt64(42), OrganizationName: "Org 42"},
		{OrganizationID: ptrInt64(43), OrganizationName: "Org 43"},
		{OrganizationID: nil, OrganizationName: "ИТОГО"},
	}
}

// reservoir role with one OrganizationID must see ONLY that org's row.
// The ИТОГО row (OrganizationID == nil) is dropped because a per-org user
// has no use for a sum across the full report — and showing it would leak
// aggregate data they shouldn't see.
func TestGet_ReservoirRole_OwnOrgOnly(t *testing.T) {
	getter := &mockSummaryGetter{summaries: roleFilterFixture()}
	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	req := makeGetRequest(&token.Claims{
		UserID:          1,
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{42},
	})
	rec := httptest.NewRecorder()
	Get(log, getter, fetcher)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
	var got []*reservoirsummary.ResponseModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want exactly 1 row (org 42), got %d: %+v", len(got), got)
	}
	if got[0].OrganizationID == nil || *got[0].OrganizationID != 42 {
		t.Errorf("want org 42, got %+v", got[0])
	}
	for _, r := range got {
		if r.OrganizationID == nil {
			t.Errorf("ИТОГО row leaked to reservoir-role response: %+v", r)
		}
	}
}

// rais behaves identically to sc — the filter has them in one branch.
// Pinned as its own test so a future refactor splitting them up doesn't
// silently drop one role.
func TestGet_RAISRole_SeesAll(t *testing.T) {
	getter := &mockSummaryGetter{summaries: roleFilterFixture()}
	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	req := makeGetRequest(&token.Claims{UserID: 1, Roles: []string{"rais"}})
	rec := httptest.NewRecorder()
	Get(log, getter, fetcher)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []*reservoirsummary.ResponseModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("rais must see all 4 rows, got %d", len(got))
	}
}

// reservoir with no orgs assigned in claims gets an empty list (200),
// never a 500 and never the full payload. Defends against misconfigured
// JWTs from leaking aggregate data.
func TestGet_ReservoirRole_EmptyOrgsReturnsEmpty(t *testing.T) {
	getter := &mockSummaryGetter{summaries: roleFilterFixture()}
	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	req := makeGetRequest(&token.Claims{
		UserID:          1,
		Roles:           []string{"reservoir"},
		OrganizationIDs: nil,
	})
	rec := httptest.NewRecorder()
	Get(log, getter, fetcher)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []*reservoirsummary.ResponseModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("reservoir with no orgs must get empty list, got %d rows", len(got))
	}
}

// No claims in the context (production: blocked by auth middleware long
// before the handler). The filter's defensive nil-claims guard must still
// return an empty list, not panic and not leak.
func TestGet_NoClaimsReturnsEmpty(t *testing.T) {
	getter := &mockSummaryGetter{summaries: roleFilterFixture()}
	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	req := makeGetRequest(nil) // no ContextWithClaims call
	rec := httptest.NewRecorder()
	Get(log, getter, fetcher)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []*reservoirsummary.ResponseModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("no-claims must get empty list, got %d rows", len(got))
	}
}

// --- modsnow masking by config ---

// TestGet_ModsnowMaskedWhenDisabled: when the per-org config has
// modsnow_enabled=false the JSON response must zero out Modsnow.Current
// and Modsnow.YearAgo regardless of what's in the underlying summary row.
// This is the JSON-side counterpart of the Excel empty-cell behaviour.
func TestGet_ModsnowMaskedWhenDisabled(t *testing.T) {
	orgID := int64(7)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Modsnow: reservoirsummary.ValueResponse{
					Current:     42,
					YearAgo:     11,
					TwoYearsAgo: 5, // not masked — column not part of report
				},
			},
		},
		configs: []reservoirsummary.ReservoirSummaryConfig{
			{OrganizationID: orgID, ModsnowEnabled: false},
		},
	}

	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var result []*reservoirsummary.ResponseModel
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("want 1 summary, got %d", len(result))
	}
	if result[0].Modsnow.Current != 0 {
		t.Errorf("Modsnow.Current: want 0 (masked), got %v", result[0].Modsnow.Current)
	}
	if result[0].Modsnow.YearAgo != 0 {
		t.Errorf("Modsnow.YearAgo: want 0 (masked), got %v", result[0].Modsnow.YearAgo)
	}
}

// TestGet_ModsnowPreservedWhenEnabled: positive branch — config says
// true, JSON keeps the values intact. Pin both branches to defend the
// gate against a sign-flip refactor.
func TestGet_ModsnowPreservedWhenEnabled(t *testing.T) {
	orgID := int64(7)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Modsnow:        reservoirsummary.ValueResponse{Current: 42, YearAgo: 11},
			},
		},
		configs: []reservoirsummary.ReservoirSummaryConfig{
			{OrganizationID: orgID, ModsnowEnabled: true},
		},
	}

	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := Get(log, getter, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var result []*reservoirsummary.ResponseModel
	json.NewDecoder(rr.Body).Decode(&result)

	if result[0].Modsnow.Current != 42 {
		t.Errorf("Modsnow.Current: want 42 (preserved), got %v", result[0].Modsnow.Current)
	}
	if result[0].Modsnow.YearAgo != 11 {
		t.Errorf("Modsnow.YearAgo: want 11 (preserved), got %v", result[0].Modsnow.YearAgo)
	}
}

// End-to-end: when the config row says volume_source=level_volume, the
// handler must load the configs from the repo and forward them into
// applyStaticFallbacks so the curve wins over the DB snapshot. Without
// this wiring, Get would still pass an empty MapConfigLookup and the
// strategy switch would silently fall back to "static" for every org.
func TestGet_VolumeSourceLevelVolume_E2E(t *testing.T) {
	orgID := int64(96)

	getter := &mockSummaryGetter{
		summaries: []*reservoirsummary.ResponseModel{
			{
				OrganizationID: ptrInt64(orgID),
				Level:          reservoirsummary.ValueResponse{Current: 200},
				Volume:         reservoirsummary.ValueResponse{Current: 100}, // snapshot the curve must override
			},
		},
		curveVolume: 150,
		configs: []reservoirsummary.ReservoirSummaryConfig{
			{ID: 1, OrganizationID: orgID, SortOrder: 1, IncludeInTotal: true, ModsnowEnabled: true, VolumeSource: "level_volume"},
		},
	}
	fetcher := &mockStaticDataFetcher{}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary?date=2025-01-01", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(), &token.Claims{Roles: []string{"sc"}}))
	rec := httptest.NewRecorder()
	Get(log, getter, fetcher)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
	var got []*reservoirsummary.ResponseModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	if got[0].Volume.Current != 150 {
		t.Errorf("Volume.Current: want 150 (curve, via level_volume strategy), got %v", got[0].Volume.Current)
	}
	if got[0].Volume.IsEdited == nil || !*got[0].Volume.IsEdited {
		t.Errorf("Volume.IsEdited: want true, got %v", got[0].Volume.IsEdited)
	}
}

// sc role sees the full payload exactly as the repo returned it, including
// the ИТОГО row. Same handler, same fixture — only the claims differ.
func TestGet_SCRole_SeesAll(t *testing.T) {
	getter := &mockSummaryGetter{summaries: roleFilterFixture()}
	fetcher := &mockStaticDataFetcher{}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	req := makeGetRequest(&token.Claims{
		UserID: 1,
		Roles:  []string{"sc"},
	})
	rec := httptest.NewRecorder()
	Get(log, getter, fetcher)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []*reservoirsummary.ResponseModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("sc must see all 4 rows (3 orgs + ИТОГО), got %d", len(got))
	}
	// Sanity-check that ИТОГО row survived.
	foundItog := false
	for _, r := range got {
		if r.OrganizationID == nil {
			foundItog = true
		}
	}
	if !foundItog {
		t.Errorf("ИТОГО row missing from sc-role response")
	}
}

