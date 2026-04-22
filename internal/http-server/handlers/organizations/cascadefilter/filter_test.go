package cascadefilter_test

import (
	"context"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/http-server/handlers/organizations/cascadefilter"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/token"
	"testing"
)

func int64Ptr(i int64) *int64 {
	return &i
}

func ctxWith(role string, orgID int64) context.Context {
	claims := &token.Claims{
		UserID:         1,
		OrganizationID: orgID,
		Name:           "Test User",
		Roles:          []string{role},
	}
	return mwauth.ContextWithClaims(context.Background(), claims)
}

func ids(list []*organization.Model) []int64 {
	out := make([]int64, 0, len(list))
	for _, o := range list {
		out = append(out, o.ID)
	}
	return out
}

func TestApply_NoClaims_Passthrough(t *testing.T) {
	orgs := []*organization.Model{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	got := cascadefilter.Apply(context.Background(), orgs)
	if len(got) != 3 {
		t.Errorf("expected passthrough (3 orgs), got %d: %v", len(got), ids(got))
	}
}

func TestApply_ScRole_Passthrough(t *testing.T) {
	orgs := []*organization.Model{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	got := cascadefilter.Apply(ctxWith("sc", 5), orgs)
	if len(got) != 3 {
		t.Errorf("expected passthrough (3 orgs), got %d: %v", len(got), ids(got))
	}
}

func TestApply_RaisRole_Passthrough(t *testing.T) {
	orgs := []*organization.Model{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	got := cascadefilter.Apply(ctxWith("rais", 5), orgs)
	if len(got) != 3 {
		t.Errorf("expected passthrough (3 orgs), got %d: %v", len(got), ids(got))
	}
}

func TestApply_NonCascadeRole_Passthrough(t *testing.T) {
	orgs := []*organization.Model{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	// Deliberately do NOT filter non-cascade, non-sc roles.
	got := cascadefilter.Apply(ctxWith("reservoir", 2), orgs)
	if len(got) != 3 {
		t.Errorf("expected passthrough (3 orgs), got %d: %v", len(got), ids(got))
	}
}

func TestApply_CascadeRole_FlatStations(t *testing.T) {
	orgs := []*organization.Model{
		{ID: 5, ParentOrganizationID: nil},
		{ID: 10, ParentOrganizationID: int64Ptr(5)},
		{ID: 11, ParentOrganizationID: int64Ptr(5)},
		{ID: 20, ParentOrganizationID: int64Ptr(7)},
		{ID: 7, ParentOrganizationID: nil},
	}
	got := cascadefilter.Apply(ctxWith("cascade", 5), orgs)

	gotIDs := ids(got)
	wantIDs := map[int64]bool{5: true, 10: true, 11: true}
	if len(gotIDs) != 3 {
		t.Fatalf("expected 3 orgs, got %d: %v", len(gotIDs), gotIDs)
	}
	for _, id := range gotIDs {
		if !wantIDs[id] {
			t.Errorf("unexpected org id %d in result (want only 5, 10, 11): %v", id, gotIDs)
		}
	}
}

func TestApply_CascadeRole_Tree(t *testing.T) {
	// Two roots: 5 with children [10, 11], 7 with child [20].
	orgs := []*organization.Model{
		{
			ID:                   5,
			ParentOrganizationID: nil,
			Items: []*organization.Model{
				{ID: 10, ParentOrganizationID: int64Ptr(5)},
				{ID: 11, ParentOrganizationID: int64Ptr(5)},
			},
		},
		{
			ID:                   7,
			ParentOrganizationID: nil,
			Items: []*organization.Model{
				{ID: 20, ParentOrganizationID: int64Ptr(7)},
			},
		},
	}
	got := cascadefilter.Apply(ctxWith("cascade", 5), orgs)
	if len(got) != 1 {
		t.Fatalf("expected 1 root (id=5), got %d: %v", len(got), ids(got))
	}
	if got[0].ID != 5 {
		t.Errorf("expected root id=5, got %d", got[0].ID)
	}
	if len(got[0].Items) != 2 {
		t.Errorf("expected root 5 to keep its 2 items, got %d", len(got[0].Items))
	}
}

func TestApply_CascadeRole_OrgIDZero_EmptyResult(t *testing.T) {
	orgs := []*organization.Model{
		{ID: 5},
		{ID: 10, ParentOrganizationID: int64Ptr(5)},
	}
	got := cascadefilter.Apply(ctxWith("cascade", 0), orgs)
	if len(got) != 0 {
		t.Errorf("expected empty result for cascade user with orgID=0, got %d: %v", len(got), ids(got))
	}
}
