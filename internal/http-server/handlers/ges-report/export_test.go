package gesreport

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/ges-report"
	gesgen "srmt-admin/internal/lib/service/excel/ges"
	"srmt-admin/internal/token"
)

// --- mocks ---

type mockReportBuilder struct {
	report *model.DailyReport
	err    error
}

func (m *mockReportBuilder) BuildDailyReport(_ context.Context, _ string, _ *int64) (*model.DailyReport, error) {
	return m.report, m.err
}

type mockPlanGetter struct {
	plans []model.PlanRow
	err   error
}

func (m *mockPlanGetter) GetGESPlansForReport(_ context.Context, _ int, _ []int) ([]model.PlanRow, error) {
	return m.plans, m.err
}

type mockOrgTypesGetter struct {
	types map[int64][]string
	err   error
}

func (m *mockOrgTypesGetter) GetOrganizationTypesMap(_ context.Context) (map[int64][]string, error) {
	return m.types, m.err
}

// --- helpers ---

func testDailyReport() *model.DailyReport {
	return &model.DailyReport{
		Date: "2026-04-16",
		Cascades: []model.CascadeReport{
			{
				CascadeID:   1,
				CascadeName: "Test Cascade",
				Stations: []model.StationReport{
					{
						OrganizationID: 100,
						Name:           "Station A",
						Config: model.StationConfig{
							TotalAggregates: 5,
						},
						Current: model.CurrentData{
							WorkingAggregates: 3,
						},
					},
					{
						OrganizationID: 200,
						Name:           "Station B",
						Config: model.StationConfig{
							TotalAggregates: 5,
						},
						Current: model.CurrentData{
							WorkingAggregates: 2,
						},
					},
				},
			},
		},
		GrandTotal: &model.SummaryBlock{
			TotalAggregates:   10,
			WorkingAggregates: 5,
		},
	}
}

func testPlans() []model.PlanRow {
	plans := make([]model.PlanRow, 0, 12)
	for m := 1; m <= 12; m++ {
		plans = append(plans, model.PlanRow{
			OrganizationID: 100,
			Year:           2026,
			Month:          m,
			PlanMlnKWh:     10.0,
		})
	}
	return plans
}

func testOrgTypes() map[int64][]string {
	return map[int64][]string{
		100: {"ges"},
		200: {"mini"},
	}
}

// createTestTemplate creates a minimal xlsx file for the generator and returns
// the temp directory path (caller should defer os.RemoveAll).
func createTestTemplate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "ges-test.xlsx")
	f := excelize.NewFile()
	// Generator expects at least one sheet
	_ = f.SaveAs(tmplPath)
	f.Close()
	return tmplPath
}

func setupExportRouter(
	t *testing.T,
	builder *mockReportBuilder,
	planGetter *mockPlanGetter,
	orgTypes *mockOrgTypesGetter,
) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: &token.Claims{
		UserID:         1,
		ContactID:      1,
		OrganizationID: 1,
		Roles:          []string{"sc"},
	}}
	loc, _ := time.LoadLocation("Asia/Tashkent")
	tmplPath := createTestTemplate(t)
	gen := gesgen.New(tmplPath)

	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Get("/export", Export(logger, builder, planGetter, orgTypes, gen, loc))
	return r
}

func doExportGET(t *testing.T, h http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/export?"+query, nil)
	req.Header.Set("Authorization", "Bearer faketoken")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// --- tests ---

func TestExport_Success(t *testing.T) {
	h := setupExportRouter(t,
		&mockReportBuilder{report: testDailyReport()},
		&mockPlanGetter{plans: testPlans()},
		&mockOrgTypesGetter{types: testOrgTypes()},
	)

	rr := doExportGET(t, h, "date=2026-04-16")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	want := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if ct != want {
		t.Errorf("Content-Type: got %q, want %q", ct, want)
	}

	cd := rr.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("Content-Disposition header missing")
	}

	if rr.Body.Len() == 0 {
		t.Error("response body is empty")
	}
}

func TestExport_MissingDate(t *testing.T) {
	h := setupExportRouter(t,
		&mockReportBuilder{report: testDailyReport()},
		&mockPlanGetter{plans: testPlans()},
		&mockOrgTypesGetter{types: testOrgTypes()},
	)

	rr := doExportGET(t, h, "format=excel")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", rr.Code, rr.Body.String())
	}
}

func TestExport_InvalidDate(t *testing.T) {
	h := setupExportRouter(t,
		&mockReportBuilder{report: testDailyReport()},
		&mockPlanGetter{plans: testPlans()},
		&mockOrgTypesGetter{types: testOrgTypes()},
	)

	rr := doExportGET(t, h, "date=not-a-date")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", rr.Code, rr.Body.String())
	}
}

