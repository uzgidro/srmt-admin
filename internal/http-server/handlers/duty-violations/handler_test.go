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
	out []*dvmodel.DutyViolation
	err error
	got dvmodel.ListFilter
}

func (m *mockLister) List(_ context.Context, f dvmodel.ListFilter) ([]*dvmodel.DutyViolation, error) {
	m.got = f
	return m.out, m.err
}

type mockUpdater struct {
	out *dvmodel.DutyViolation
	err error
}

func (m *mockUpdater) Update(_ context.Context, _ int64, _ dvmodel.UpdateRequest) (*dvmodel.DutyViolation, error) {
	return m.out, m.err
}

type mockDeleter struct {
	err error
}

func (m *mockDeleter) Delete(_ context.Context, _ int64) error { return m.err }

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

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
	svc := &mockLister{out: []*dvmodel.DutyViolation{{ID: 1}, {ID: 2}}}
	req := authedRequest(http.MethodGet, "/duty-violations", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []*dvmodel.DutyViolation
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 rows, got %d", len(got))
	}
}

// Confirms the handler forwards org_id, from and to to the service intact.
// Catches off-by-one parser bugs that would silently drop a filter.
func TestList_ForwardsAllFilters(t *testing.T) {
	svc := &mockLister{out: nil}
	req := authedRequest(http.MethodGet,
		"/duty-violations?organization_id=42&from=2026-06-01&to=2026-06-30", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc)(rec, req)

	if svc.got.OrganizationID == nil || *svc.got.OrganizationID != 42 {
		t.Errorf("org filter not forwarded: %+v", svc.got)
	}
	if svc.got.From == nil || !svc.got.From.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("from filter wrong: %+v", svc.got.From)
	}
	if svc.got.To == nil {
		t.Error("to filter not forwarded")
	}
}

func TestList_InvalidFilter_400(t *testing.T) {
	svc := &mockLister{}
	req := authedRequest(http.MethodGet, "/duty-violations?organization_id=abc", "", 1)
	rec := httptest.NewRecorder()
	List(quietLog(), svc)(rec, req)
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
	List(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("want `[]`, got %q", body)
	}
}

// --- PATCH /duty-violations/{id} ---

func TestEdit_HappyPath(t *testing.T) {
	svc := &mockUpdater{out: &dvmodel.DutyViolation{ID: 5, OrganizationID: 103}}
	req := authedRequest(http.MethodPatch, "/duty-violations/5", validCreateBody(), 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
}

func TestEdit_NotFound(t *testing.T) {
	svc := &mockUpdater{err: storage.ErrNotFound}
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
	svc := &mockUpdater{err: storage.ErrForeignKeyViolation}
	req := authedRequest(http.MethodPatch, "/duty-violations/5", validCreateBody(), 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: want 422, got %d", rec.Code)
	}
}

func TestEdit_CheckConstraintViolation_400(t *testing.T) {
	svc := &mockUpdater{err: storage.ErrCheckConstraintViolation}
	req := authedRequest(http.MethodPatch, "/duty-violations/5", validCreateBody(), 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
}

func TestEdit_OrgAccessDenied(t *testing.T) {
	svc := &mockUpdater{out: &dvmodel.DutyViolation{ID: 5}}
	req := httptest.NewRequest(http.MethodPatch, "/duty-violations/5", strings.NewReader(validCreateBody()))
	req = req.WithContext(mwauth.ContextWithClaims(req.Context(),
		&token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationIDs: []int64{50}}))
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Edit(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
}

// --- DELETE /duty-violations/{id} ---

func TestDelete_HappyPath(t *testing.T) {
	svc := &mockDeleter{}
	req := authedRequest(http.MethodDelete, "/duty-violations/5", "", 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status: want 204, got %d", rec.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := &mockDeleter{err: storage.ErrNotFound}
	req := authedRequest(http.MethodDelete, "/duty-violations/9999", "", 1)
	req = withID(req, "9999")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", rec.Code)
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
	svc := &mockDeleter{err: errors.New("db down")}
	req := authedRequest(http.MethodDelete, "/duty-violations/5", "", 1)
	req = withID(req, "5")
	rec := httptest.NewRecorder()
	Delete(quietLog(), svc)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", rec.Code)
	}
}
