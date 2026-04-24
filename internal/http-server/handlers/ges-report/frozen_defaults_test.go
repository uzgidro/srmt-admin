package gesreport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/token"
)

// captureFrozenRepo records calls and returns canned data for FrozenDefault
// handler tests. Implements all interfaces the three handlers need.
type captureFrozenRepo struct {
	mu sync.Mutex

	// Upsert
	upsertCalled int
	upsertReq    model.UpsertFrozenDefaultRequest
	upsertErr    error

	// Delete
	deleteCalled int
	deleteOrgID  int64
	deleteField  string
	deleteErr    error

	// Get
	listResult []model.FrozenDefault
	listErr    error

	// CheckCascadeStationAccess deps
	parents map[int64]*int64 // org -> parent_org_id (used by cascade access check)
}

func (c *captureFrozenRepo) UpsertFrozenDefault(_ context.Context, req model.UpsertFrozenDefaultRequest, _ int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upsertCalled++
	c.upsertReq = req
	return c.upsertErr
}

func (c *captureFrozenRepo) DeleteFrozenDefault(_ context.Context, orgID int64, field string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleteCalled++
	c.deleteOrgID = orgID
	c.deleteField = field
	return c.deleteErr
}

func (c *captureFrozenRepo) ListFrozenDefaults(_ context.Context) ([]model.FrozenDefault, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.listResult, c.listErr
}

func (c *captureFrozenRepo) GetOrganizationParentID(_ context.Context, orgID int64) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parents == nil {
		return nil, nil
	}
	return c.parents[orgID], nil
}

func newFrozenRouter(repo *captureFrozenRepo, claims *token.Claims) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	verifier := &mockTokenVerifier{claims: claims}
	r := chi.NewRouter()
	r.Use(mwauth.Authenticator(verifier))
	r.Put("/ges/frozen-defaults", UpsertFrozenDefault(log, repo))
	r.Delete("/ges/frozen-defaults", DeleteFrozenDefault(log, repo))
	r.Get("/ges/frozen-defaults", ListFrozenDefaults(log, repo))
	return r
}

func doFrozen(t *testing.T, repo *captureFrozenRepo, claims *token.Claims, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := newFrozenRouter(repo, claims)
	var bodyReader *bytes.Buffer
	if body == "" {
		bodyReader = bytes.NewBuffer(nil)
	} else {
		bodyReader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, "/ges/frozen-defaults", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func scClaimsFrozen() *token.Claims {
	return &token.Claims{UserID: 1, OrganizationID: 1, Roles: []string{"sc"}}
}

// TestUpsertFrozenDefault_InvalidFieldName_BadRequest — field_name="foo" → 400.
func TestUpsertFrozenDefault_InvalidFieldName_BadRequest(t *testing.T) {
	repo := &captureFrozenRepo{}
	body := `{"organization_id": 100, "field_name": "foo", "frozen_value": 1.0}`
	rr := doFrozen(t, repo, scClaimsFrozen(), http.MethodPut, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertCalled != 0 {
		t.Errorf("repo must not be called on validation failure; got %d calls", repo.upsertCalled)
	}
}

// TestUpsertFrozenDefault_NegativeValue_BadRequest — frozen_value=-1 → 400.
func TestUpsertFrozenDefault_NegativeValue_BadRequest(t *testing.T) {
	repo := &captureFrozenRepo{}
	body := `{"organization_id": 100, "field_name": "water_head_m", "frozen_value": -1.0}`
	rr := doFrozen(t, repo, scClaimsFrozen(), http.MethodPut, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertCalled != 0 {
		t.Errorf("repo must not be called on validation failure; got %d calls", repo.upsertCalled)
	}
}

// TestUpsertFrozenDefault_OK — happy path.
func TestUpsertFrozenDefault_OK(t *testing.T) {
	repo := &captureFrozenRepo{}
	body := `{"organization_id": 100, "field_name": "water_head_m", "frozen_value": 45.0}`
	rr := doFrozen(t, repo, scClaimsFrozen(), http.MethodPut, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertCalled != 1 {
		t.Fatalf("repo upsert calls: want 1, got %d", repo.upsertCalled)
	}
	if repo.upsertReq.OrganizationID != 100 ||
		repo.upsertReq.FieldName != "water_head_m" ||
		repo.upsertReq.FrozenValue != 45.0 {
		t.Errorf("upsert payload mismatch: %+v", repo.upsertReq)
	}
}

// TestUpsertFrozenDefault_CascadeUserOtherOrg_Forbidden — cascade-юзер пытается
// заморозить чужую org → 403.
func TestUpsertFrozenDefault_CascadeUserOtherOrg_Forbidden(t *testing.T) {
	otherParent := int64(999)
	repo := &captureFrozenRepo{
		parents: map[int64]*int64{200: &otherParent},
	}
	cascadeClaims := &token.Claims{UserID: 5, OrganizationID: 1, Roles: []string{"cascade"}}
	body := `{"organization_id": 200, "field_name": "water_head_m", "frozen_value": 45.0}`
	rr := doFrozen(t, repo, cascadeClaims, http.MethodPut, body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertCalled != 0 {
		t.Errorf("repo must not be called when access denied; got %d calls", repo.upsertCalled)
	}
}

// TestUpsertFrozenDefault_NonIntegerForAggregates_BadRequest — frozen_value=3.7
// для working_aggregates → 400 (агрегаты должны быть целыми).
func TestUpsertFrozenDefault_NonIntegerForAggregates_BadRequest(t *testing.T) {
	repo := &captureFrozenRepo{}
	body := `{"organization_id": 100, "field_name": "working_aggregates", "frozen_value": 3.7}`
	rr := doFrozen(t, repo, scClaimsFrozen(), http.MethodPut, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if repo.upsertCalled != 0 {
		t.Errorf("repo must not be called when aggregate value is non-integer; got %d calls", repo.upsertCalled)
	}
}

// TestDeleteFrozenDefault_OK — happy path delete.
func TestDeleteFrozenDefault_OK(t *testing.T) {
	repo := &captureFrozenRepo{}
	body := `{"organization_id": 100, "field_name": "water_head_m"}`
	rr := doFrozen(t, repo, scClaimsFrozen(), http.MethodDelete, body)
	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Fatalf("status: want 204 or 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if repo.deleteCalled != 1 {
		t.Fatalf("repo delete calls: want 1, got %d", repo.deleteCalled)
	}
	if repo.deleteOrgID != 100 || repo.deleteField != "water_head_m" {
		t.Errorf("delete params mismatch: org=%d field=%q", repo.deleteOrgID, repo.deleteField)
	}
}

// TestListFrozenDefaults_OK — GET returns the list as JSON.
func TestListFrozenDefaults_OK(t *testing.T) {
	repo := &captureFrozenRepo{
		listResult: []model.FrozenDefault{
			{OrganizationID: 100, FieldName: "water_head_m", FrozenValue: 45.0},
			{OrganizationID: 200, FieldName: "working_aggregates", FrozenValue: 3.0},
		},
	}
	rr := doFrozen(t, repo, scClaimsFrozen(), http.MethodGet, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var got []model.FrozenDefault
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, rr.Body.String())
	}
	if len(got) != 2 {
		t.Fatalf("len(got): want 2, got %d", len(got))
	}
}
