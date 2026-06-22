package reservoirsummary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	model "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/storage"
)

// --- mocks ---

type mockConfigUpserter struct {
	gotReq model.UpsertReservoirSummaryConfigRequest
	err    error
	calls  int
}

func (m *mockConfigUpserter) UpsertReservoirSummaryConfig(_ context.Context, req model.UpsertReservoirSummaryConfigRequest) error {
	m.calls++
	m.gotReq = req
	return m.err
}

type mockConfigGetter struct {
	configs []model.ReservoirSummaryConfig
	err     error
}

func (m *mockConfigGetter) GetAllReservoirSummaryConfigs(_ context.Context) ([]model.ReservoirSummaryConfig, error) {
	return m.configs, m.err
}

type mockConfigDeleter struct {
	gotOrgID int64
	err      error
	calls    int
}

func (m *mockConfigDeleter) DeleteReservoirSummaryConfig(_ context.Context, orgID int64) error {
	m.calls++
	m.gotOrgID = orgID
	return m.err
}

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- UpsertConfig ---

func TestUpsertConfig_HappyPath(t *testing.T) {
	repo := &mockConfigUpserter{}
	body := `{"organization_id":42,"sort_order":3,"include_in_total":true}`
	req := httptest.NewRequest(http.MethodPost, "/reservoir-summary/config", strings.NewReader(body))
	rec := httptest.NewRecorder()

	UpsertConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
	if repo.calls != 1 {
		t.Errorf("repo calls: want 1, got %d", repo.calls)
	}
	if repo.gotReq.OrganizationID != 42 || repo.gotReq.SortOrder != 3 || !repo.gotReq.IncludeInTotal {
		t.Errorf("repo received wrong req: %+v", repo.gotReq)
	}
}

// TestUpsertConfig_ForwardsModsnowEnabled is a regression guard: the
// handler is a thin pass-through, so the field is forwarded automatically
// once it exists on the request struct (added in commit "reservoir-summary:
// add modsnow_enabled to config"). Pin both branches so a future refactor
// that introduces request-side defaulting / coercion of the flag can't
// silently flip the bit on the way to the repo.
func TestUpsertConfig_ForwardsModsnowEnabled(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"false explicit", `{"organization_id":42,"sort_order":3,"include_in_total":true,"modsnow_enabled":false}`, false},
		{"true explicit", `{"organization_id":42,"sort_order":3,"include_in_total":true,"modsnow_enabled":true}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockConfigUpserter{}
			req := httptest.NewRequest(http.MethodPost, "/reservoir-summary/config", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			UpsertConfig(quietLog(), repo)(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
			}
			if repo.gotReq.ModsnowEnabled != tc.want {
				t.Errorf("ModsnowEnabled forwarded to repo: want %v, got %v (req=%+v)", tc.want, repo.gotReq.ModsnowEnabled, repo.gotReq)
			}
		})
	}
}

func TestUpsertConfig_InvalidJSON(t *testing.T) {
	repo := &mockConfigUpserter{}
	req := httptest.NewRequest(http.MethodPost, "/reservoir-summary/config", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()

	UpsertConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", rec.Code)
	}
	if repo.calls != 0 {
		t.Errorf("repo must not be called on bad JSON; got %d calls", repo.calls)
	}
}

func TestUpsertConfig_ValidationFailure(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing organization_id", `{"sort_order":1,"include_in_total":true}`},
		{"organization_id = 0", `{"organization_id":0,"sort_order":1,"include_in_total":true}`},
		{"negative sort_order", `{"organization_id":1,"sort_order":-1,"include_in_total":true}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockConfigUpserter{}
			req := httptest.NewRequest(http.MethodPost, "/reservoir-summary/config", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			UpsertConfig(quietLog(), repo)(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status: want 400, got %d", rec.Code)
			}
			if repo.calls != 0 {
				t.Errorf("repo must not be called on validation failure; got %d", repo.calls)
			}
		})
	}
}

