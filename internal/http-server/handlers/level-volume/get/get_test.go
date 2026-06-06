package get

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/model/levelvolume"
	"srmt-admin/internal/token"
)

type mockGetter struct {
	called bool
	resp   *levelvolume.Model
}

func (m *mockGetter) GetLevelVolume(_ context.Context, orgID int64, level float64) (*levelvolume.Model, error) {
	m.called = true
	if m.resp != nil {
		return m.resp, nil
	}
	return &levelvolume.Model{OrganizationID: orgID, Level: level, Volume: 100}, nil
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func makeReq(orgID, level string, claims *token.Claims) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/level-volume?id="+orgID+"&level="+level, nil)
	if claims != nil {
		req = req.WithContext(mwauth.ContextWithClaims(req.Context(), claims))
	}
	return req
}

// sc role can query any organization — no org-membership check.
func TestGet_SCRole_AnyOrg(t *testing.T) {
	getter := &mockGetter{}
	req := makeReq("99", "100.5", &token.Claims{UserID: 1, Roles: []string{"sc"}})
	rec := httptest.NewRecorder()

	New(quietLog(), getter)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
	if !getter.called {
		t.Errorf("repo must be called for sc")
	}
}

// reservoir role querying its own org gets the data.
func TestGet_ReservoirRole_OwnOrg(t *testing.T) {
	getter := &mockGetter{}
	req := makeReq("42", "100.5", &token.Claims{
		UserID:          1,
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{42},
	})
	rec := httptest.NewRecorder()

	New(quietLog(), getter)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	if !getter.called {
		t.Errorf("repo must be called for own-org reservoir request")
	}
}

// reservoir role querying ANOTHER organization is blocked with 403 before
// the repo is touched. This is the security guarantee of this PR.
func TestGet_ReservoirRole_OtherOrg_Forbidden(t *testing.T) {
	getter := &mockGetter{}
	req := makeReq("99", "100.5", &token.Claims{
		UserID:          1,
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{42},
	})
	rec := httptest.NewRecorder()

	New(quietLog(), getter)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d (body %s)", rec.Code, rec.Body.String())
	}
	if getter.called {
		t.Errorf("repo MUST NOT be called when access is denied — leak risk")
	}
}

// reservoir role with no orgs in claims is rejected; repo is not touched.
func TestGet_ReservoirRole_EmptyOrgs_Forbidden(t *testing.T) {
	getter := &mockGetter{}
	req := makeReq("42", "100.5", &token.Claims{
		UserID:          1,
		Roles:           []string{"reservoir"},
		OrganizationIDs: nil,
	})
	rec := httptest.NewRecorder()

	New(quietLog(), getter)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", rec.Code)
	}
	if getter.called {
		t.Errorf("repo MUST NOT be called for no-orgs reservoir")
	}
}
