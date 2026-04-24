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

// ---------- CheckOrgAccess ----------

func TestCheckOrgAccess_SCRole_FullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"sc"},
		OrganizationID: 1,
	})

	if err := CheckOrgAccess(ctx, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccess_RaisRole_FullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"rais"},
		OrganizationID: 1,
	})

	if err := CheckOrgAccess(ctx, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccess_ReservoirRole_OwnOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	if err := CheckOrgAccess(ctx, 5); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccess_ReservoirRole_DifferentOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	err := CheckOrgAccess(ctx, 10)
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCheckOrgAccess_ReservoirRole_NoOrgAssigned(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 0,
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
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	err := CheckOrgAccess(ctx, 0)
	if err != ErrNoOrganization {
		t.Fatalf("expected ErrNoOrganization, got %v", err)
	}
}

func TestCheckOrgAccess_MultipleRolesWithSC(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir", "sc"},
		OrganizationID: 5,
	})

	if err := CheckOrgAccess(ctx, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// ---------- CheckOrgAccessBatch ----------

func TestCheckOrgAccessBatch_SCRole_AllOrgs(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"sc"},
		OrganizationID: 1,
	})

	if err := CheckOrgAccessBatch(ctx, []int64{1, 2, 3, 999}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccessBatch_ReservoirRole_OwnOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	if err := CheckOrgAccessBatch(ctx, []int64{5, 5, 5}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccessBatch_ReservoirRole_ForeignOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	err := CheckOrgAccessBatch(ctx, []int64{5, 10})
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCheckOrgAccessBatch_EmptySlice(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	if err := CheckOrgAccessBatch(ctx, []int64{}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckOrgAccessBatch_DuplicatesCheckedOnce(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
	})

	// All duplicates of own org — should pass
	if err := CheckOrgAccessBatch(ctx, []int64{5, 5, 5, 5}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// ---------- GetOrganizationID ----------

func TestGetOrganizationID_HasOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		OrganizationID: 5,
	})

	orgID, err := GetOrganizationID(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if orgID != 5 {
		t.Fatalf("expected 5, got %d", orgID)
	}
}

func TestGetOrganizationID_NoOrg(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		OrganizationID: 0,
	})

	orgID, err := GetOrganizationID(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if orgID != 0 {
		t.Fatalf("expected 0, got %d", orgID)
	}
}

func TestGetOrganizationID_NoClaims(t *testing.T) {
	ctx := context.Background()

	orgID, err := GetOrganizationID(ctx)
	if err != ErrClaimsNotFound {
		t.Fatalf("expected ErrClaimsNotFound, got %v", err)
	}
	if orgID != 0 {
		t.Fatalf("expected 0, got %d", orgID)
	}
}

// ---------- CheckCascadeStationAccess ----------

func ptr(v int64) *int64 { return &v }

func TestCheckCascadeStationAccess_ScFullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"sc"},
		OrganizationID: 1,
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	if err := CheckCascadeStationAccess(ctx, 999, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckCascadeStationAccess_RaisFullAccess(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"rais"},
		OrganizationID: 1,
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
		Roles:          []string{"cascade"},
		OrganizationID: cascadeOrgID,
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
		Roles:          []string{"cascade"},
		OrganizationID: cascadeOrgID,
	})
	checker := &mockCascadeChecker{parents: map[int64]*int64{}}

	// stationOrgID == claims.OrganizationID → allowed without parent lookup
	if err := CheckCascadeStationAccess(ctx, cascadeOrgID, checker); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckCascadeStationAccess_CascadeForeignStation(t *testing.T) {
	cascadeOrgID := int64(10)
	stationOrgID := int64(20)
	otherCascade := int64(99)

	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"cascade"},
		OrganizationID: cascadeOrgID,
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

func TestCheckCascadeStationAccess_DefaultFallback(t *testing.T) {
	// A role that is not sc/rais/cascade falls back to CheckOrgAccess
	ctx := contextWithClaims(&token.Claims{
		Roles:          []string{"reservoir"},
		OrganizationID: 5,
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
		Roles:          []string{"cascade"},
		OrganizationID: cascadeOrgID,
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
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationID: 1, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 10, Valid: true}}

	if err := CheckShutdownOwnership(ctx, 42, checker); err != nil {
		t.Fatalf("expected nil for cascade owner, got %v", err)
	}
}

func TestCheckShutdownOwnership_CascadeForeignForbidden(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationID: 1, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 11, Valid: true}}

	err := CheckShutdownOwnership(ctx, 42, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cascade non-owner, got %v", err)
	}
}

func TestCheckShutdownOwnership_CascadeNullOwnerForbidden(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationID: 1, Roles: []string{"cascade"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Valid: false}}

	err := CheckShutdownOwnership(ctx, 42, checker)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cascade on orphaned record, got %v", err)
	}
}

func TestCheckShutdownOwnership_NotFoundPropagates(t *testing.T) {
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationID: 1, Roles: []string{"cascade"}})
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
	ctx := contextWithClaims(&token.Claims{UserID: 10, OrganizationID: 5, Roles: []string{"reservoir"}})
	checker := &mockOwnerChecker{owner: sql.NullInt64{Int64: 99, Valid: true}}

	if err := CheckShutdownOwnership(ctx, 42, checker); err != nil {
		t.Fatalf("expected nil for non-cascade non-sc role, got %v", err)
	}
	if checker.calls != 0 {
		t.Errorf("repo must NOT be called for non-cascade non-sc role; got %d calls", checker.calls)
	}
}
