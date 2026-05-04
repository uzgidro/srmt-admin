package reservoirflood

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	selgen "srmt-admin/internal/lib/service/excel/sel"
	"srmt-admin/internal/token"
)

// ---------- mocks ----------

type fakeBuilder struct {
	mu       sync.Mutex
	gotDate  time.Time
	gotHour  int
	gotName  string
	out      *selgen.Report
	err      error
}

func (b *fakeBuilder) BuildReport(_ context.Context, date time.Time, hour int, authorShort string) (*selgen.Report, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.gotDate = date
	b.gotHour = hour
	b.gotName = authorShort
	if b.out != nil {
		return b.out, b.err
	}
	return &selgen.Report{Date: date, Hour: hour, AuthorShort: authorShort}, b.err
}

type fakeGenerator struct{}

func (fakeGenerator) GenerateExcel(_ *selgen.Report) (*excelize.File, error) {
	// Return an empty workbook so the handler can serialize it.
	return excelize.NewFile(), nil
}

// captureLogger collects log records so we can assert on warning emission.
type captureLogger struct {
	mu      sync.Mutex
	records []string
}

func (c *captureLogger) Handler() slog.Handler {
	return &captureHandler{owner: c}
}

type captureHandler struct{ owner *captureLogger }

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.owner.mu.Lock()
	defer h.owner.mu.Unlock()
	h.owner.records = append(h.owner.records, r.Level.String()+": "+r.Message)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func (c *captureLogger) hasWarn(substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.records {
		if strings.HasPrefix(r, "WARN") && strings.Contains(r, substr) {
			return true
		}
	}
	return false
}

// ---------- helpers ----------

func newExportRouter(builder SelReportBuilder, gen SelExcelGenerator, log *slog.Logger, claims *token.Claims) http.Handler {
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(&mockTokenVerifier{claims: claims}))
	loc, _ := time.LoadLocation("Asia/Tashkent")
	r.Get("/reservoir-flood/export", GetExport(log, builder, gen, loc))
	return r
}

func doExportRequest(t *testing.T, builder SelReportBuilder, gen SelExcelGenerator, log *slog.Logger, claims *token.Claims, target string) *httptest.ResponseRecorder {
	t.Helper()
	r := newExportRouter(builder, gen, log, claims)
	req := httptest.NewRequest(http.MethodGet, target, bytes.NewBuffer(nil))
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func discardExportLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ---------- tests ----------

func TestGetExport_BadDate(t *testing.T) {
	rr := doExportRequest(t, &fakeBuilder{}, fakeGenerator{}, discardExportLogger(), scClaims(),
		"/reservoir-flood/export?date=not-a-date")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExport_BadFormat(t *testing.T) {
	rr := doExportRequest(t, &fakeBuilder{}, fakeGenerator{}, discardExportLogger(), scClaims(),
		"/reservoir-flood/export?date=2026-05-04&format=mp3")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExport_BadHour(t *testing.T) {
	for _, h := range []string{"24", "-1", "abc"} {
		rr := doExportRequest(t, &fakeBuilder{}, fakeGenerator{}, discardExportLogger(), scClaims(),
			"/reservoir-flood/export?date=2026-05-04&hour="+h)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("hour=%q: want 400, got %d, body: %s", h, rr.Code, rr.Body.String())
		}
	}
}

func TestGetExport_DefaultHour(t *testing.T) {
	b := &fakeBuilder{}
	rr := doExportRequest(t, b, fakeGenerator{}, discardExportLogger(), scClaims(),
		"/reservoir-flood/export?date=2026-05-04")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if b.gotHour != 0 {
		t.Errorf("default hour: want 0, got %d", b.gotHour)
	}
}

func TestGetExport_HourOutsideWindow(t *testing.T) {
	cap := &captureLogger{}
	log := slog.New(cap.Handler())
	rr := doExportRequest(t, &fakeBuilder{}, fakeGenerator{}, log, scClaims(),
		"/reservoir-flood/export?date=2026-05-04&hour=12")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200 (outside window not rejected), got %d, body: %s", rr.Code, rr.Body.String())
	}
	if !cap.hasWarn("reporting window") {
		t.Errorf("expected WARN about reporting window for hour=12; got records: %v", cap.records)
	}
}

func TestGetExport_HourInsideWindow(t *testing.T) {
	for _, h := range []string{"21", "22", "23", "0", "5", "8"} {
		cap := &captureLogger{}
		log := slog.New(cap.Handler())
		rr := doExportRequest(t, &fakeBuilder{}, fakeGenerator{}, log, scClaims(),
			"/reservoir-flood/export?date=2026-05-04&hour="+h)
		if rr.Code != http.StatusOK {
			t.Fatalf("hour=%s: want 200, got %d", h, rr.Code)
		}
		if cap.hasWarn("reporting window") {
			t.Errorf("hour=%s inside window: must NOT emit reporting-window WARN; got records: %v", h, cap.records)
		}
	}
}

func TestGetExport_ExcelHappyPath(t *testing.T) {
	rr := doExportRequest(t, &fakeBuilder{}, fakeGenerator{}, discardExportLogger(), scClaims(),
		"/reservoir-flood/export?date=2026-05-04&format=excel")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body length=%d", rr.Code, rr.Body.Len())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "spreadsheetml.sheet") {
		t.Errorf("Content-Type: want xlsx mime, got %q", ct)
	}
	cd := rr.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".xlsx") {
		t.Errorf("Content-Disposition: want filename=*.xlsx, got %q", cd)
	}
}
