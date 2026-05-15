package gesreport

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	gesreportservice "srmt-admin/internal/lib/service/ges-report"
)

// TestGetReport_ConsumptionExceedsIdle_400Structured pins the handler-side
// translation of the typed *ReportValidationError into a 400 with structured
// body. Frontend keys off `code` for localization and uses `details` to list
// every offending station.
func TestGetReport_ConsumptionExceedsIdle_400Structured(t *testing.T) {
	builder := &mockReportBuilder{
		err: &gesreportservice.ReportValidationError{
			Code: gesreportservice.ReportValidationErrorCode,
			Violations: []gesreportservice.ConsumptionViolation{
				{
					OrganizationID:   16,
					OrganizationName: "ГЭС-1",
					Date:             "2026-04-22",
					IdleM3s:          2.0,
					ConsumptionM3s:   5.0,
				},
				{
					OrganizationID:   17,
					OrganizationName: "ГЭС-2",
					Date:             "2026-04-22",
					IdleM3s:          1.0,
					ConsumptionM3s:   3.0,
				},
			},
		},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := GetReport(log, builder)

	req := httptest.NewRequest(http.MethodGet, "/ges/daily-report?date=2026-04-22", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var body struct {
		Error   string                   `json:"error"`
		Code    string                   `json:"code"`
		Details []map[string]any         `json:"details"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v; raw=%s", err, rr.Body.String())
	}
	if body.Code != "report.consumption_exceeds_idle" {
		t.Errorf("code: want %q, got %q", "report.consumption_exceeds_idle", body.Code)
	}
	if body.Error == "" {
		t.Error("error message must be populated for frontend fallback")
	}
	if len(body.Details) != 2 {
		t.Fatalf("details: want 2 entries, got %d (body: %s)", len(body.Details), rr.Body.String())
	}

	// First violation: ГЭС-1 (id=16), idle=2, consumption=5.
	d0 := body.Details[0]
	if id, _ := d0["organization_id"].(float64); id != 16 {
		t.Errorf("details[0].organization_id: want 16, got %v", d0["organization_id"])
	}
	if d0["organization_name"] != "ГЭС-1" {
		t.Errorf("details[0].organization_name: want %q, got %v", "ГЭС-1", d0["organization_name"])
	}
	if v, _ := d0["idle_m3_s"].(float64); v != 2.0 {
		t.Errorf("details[0].idle_m3_s: want 2.0, got %v", d0["idle_m3_s"])
	}
	if v, _ := d0["consumption_m3_s"].(float64); v != 5.0 {
		t.Errorf("details[0].consumption_m3_s: want 5.0, got %v", d0["consumption_m3_s"])
	}
	if d0["date"] != "2026-04-22" {
		t.Errorf("details[0].date: want %q, got %v", "2026-04-22", d0["date"])
	}

	// Second violation: ensure both are included (no early-exit).
	d1 := body.Details[1]
	if id, _ := d1["organization_id"].(float64); id != 17 {
		t.Errorf("details[1].organization_id: want 17, got %v", d1["organization_id"])
	}
}

// Generic non-validation errors still go through the 500 path. Pins that the
// new typed-error branch did not accidentally swallow other errors.
func TestGetReport_GenericError_500(t *testing.T) {
	builder := &mockReportBuilder{
		err: errPlain("database is on fire"),
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := GetReport(log, builder)

	req := httptest.NewRequest(http.MethodGet, "/ges/daily-report?date=2026-04-22", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// errPlain is a tiny string error used in tests to differentiate "anything
// other than a typed error" without dragging fmt into the assertion.
type errPlain string

func (e errPlain) Error() string { return string(e) }