func TestUpsertConfig_RepoError(t *testing.T) {
	repo := &mockConfigUpserter{err: errors.New("db down")}
	body := `{"organization_id":42,"sort_order":3,"include_in_total":true}`
	req := httptest.NewRequest(http.MethodPost, "/reservoir-summary/config", strings.NewReader(body))
	rec := httptest.NewRecorder()

	UpsertConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", rec.Code)
	}
}

// FK violation on organization_id (non-existent org) must return 422 with
// an actionable message — not the generic 500 that hides the cause.
func TestUpsertConfig_NonExistentOrgReturns422(t *testing.T) {
	repo := &mockConfigUpserter{err: storage.ErrForeignKeyViolation}
	body := `{"organization_id":99999,"sort_order":3,"include_in_total":true}`
	req := httptest.NewRequest(http.MethodPost, "/reservoir-summary/config", strings.NewReader(body))
	rec := httptest.NewRecorder()

	UpsertConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: want 422, got %d", rec.Code)
	}
}

// --- GetConfigs ---

// Without claims in context the handler falls back to "no access" (empty list).
// Verifies the security default: missing auth = empty result, never leak.
func TestGetConfigs_NoAuthReturnsEmpty(t *testing.T) {
	repo := &mockConfigGetter{
		configs: []model.ReservoirSummaryConfig{
			{ID: 1, OrganizationID: 10, OrganizationName: "Андижон", SortOrder: 1, IncludeInTotal: true},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary/config", nil)
	rec := httptest.NewRecorder()

	GetConfigs(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var got []model.ReservoirSummaryConfig
	if err := json.Unmarshal(bytes.TrimSpace(rec.Body.Bytes()), &got); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	if len(got) != 0 {
		t.Errorf("expected empty list (no claims = no access), got %d items", len(got))
	}
}

func TestGetConfigs_RepoError(t *testing.T) {
	repo := &mockConfigGetter{err: errors.New("db down")}
	req := httptest.NewRequest(http.MethodGet, "/reservoir-summary/config", nil)
	rec := httptest.NewRecorder()

	GetConfigs(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", rec.Code)
	}
}

// --- DeleteConfig ---

func TestDeleteConfig_HappyPath(t *testing.T) {
	repo := &mockConfigDeleter{}
	req := httptest.NewRequest(http.MethodDelete, "/reservoir-summary/config?organization_id=42", nil)
	rec := httptest.NewRecorder()

	DeleteConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status: want 204, got %d", rec.Code)
	}
	if repo.calls != 1 || repo.gotOrgID != 42 {
		t.Errorf("repo not called correctly: calls=%d orgID=%d", repo.calls, repo.gotOrgID)
	}
}

func TestDeleteConfig_MissingParam(t *testing.T) {
	cases := []struct{ name, url string }{
		{"empty", "/reservoir-summary/config"},
		{"non-numeric", "/reservoir-summary/config?organization_id=abc"},
		{"zero", "/reservoir-summary/config?organization_id=0"},
		{"negative", "/reservoir-summary/config?organization_id=-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockConfigDeleter{}
			req := httptest.NewRequest(http.MethodDelete, tc.url, nil)
			rec := httptest.NewRecorder()

			DeleteConfig(quietLog(), repo)(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status: want 400, got %d", rec.Code)
			}
			if repo.calls != 0 {
				t.Errorf("repo must not be called; got %d", repo.calls)
			}
		})
	}
}

func TestDeleteConfig_NotFound(t *testing.T) {
	repo := &mockConfigDeleter{err: storage.ErrNotFound}
	req := httptest.NewRequest(http.MethodDelete, "/reservoir-summary/config?organization_id=42", nil)
	rec := httptest.NewRecorder()

	DeleteConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", rec.Code)
	}
}

func TestDeleteConfig_RepoError(t *testing.T) {
	repo := &mockConfigDeleter{err: errors.New("db down")}
	req := httptest.NewRequest(http.MethodDelete, "/reservoir-summary/config?organization_id=42", nil)
	rec := httptest.NewRecorder()

	DeleteConfig(quietLog(), repo)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", rec.Code)
	}
}
