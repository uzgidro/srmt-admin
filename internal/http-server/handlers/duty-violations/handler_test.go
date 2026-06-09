package dutyviolationshandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	dvmodel "srmt-admin/internal/lib/model/duty-violations"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

// --- mocks ---

type mockCreator struct {
	out *dvmodel.DutyViolation
	err error
}

func (m *mockCreator) Create(_ context.Context, _ dvmodel.CreateRequest, _ int64) (*dvmodel.DutyViolation, error) {
	return m.out, m.err
}

type mockLister struct {
	out []dvmodel.OrgGroup
	err error
	got dvmodel.ListFilter
}

func (m *mockLister) List(_ context.Context, f dvmodel.ListFilter) ([]dvmodel.OrgGroup, error) {
	m.got = f
	return m.out, m.err
}

type mockUpdater struct {
	// GetByID half — what the IDOR-defense read returns
	getOut *dvmodel.DutyViolation
	getErr error

	// Update half — what the actual mutation returns
	out *dvmodel.DutyViolation
	err error

	updateCalled bool
}

func (m *mockUpdater) GetByID(_ context.Context, _ int64) (*dvmodel.DutyViolation, error) {
	return m.getOut, m.getErr
}

func (m *mockUpdater) Update(_ context.Context, _ int64, _ dvmodel.UpdateRequest) (*dvmodel.DutyViolation, error) {
	m.updateCalled = true
	return m.out, m.err
}

type mockDeleter struct {
	getOut       *dvmodel.DutyViolation
	getErr       error
	err          error
	deleteCalled bool
}

func (m *mockDeleter) GetByID(_ context.Context, _ int64) (*dvmodel.DutyViolation, error) {
	return m.getOut, m.getErr
}