func TestExport_InvalidFormat(t *testing.T) {
	h := setupExportRouter(t,
		&mockReportBuilder{report: testDailyReport()},
		&mockPlanGetter{plans: testPlans()},
		&mockOrgTypesGetter{types: testOrgTypes()},
	)

	rr := doExportGET(t, h, "date=2026-04-16&format=csv")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", rr.Code, rr.Body.String())
	}
}

func TestCountOrgTypes_CommaInMicroName(t *testing.T) {
	cascades := []model.CascadeReport{{
		Stations: []model.StationReport{
			{OrganizationID: 1, Name: "Зомин микроГЭС-1,2"},
			{OrganizationID: 2, Name: "Чирчик ГЭС-7"},
		},
	}}
	typesMap := map[int64][]string{
		1: {"micro"},
		2: {"ges"},
	}
	got := countOrgTypes(cascades, typesMap)
	want := gesgen.OrgTypeCounts{GES: 1, Mini: 0, Micro: 2, Total: 3}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCountOrgTypes_NoCommaInName(t *testing.T) {
	cascades := []model.CascadeReport{{
		Stations: []model.StationReport{{OrganizationID: 1, Name: "Туполанг ГЭС"}},
	}}
	got := countOrgTypes(cascades, map[int64][]string{1: {"ges"}})
	want := gesgen.OrgTypeCounts{GES: 1, Total: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCountOrgTypes_MultipleCommasInName(t *testing.T) {
	cascades := []model.CascadeReport{{
		Stations: []model.StationReport{{OrganizationID: 1, Name: "ГЭС-1,2,3"}},
	}}
	got := countOrgTypes(cascades, map[int64][]string{1: {"ges"}})
	want := gesgen.OrgTypeCounts{GES: 3, Total: 3}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCountOrgTypes_EmptyName(t *testing.T) {
	cascades := []model.CascadeReport{{
		Stations: []model.StationReport{{OrganizationID: 1, Name: ""}},
	}}
	got := countOrgTypes(cascades, map[int64][]string{1: {"micro"}})
	want := gesgen.OrgTypeCounts{Micro: 1, Total: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCountOrgTypes_MultipleCascades(t *testing.T) {
	cascades := []model.CascadeReport{
		{Stations: []model.StationReport{
			{OrganizationID: 1, Name: "ГЭС-1,2"},
			{OrganizationID: 2, Name: "ГЭС-3"},
		}},
		{Stations: []model.StationReport{
			{OrganizationID: 3, Name: "микроГЭС"},
		}},
	}
	typesMap := map[int64][]string{
		1: {"ges"},
		2: {"ges"},
		3: {"micro"},
	}
	got := countOrgTypes(cascades, typesMap)
	want := gesgen.OrgTypeCounts{GES: 3, Micro: 1, Total: 4}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCountOrgTypes_MultipleTypesPerStation(t *testing.T) {
	cascades := []model.CascadeReport{{
		Stations: []model.StationReport{{OrganizationID: 1, Name: "ГЭС-1,2"}},
	}}
	typesMap := map[int64][]string{1: {"ges", "mini"}}
	got := countOrgTypes(cascades, typesMap)
	want := gesgen.OrgTypeCounts{GES: 2, Mini: 2, Total: 4}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCountOrgTypes_UnknownTypeIgnored(t *testing.T) {
	cascades := []model.CascadeReport{{
		Stations: []model.StationReport{
			{OrganizationID: 1, Name: "strange"},
			{OrganizationID: 2, Name: "ГЭС"},
		},
	}}
	typesMap := map[int64][]string{
		1: {"virtual"},
		2: {"ges"},
	}
	got := countOrgTypes(cascades, typesMap)
	want := gesgen.OrgTypeCounts{GES: 1, Total: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// Reserve validation now lives in the service layer (clamp ≥ 0) and the DB
// CHECK constraint on ges_daily_data, so the handler no longer rejects
// "negative reserve" requests. modernization/repair query params have been
// removed entirely — the values come from report.GrandTotal.*Aggregates.
func TestExport_IgnoresLegacyAggregateQueryParams(t *testing.T) {
	h := setupExportRouter(t,
		&mockReportBuilder{report: testDailyReport()},
		&mockPlanGetter{plans: testPlans()},
		&mockOrgTypesGetter{types: testOrgTypes()},
	)

	// Legacy "modernization=4&repair=3" params, which once would have caused
	// a 400 because reserve = 10-5-4-3 = -2, must now be silently ignored.
	rr := doExportGET(t, h, "date=2026-04-16&modernization=4&repair=3")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
}
