package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

// ---------- mockCascadeChecker ----------

type mockCascadeChecker struct {
	parents map[int64]*int64 // orgID → parentID
}

func (m *mockCascadeChecker) GetOrganizationParentID(_ context.Context, orgID int64) (*int64, error) {
	if p, ok := m.parents[orgID]; ok {
		return p, nil
	}
	return nil, storage.ErrNotFound
}

func contextWithClaims(claims *token.Claims) context.Context {
	return mwauth.ContextWithClaims(context.Background(), claims)
}

// ---------- ContainsOrg ----------

func TestContainsOrg(t *testing.T) {
	cases := []struct {
		name string
		ids  []int64
		id   int64
		want bool
	}{
		{"empty slice", nil, 5, false},
		{"found", []int64{5, 10}, 10, true},
		{"not found", []int64{5, 10}, 7, false},
		{"single match", []int64{5}, 5, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ContainsOrg(tc.ids, tc.id); got != tc.want {
				t.Errorf("ContainsOrg(%v, %d): want %v, got %v", tc.ids, tc.id, tc.want, got)
			}
		})
	}
}

// ---------- CheckOrgAccess ----------

func TestCheckOrgAccess_SCRole_FullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"sc"},
		OrganizationIDs: []int64{1},
	})

	if err := CheckOrgAccess(ctx, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccess_RaisRole_FullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"rais"},
		OrganizationIDs: []int64{1},
	})

	if err := CheckOrgAccess(ctx, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccess_ReservoirRole_OwnOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	if err := CheckOrgAccess(ctx, 5); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccess_ReservoirRole_DifferentOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	err := CheckOrgAccess(ctx, 10)
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCheckOrgAccess_ReservoirRole_NoOrgAssigned(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: nil,
	})

	err := CheckOrgAccess(ctx, 5)
	if err != ErrNoOrganization {
		t.Fatalf("expected ErrNoOrganization, got %v", err)
	}
}

func TestCheckOrgAccess_NoClaimsInContext(t *testing.T) {
	ctx := context.Background()

	err := CheckOrgAccess(ctx, 5)
	if err != ErrClaimsNotFound {
		t.Fatalf("expected ErrClaimsNotFound, got %v", err)
	}
}

func TestCheckOrgAccess_ResourceOrgIDZero(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	err := CheckOrgAccess(ctx, 0)
	if err != ErrNoOrganization {
		t.Fatalf("expected ErrNoOrganization, got %v", err)
	}
}

func TestCheckOrgAccess_MultipleRolesWithSC(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir", "sc"},
		OrganizationIDs: []int64{5},
	})

	if err := CheckOrgAccess(ctx, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// Multi-org: a user bound to several organizations has access to each of them
// and is denied for any organization outside the list.
func TestCheckOrgAccess_MultiOrg_MemberAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5, 10},
	})

	if err := CheckOrgAccess(ctx, 5); err != nil {
		t.Errorf("org 5 (member): expected nil, got %v", err)
	}
	if err := CheckOrgAccess(ctx, 10); err != nil {
		t.Errorf("org 10 (member): expected nil, got %v", err)
	}
	if err := CheckOrgAccess(ctx, 7); err != ErrForbidden {
		t.Errorf("org 7 (non-member): expected ErrForbidden, got %v", err)
	}
}

// ---------- CheckOrgAccessBatch ----------

func TestCheckOrgAccessBatch_SCRole_AllOrgs(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"sc"},
		OrganizationIDs: []int64{1},
	})

	if err := CheckOrgAccessBatch(ctx, []int64{1, 2, 3, 999}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccessBatch_ReservoirRole_OwnOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	if err := CheckOrgAccessBatch(ctx, []int64{5, 5, 5}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccessBatch_ReservoirRole_ForeignOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	err := CheckOrgAccessBatch(ctx, []int64{5, 10})
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCheckOrgAccessBatch_EmptySlice(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	if err := CheckOrgAccessBatch(ctx, []int64{}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccessBatch_DuplicatesCheckedOnce(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})

	// All duplicates of own org — should pass
	if err := CheckOrgAccessBatch(ctx, []int64{5, 5, 5, 5}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// Multi-org batch: every requested org is in the user's list.
func TestCheckOrgAccessBatch_MultiOrg_AllMembers(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5, 10},
	})

	if err := CheckOrgAccessBatch(ctx, []int64{5, 10, 5}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := CheckOrgAccessBatch(ctx, []int64{5, 7}); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden for non-member 7, got %v", err)
	}
}

// ---------- GetOrganizationIDs ----------

func TestGetOrganizationIDs_HasOrgs(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		OrganizationIDs: []int64{5, 10},
	})

	ids, err := GetOrganizationIDs(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(ids) != 2 || ids[0] != 5 || ids[1] != 10 {
		t.Fatalf("expected [5 10], got %v", ids)
	}
}