func (m *mockDeleter) Delete(_ context.Context, _ int64) error {
	m.deleteCalled = true
	return m.err
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// tashkentLoc is loaded once and shared across tests — Asia/Tashkent is the
// project's operational-day timezone, fixed +05:00 (no DST), so we don't
// need any rebuild or per-test setup. Falling back to UTC keeps tests
// runnable on systems without tzdata installed.
var tashkentLoc = func() *time.Location {
	if l, err := time.LoadLocation("Asia/Tashkent"); err == nil {
		return l
	}
	return time.FixedZone("Asia/Tashkent", 5*60*60)
}()

// scClaims gives the caller full org access (CheckOrgAccess pass-through).
func scClaims(userID int64) *token.Claims {
	return &token.Claims{UserID: userID, Roles: []string{"sc"}}
}

func validCreateBody() string {
	return `{
		"organization_id": 103,
		"start_time": "2026-06-08T08:00:00Z",
		"end_time": "2026-06-08T20:00:00Z",
		"duty_officer_name": "Иванов И.И.",
		"reason": "Прогул",
		"file_ids": [42, 43]
	}`
}

// withID injects a chi URL param into the request context. Used to
// simulate routing without spinning up the full router.
func withID(req *http.Request, id string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func authedRequest(method, url, body string, userID int64) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r = r.WithContext(mwauth.ContextWithClaims(r.Context(), scClaims(userID)))
	return r
}

// --- POST /duty-violations ---

func TestAdd_HappyPath(t *testing.T) {
	want := &dvmodel.DutyViolation{ID: 7, OrganizationID: 103}
	svc := &mockCreator{out: want}
	req := authedRequest(http.MethodPost, "/duty-violations", validCreateBody(), 1)
	rec := httptest.NewRecorder()

	Add(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
	var got dvmodel.DutyViolation
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("response id: want 7, got %d", got.ID)
	}
}

func TestAdd_InvalidJSON(t *testing.T) {
	svc := &mockCreator{}
	req := authedRequest(http.MethodPost, "/duty-violations", "{not json", 1)
	rec := httptest.NewRecorder()
	Add(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

func TestAdd_ValidationFailures(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing organization_id",
			`{"start_time":"2026-06-08T08:00:00Z","end_time":"2026-06-08T20:00:00Z","duty_officer_name":"X","reason":"Y"}`},
		{"end <= start",
			`{"organization_id":1,"start_time":"2026-06-08T20:00:00Z","end_time":"2026-06-08T08:00:00Z","duty_officer_name":"X","reason":"Y"}`},
		{"blank name",
			`{"organization_id":1,"start_time":"2026-06-08T08:00:00Z","end_time":"2026-06-08T20:00:00Z","duty_officer_name":"","reason":"Y"}`},
		{"blank reason",
			`{"organization_id":1,"start_time":"2026-06-08T08:00:00Z","end_time":"2026-06-08T20:00:00Z","duty_officer_name":"X","reason":""}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockCreator{}
			req := authedRequest(http.MethodPost, "/duty-violations", tc.body, 1)
			rec := httptest.NewRecorder()
			Add(quietLog(), svc)(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status: want 400, got %d (body %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// Non-privileged caller targeting a foreign org gets 403 BEFORE the
// service is touched. Confirms authorization is checked at the boundary,
// not relied on inside the service.
func TestAdd_OrgAccessDenied(t *testing.T) {
	svc := &mockCreator{out: &dvmodel.DutyViolation{ID: 1}}
	req := httptest.NewRequest(http.MethodPost, "/duty-violations", strings.NewReader(validCreateBody()))
	// claims have org [50] — request targets 103 → forbidden
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	rec := httptest.NewRecorder()
	Add(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
}

func TestAdd_ServiceFKViolation_422(t *testing.T) {
	svc := &mockCreator{err: storage.ErrForeignKeyViolation}
	req := authedRequest(http.MethodPost, "/duty-violations", validCreateBody(), 1)
	rec := httptest.NewRecorder()
	Add(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: want 422, got %d", rec.Code)
	}
}

// CHECK constraint (whitespace-only field) bubbles up as a 400, not a 500.
// The migration has CHECK (length(trim(text)) > 0) on duty_officer_name and
// reason; the validator's min=1 doesn't trim, so this path is reachable.
func TestAdd_CheckConstraintViolation_400(t *testing.T) {
	svc := &mockCreator{err: storage.ErrCheckConstraintViolation}
	req := authedRequest(http.MethodPost, "/duty-violations", validCreateBody(), 1)
	rec := httptest.NewRecorder()
	Add(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

func TestAdd_ServiceError_500(t *testing.T) {
	svc := &mockCreator{err: errors.New("db down")}
	req := authedRequest(http.MethodPost, "/duty-violations", validCreateBody(), 1)
	rec := httptest.NewRecorder()
	Add(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", rec.Code)
	}
}

// --- GET /duty-violations ---

func TestList_HappyPath_NoFilters(t *testing.T) {
	svc := &mockLister{out: []dvmodel.OrgGroup{
		{ID: 1, Name: "Org A", Violations: []dvmodel.DutyViolation{{ID: 1}}},
		{ID: 2, Name: "Org B", Violations: []dvmodel.DutyViolation{{ID: 2}, {ID: 3}}},
	}}
	req := authedRequest(http.MethodGet, "/duty-violations", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []dvmodel.OrgGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 groups, got %d", len(got))
	}
	if got[0].ID != 1 || got[0].Name != "Org A" || len(got[0].Violations) != 1 {
		t.Errorf("group A wrong: %+v", got[0])
	}
	if got[1].ID != 2 || got[1].Name != "Org B" || len(got[1].Violations) != 2 {
		t.Errorf("group B wrong: %+v", got[1])
	}
}

// Confirms the handler forwards org_id and date to the service.
// date=2026-06-08 → Day = 2026-06-08T05:00 Tashkent (start of op-day).
// The repo handles the +24h end-of-window.
func TestList_ForwardsAllFilters(t *testing.T) {
	svc := &mockLister{out: nil}
	req := authedRequest(http.MethodGet,
		"/duty-violations?organization_id=42&date=2026-06-08", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if svc.got.OrganizationID == nil || *svc.got.OrganizationID != 42 {
		t.Errorf("org filter not forwarded: %+v", svc.got)
	}
	wantDay := time.Date(2026, 6, 8, 5, 0, 0, 0, tashkentLoc)
	if svc.got.Day == nil || !svc.got.Day.Equal(wantDay) {
		t.Errorf("date filter wrong: want %v, got %v", wantDay, svc.got.Day)
	}
}

// Op-day anchor uses the configured timezone, not UTC. ?date=2026-06-08
// with Asia/Tashkent loc must yield 00:00 UTC of that calendar date
// (05:00 local = 00:00 UTC). Regression guard against re-introducing
// time.Parse, which would silently produce midnight UTC.
func TestList_DateUsesLocationNotUTC(t *testing.T) {
	svc := &mockLister{out: nil}
	req := authedRequest(http.MethodGet, "/duty-violations?date=2026-06-08", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	wantUTC := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	if svc.got.Day == nil || !svc.got.Day.UTC().Equal(wantUTC) {
		t.Errorf("date in UTC: want %v, got %v", wantUTC, svc.got.Day.UTC())
	}
}

// Without ?date the handler forwards a nil Day filter — the repo then
// returns every record. Matches the contract used by the project's other
// list endpoints (incidents/visits also default to "no day filter" when
// the parameter is missing inside the handler, before the handler's own
// fallback kicks in).
func TestList_NoDateForwardsNilDay(t *testing.T) {
	svc := &mockLister{out: nil}
	req := authedRequest(http.MethodGet, "/duty-violations", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if svc.got.Day != nil {
		t.Errorf("missing ?date must NOT set Day filter, got %v", svc.got.Day)
	}
}

func TestList_InvalidFilter_400(t *testing.T) {
	svc := &mockLister{}
	req := authedRequest(http.MethodGet, "/duty-violations?organization_id=abc", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

// nil result from the service must render as `[]`, never `null` — the
// frontend should never have to guard against missing-list shapes.
func TestList_NilResult_RendersEmptyArray(t *testing.T) {
	svc := &mockLister{out: nil}
	req := authedRequest(http.MethodGet, "/duty-violations", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("want `[]`, got %q", body)
	}
}

// Tenant scoping: a non-privileged caller WITHOUT organization_id in the
// query must NOT see every record. The handler force-injects their own
// first org from claims so the SQL filter binds.
func TestList_NonPrivilegedNoFilter_AutoScoped(t *testing.T) {
	svc := &mockLister{out: nil}
	req := httptest.NewRequest(http.MethodGet, "/duty-violations", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	if svc.got.OrganizationID == nil || *svc.got.OrganizationID != 50 {
		t.Errorf("non-privileged caller must be auto-scoped to own org; got filter %+v", svc.got)
	}
}

// Tenant scoping: a non-privileged caller passing a FOREIGN organization_id
// gets 403, NOT a quiet empty list (we want loud rejection so the
// frontend bug surfaces).
func TestList_NonPrivilegedForeignOrg_Forbidden(t *testing.T) {
	svc := &mockLister{}
	req := httptest.NewRequest(http.MethodGet, "/duty-violations?organization_id=999", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
}

// Non-privileged caller without ANY orgs in claims gets an empty list,
// not a wildcard query.
func TestList_NonPrivilegedNoOrgs_ReturnsEmpty(t *testing.T) {
	svc := &mockLister{out: []dvmodel.OrgGroup{
		{ID: 1, Name: "Org A", Violations: []dvmodel.DutyViolation{{ID: 1}}},
	}}
	req := httptest.NewRequest(http.MethodGet, "/duty-violations", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: nil}))
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("want `[]`, got %q", body)
	}
	// Crucially: the service was NEVER called. A wildcard query against
	// the DB would leak everything.
	if svc.got.OrganizationID != nil {
		t.Errorf("service must not be invoked for no-orgs caller; got filter %+v", svc.got)
	}
}

// sc role keeps unrestricted access — no auto-scoping, no 403 on any
// organization_id, including absent.
func TestList_PrivilegedNoFilter_PassesThrough(t *testing.T) {
	svc := &mockLister{out: []dvmodel.OrgGroup{
		{ID: 1, Name: "Org A", Violations: []dvmodel.DutyViolation{{ID: 1}}},
		{ID: 2, Name: "Org B", Violations: []dvmodel.DutyViolation{{ID: 2}}},
	}}
	req := authedRequest(http.MethodGet, "/duty-violations", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc, tashkentLoc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	if svc.got.OrganizationID != nil {
		t.Errorf("sc must NOT be auto-scoped; got filter %+v", svc.got)
	}
}

// --- PATCH /duty-violations/{id} ---

// existingDV is the typical "record already in DB" the IDOR-defense read
// returns. Pass an OrganizationID; sc claims pass through CheckOrgAccess
// for any value, so most tests just use the same org as the request body.
func existingDV(orgID int64) *dvmodel.DutyViolation {
	return &dvmodel.DutyViolation{ID: 5, OrganizationID: orgID}
}

func TestEdit_HappyPath(t *testing.T) {
	svc := &mockUpdater{
		getOut: existingDV(103),
		out:    &dvmodel.DutyViolation{ID: 5, OrganizationID: 103},
	}
	req := authedRequest(http.MethodPatch, "/duty-violations/5", validCreateBody(), 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
}

// GetByID-side ErrNotFound (record doesn't exist) must return 404 from the
// pre-update read — not propagate as a 500.
func TestEdit_NotFoundOnPreRead(t *testing.T) {
	svc := &mockUpdater{getErr: storage.ErrNotFound}
	req := authedRequest(http.MethodPatch, "/duty-violations/9999", validCreateBody(), 1)
	req = withID(req, "9999")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", rec.Code)
	}
	if svc.updateCalled {
		t.Errorf("Update must not run when pre-read returns NotFound")
	}
}

// NotFound on the Update half (after a successful pre-read) — rare race
// window. Still surfaces as 404 to the caller.
func TestEdit_NotFoundOnUpdate(t *testing.T) {
	svc := &mockUpdater{getOut: existingDV(103), err: storage.ErrNotFound}
	req := authedRequest(http.MethodPatch, "/duty-violations/9999", validCreateBody(), 1)
	req = withID(req, "9999")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", rec.Code)
	}
}

func TestEdit_BadID(t *testing.T) {
	svc := &mockUpdater{}
	req := authedRequest(http.MethodPatch, "/duty-violations/abc", validCreateBody(), 1)
	req = withID(req, "abc")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

func TestEdit_FKViolation_422(t *testing.T) {
	svc := &mockUpdater{getOut: existingDV(103), err: storage.ErrForeignKeyViolation}
	req := authedRequest(http.MethodPatch, "/duty-violations/5", validCreateBody(), 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: want 422, got %d", rec.Code)
	}
}

func TestEdit_CheckConstraintViolation_400(t *testing.T) {
	svc := &mockUpdater{getOut: existingDV(103), err: storage.ErrCheckConstraintViolation}
	req := authedRequest(http.MethodPatch, "/duty-violations/5", validCreateBody(), 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

// IDOR defense: caller's claims grant org 50; the EXISTING record belongs
// to org 999. Even though the request body says organization_id=103 (which
// they also don't own), the handler must reject based on the existing
// org, NOT the body. Update MUST NOT be called.
func TestEdit_IDOR_PreReadDeniesForForeignOrg(t *testing.T) {
	svc := &mockUpdater{getOut: existingDV(999)}
	req := httptest.NewRequest(http.MethodPatch, "/duty-violations/5",
		strings.NewReader(validCreateBody()))
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
	if svc.updateCalled {
		t.Errorf("Update must not run when pre-read fails the org check")
	}
}

// IDOR via reassignment: caller owns the existing record's org (50) but
// tries to MOVE the record to org 999 they don't own. Must reject — even
// though the caller is allowed to read+edit the record in its current org.
func TestEdit_IDOR_RejectsTransferToForeignOrg(t *testing.T) {
	svc := &mockUpdater{getOut: existingDV(50)}
	body := strings.Replace(validCreateBody(), `"organization_id": 103`, `"organization_id": 999`, 1)
	req := httptest.NewRequest(http.MethodPatch, "/duty-violations/5", strings.NewReader(body))
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
	if svc.updateCalled {
		t.Errorf("Update must not run when transfer target is foreign")
	}
}

// sc role can reassign freely — they have access to every org. Sanity
// check that the IDOR guard doesn't accidentally block privileged users.
func TestEdit_SCCanReassign(t *testing.T) {
	svc := &mockUpdater{
		getOut: existingDV(50),
		out:    &dvmodel.DutyViolation{ID: 5, OrganizationID: 999},
	}
	body := strings.Replace(validCreateBody(), `"organization_id": 103`, `"organization_id": 999`, 1)
	req := authedRequest(http.MethodPatch, "/duty-violations/5", body, 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("sc must be able to reassign: status %d, body %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE /duty-violations/{id} ---

func TestDelete_HappyPath(t *testing.T) {
	svc := &mockDeleter{getOut: existingDV(103)}
	req := authedRequest(http.MethodDelete, "/duty-violations/5", "", 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status: want 204, got %d", rec.Code)
	}
	if !svc.deleteCalled {
		t.Errorf("Delete must run on happy path")
	}
}

// Pre-read ErrNotFound surfaces as 404 — Delete is not attempted.
func TestDelete_NotFoundOnPreRead(t *testing.T) {
	svc := &mockDeleter{getErr: storage.ErrNotFound}
	req := authedRequest(http.MethodDelete, "/duty-violations/9999", "", 1)
	req = withID(req, "9999")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", rec.Code)
	}
	if svc.deleteCalled {
		t.Errorf("Delete must not run when pre-read fails")
	}
}

func TestDelete_BadID(t *testing.T) {
	svc := &mockDeleter{}
	req := authedRequest(http.MethodDelete, "/duty-violations/0", "", 1)
	req = withID(req, "0")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

func TestDelete_ServiceError_500(t *testing.T) {
	svc := &mockDeleter{getOut: existingDV(103), err: errors.New("db down")}
	req := authedRequest(http.MethodDelete, "/duty-violations/5", "", 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", rec.Code)
	}
}

// IDOR defense: a caller targeting a record they don't own gets 403 and
// the row is NOT deleted.
func TestDelete_IDOR_RejectsForeignOrg(t *testing.T) {
	svc := &mockDeleter{getOut: existingDV(999)}
	req := httptest.NewRequest(http.MethodDelete, "/duty-violations/5", nil)
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
	if svc.deleteCalled {
		t.Errorf("Delete must NOT run for foreign org")
	}
}
