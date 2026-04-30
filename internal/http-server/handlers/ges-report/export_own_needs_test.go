package gesreport

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/ges-report"
	ownneedsgen "srmt-admin/internal/lib/service/excel/ownneeds"
	"srmt-admin/internal/token"
)

const ownNeedsTemplatePath = "../../../../template/own-needs.xlsx"

type mockOwnNeedsBuilder struct {
	report *model.OwnNeedsReport
	err    error
}

func (m *mockOwnNeedsBuilder) BuildOwnNeedsReport(_ context.Context, _ string) (*model.OwnNeedsReport, error) {
	return m.report, m.err
}

func ownNeedsPtr(v float64) *float64 { return &v }

func testOwnNeedsReport() *model.OwnNeedsReport {
	return &model.OwnNeedsReport{
		Date: "2026-04-27",
		Cascades: []model.OwnNeedsCascade{
			{
				CascadeID:   1,
				CascadeName: "Test Cascade",
				Stations: []model.OwnNeedsStation{
					{
						OrganizationID: 100, Name: "Station A",
						InstalledCapacityMWt: 10.0, OwnConsumptionKWh: ownNeedsPtr(500.0),
						MTDOwnConsumptionKWh: 1000, YTDOwnConsumptionKWh: 5000,
					},
				},
				Totals: model.OwnNeedsTotals{
					InstalledCapacityMWt: 10.0, OwnConsumptionKWh: 500.0,
					MTDOwnConsumptionKWh: 1000, YTDOwnConsumptionKWh: 5000,
				},
			},
		},
		GrandTotal: model.OwnNeedsTotals{
			InstalledCapacityMWt: 10.0, OwnConsumptionKWh: 500.0,
			MTDOwnConsumptionKWh: 1000, YTDOwnConsumptionKWh: 5000,
		},
	}
}

func setupOwnNeedsRouter(t *testing.T, builder *mockOwnNeedsBuilder, role string) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: &token.Claims{
		UserID: 1, ContactID: 1, OrganizationID: 1,
		Roles: []string{role},
	}}
	loc, _ := time.LoadLocation("Asia/Tashkent")
	gen := ownneedsgen.New(ownNeedsTemplatePath)

	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Group(func(r chi.Router) {
		r.Use(mwauth.RequireAnyRole("sc", "rais"))
		r.Get("/own-needs/export", ExportOwnNeeds(logger, builder, gen, loc))
	})
	return r
}

func doOwnNeedsGET(t *testing.T, h http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/own-needs/export?"+query, nil)
	req.Header.Set("Authorization", "Bearer faketoken")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestExportOwnNeeds_OK(t *testing.T) {
	h := setupOwnNeedsRouter(t, &mockOwnNeedsBuilder{report: testOwnNeedsReport()}, "sc")
	rr := doOwnNeedsGET(t, h, "date=2026-04-27")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("Content-Type: got %q", ct)
	}
	cd := rr.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "Own-Needs-2026-04-27.xlsx") {
		t.Errorf("Content-Disposition: got %q, want contains Own-Needs-2026-04-27.xlsx", cd)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("body is empty")
	}

	// Body must be a valid xlsx that excelize can parse.
	f, err := excelize.OpenReader(rr.Body)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "27.04.26" {
		t.Errorf("sheet name: got %v, want [27.04.26]", sheets)
	}
}

func TestExportOwnNeeds_RaisRoleAllowed(t *testing.T) {
	h := setupOwnNeedsRouter(t, &mockOwnNeedsBuilder{report: testOwnNeedsReport()}, "rais")
	rr := doOwnNeedsGET(t, h, "date=2026-04-27")
	if rr.Code != http.StatusOK {
		t.Fatalf("rais should be allowed: got %d. body=%s", rr.Code, rr.Body.String())
	}
}

func TestExportOwnNeeds_CascadeRoleForbidden(t *testing.T) {
	h := setupOwnNeedsRouter(t, &mockOwnNeedsBuilder{report: testOwnNeedsReport()}, "cascade")
	rr := doOwnNeedsGET(t, h, "date=2026-04-27")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("cascade role should be 403: got %d. body=%s", rr.Code, rr.Body.String())
	}
}

func TestExportOwnNeeds_RequiresDate(t *testing.T) {
	h := setupOwnNeedsRouter(t, &mockOwnNeedsBuilder{report: testOwnNeedsReport()}, "sc")
	rr := doOwnNeedsGET(t, h, "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rr.Code)
	}
}

func TestExportOwnNeeds_BadDateFormat(t *testing.T) {
	h := setupOwnNeedsRouter(t, &mockOwnNeedsBuilder{report: testOwnNeedsReport()}, "sc")
	rr := doOwnNeedsGET(t, h, "date=not-a-date")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rr.Code)
	}
}

func TestExportOwnNeeds_BuilderErrorReturns500(t *testing.T) {
	h := setupOwnNeedsRouter(t, &mockOwnNeedsBuilder{err: errBuilder("boom")}, "sc")
	rr := doOwnNeedsGET(t, h, "date=2026-04-27")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", rr.Code)
	}
}

// --- helpers ---

type errBuilder string

func (e errBuilder) Error() string { return string(e) }