func TestGetOrganizationIDs_NoOrgs(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		OrganizationIDs: nil,
	})

	ids, err := GetOrganizationIDs(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty, got %v", ids)
	}
}

func TestGetOrganizationIDs_NoClaims(t *testing.T) {
	ctx := context.Background()

	ids, err := GetOrganizationIDs(ctx)
	if err != ErrClaimsNotFound {
		t.Fatalf("expected ErrClaimsNotFound, got %v", err)
	}
	if ids != nil {
		t.Fatalf("expected nil, got %v", ids)
	}
}

// ---------- CheckCascadeStationAccess ----------

func ptr(v int64) *int64 { return &v }

func TestCheckCascadeStationAccess_ScFullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"sc"},
		OrganizationIDs: []int64{1},
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	if err := CheckCascadeStationAccess(ctx, 999, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckCascadeStationAccess_RaisFullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"rais"},
		OrganizationIDs: []int64{1},
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	if err := CheckCascadeStationAccess(ctx, 999, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckCascadeStationAccess_CascadeOwnStation(t *testing.T) {
	cascadeOrgID := int64(10)
	stationOrgID := int64(20)

	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{cascadeOrgID},
	})
	checker := &mockCascadeChecker{
		parents: map[int64]*int64{
			stationOrgID: ptr(cascadeOrgID), // station's parent is the cascade org
		},
	}

	if err := CheckCascadeStationAccess(ctx, stationOrgID, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckCascadeStationAccess_CascadeSelfOrg(t *testing.T) {
	cascadeOrgID := int64(10)

	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{cascadeOrgID},
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	// stationOrgID in claims.OrganizationIDs → allowed without parent lookup
	if err := CheckCascadeStationAccess(ctx, cascadeOrgID, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckCascadeStationAccess_CascadeForeignStation(t *testing.T) {
	cascadeOrgID := int64(10)
	stationOrgID := int64(20)
	otherCascade := int64(99)

	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{cascadeOrgID},
	})
	checker := &mockCascadeChecker{
		parents: map[int64]*int64{
			stationOrgID: ptr(otherCascade), // belongs to a different cascade
		},
	}

	err := CheckCascadeStationAccess(ctx, stationOrgID, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// Multi-org cascade: a user bound to two cascades may access stations of
// either cascade (matched via parent_org_id), and is denied for a third.
func TestCheckCascadeStationAccess_CascadeMultiOrg(t *testing.T) {
	cascadeA := int64(10)
	cascadeB := int64(20)
	stationOfB := int64(25)
	stationOfThird := int64(35)
	thirdCascade := int64(99)

	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{cascadeA, cascadeB},
	})
	checker := &mockCascadeChecker{
		parents: map[int64]*int64{
			stationOfB:     ptr(cascadeB),
			stationOfThird: ptr(thirdCascade),
		},
	}

	if err := CheckCascadeStationAccess(ctx, stationOfB, checker); err != nil {
		t.Errorf("station of cascade B: expected nil, got %v", err)
	}
	if err := CheckCascadeStationAccess(ctx, cascadeA, checker); err != nil {
		t.Errorf("cascade A self: expected nil, got %v", err)
	}
	err := CheckCascadeStationAccess(ctx, stationOfThird, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("station of third cascade: expected ErrForbidden, got %v", err)
	}
}

// A station org with no registry entry (parent lookup returns ErrNotFound)
// must read as ErrForbidden for a cascade user, not leak storage.ErrNotFound.
func TestCheckCascadeStationAccess_CascadeUnknownStation(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{10},
	})
	// Empty parents map → mock returns storage.ErrNotFound for any orgID.
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	err := CheckCascadeStationAccess(ctx, 20, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for unknown station, got %v", err)
	}
}

// stationOrgID == 0 must short-circuit to ErrNoOrganization before any role
// branch or DB lookup.
func TestCheckCascadeStationAccess_ZeroStationID(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{10},
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	err := CheckCascadeStationAccess(ctx, 0, checker)
	if err != ErrNoOrganization {
		t.Fatalf("expected ErrNoOrganization, got %v", err)
	}
}

