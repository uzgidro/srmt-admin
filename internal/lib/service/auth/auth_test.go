package auth

import (
	"context"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/token"
)

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