func TestCheckCascadeStationAccess_DefaultFallback(t *testing.T) {
	// A role that is not sc/rais/cascade falls back to CheckOrgAccess
	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"reservoir"},
		OrganizationIDs: []int64{5},
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	// Own org — should pass via CheckOrgAccess
	if err := CheckCascadeStationAccess(ctx, 5, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	// Foreign org — should fail via CheckOrgAccess
	err := CheckCascadeStationAccess(ctx, 999, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ---------- CheckCascadeStationAccessBatch ----------

func TestCheckCascadeStationAccessBatch_MixedAccess(t *testing.T) {
	cascadeOrgID := int64(10)
	ownStation := int64(20)
	foreignStation := int64(30)
	otherCascade := int64(99)

	ctx := contextWithClaims(&token.Claims{
		Roles:           []string{"cascade"},
		OrganizationIDs: []int64{cascadeOrgID},
	})
	checker := &mockCascadeChecker{
		parents: map[int64]*int64{
			ownStation:     ptr(cascadeOrgID),
			foreignStation: ptr(otherCascade),
		},
	}

	err := CheckCascadeStationAccessBatch(ctx, []int64{ownStation, foreignStation}, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	// All own stations — should pass
	if err := CheckCascadeStationAccessBatch(ctx, []int64{ownStation, cascadeOrgID}, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// ---------- mockOwnerChecker ----------

// mockOwnerChecker implements auth.OwnerChecker for ownership tests.
// `owner` is returned for any shutdown ID; `err` (if set) overrides.
type mockOwnerChecker struct {
	owner sql.NullInt64
	err   error
	calls int
}

func (m *mockOwnerChecker) GetShutdownCreatedByUserID(_ context.Context, _ int64) (sql.NullInt64, error) {
	m.calls++
	if m.err != nil {
		return sql.NullInt64{}, m.err
	}
	return m.owner, nil
}

// ---------- CheckShutdownOwnership ----------

func TestCheckShutdownOwnership_SCRoleBypasses(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 1, Roles: []string{"sc"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 99, Valid: true}}

	if err := CheckShutdownOwnership(ctx, 42, checker); err != nil {
		t.Fatalf("expected nil for sc role, got %v", err)
	}
	if checker.calls != 0 {
		t.Errorf("repo must NOT be called for sc role; got %d calls", checker.calls)
	}
}

func TestCheckShutdownOwnership_RaisRoleBypasses(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 1, Roles: []string{"rais"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 99, Valid: true}}

	if err := CheckShutdownOwnership(ctx, 42, checker); err != nil {
		t.Fatalf("expected nil for rais role, got %v", err)
	}
	if checker.calls != 0 {
		t.Errorf("repo must NOT be called for rais role; got %d calls", checker.calls)
	}
}

func TestCheckShutdownOwnership_CascadeOwnSuccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationIDs: []int64{1}, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 10, Valid: true}}

	if err := CheckShutdownOwnership(ctx, 42, checker); err != nil {
		t.Fatalf("expected nil for cascade owner, got %v", err)
	}
}

func TestCheckShutdownOwnership_CascadeForeignForbidden(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationIDs: []int64{1}, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 11, Valid: true}}

	err := CheckShutdownOwnership(ctx, 42, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cascade non-owner, got %v", err)
	}
}

func TestCheckShutdownOwnership_CascadeNullOwnerForbidden(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationIDs: []int64{1}, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Valid: false}}

	err := CheckShutdownOwnership(ctx, 42, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cascade on orphaned record, got %v", err)
	}
}

func TestCheckShutdownOwnership_NotFoundPropagates(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationIDs: []int64{1}, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{err: storage.ErrNotFound}

	err := CheckShutdownOwnership(ctx, 42, checker)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCheckShutdownOwnership_NoClaims(t *testing.T) {
	ctx := context.Background()
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 1, Valid: true}}

	err := CheckShutdownOwnership(ctx, 42, checker)
	if !errors.Is(err, ErrClaimsNotFound) {
		t.Fatalf("expected ErrClaimsNotFound, got %v", err)
	}
	if checker.calls != 0 {
		t.Errorf("repo must NOT be called when claims absent; got %d calls", checker.calls)
	}
}

// Other roles (not sc/rais/cascade, e.g. reservoir) should not be subject to
// the ownership restriction — it is a cascade-specific add-on. The existing
// CheckOrgAccess at the route or handler level handles them.
func TestCheckShutdownOwnership_OtherRoleSkipsCheck(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationIDs: []int64{5}, Roles: []string{"reservoir"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 99, Valid: true}}

	if err := CheckShutdownOwnership(ctx, 42, checker); err != nil {
		t.Fatalf("expected nil for non-cascade non-sc role, got %v", err)
	}
	if checker.calls != 0 {
		t.Errorf("repo must NOT be called for non-cascade non-sc role; got %d calls", checker.calls)
	}
}
