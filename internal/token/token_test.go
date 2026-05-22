package token

import (
	"testing"
	"time"

	"srmt-admin/internal/lib/model/user"
)

// Access token must carry the user's full organization list under "org_ids".
// Regression guard for the single-org -> multi-org claim migration.
func TestCreateAccessToken_CarriesOrganizationIDs(t *testing.T) {
	svc, err := New("test-secret", time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	u := &user.Model{
		ID:              42,
		ContactID:       7,
		Name:            "Дежурный",
		Roles:           []string{"cascade", "reservoir_flood"},
		OrganizationIDs: []int64{5, 10},
	}

	pair, err := svc.Create(u)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	claims, err := svc.Verify(pair.AccessToken)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if len(claims.OrganizationIDs) != 2 ||
		claims.OrganizationIDs[0] != 5 || claims.OrganizationIDs[1] != 10 {
		t.Errorf("OrganizationIDs: want [5 10], got %v", claims.OrganizationIDs)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID: want 42, got %d", claims.UserID)
	}
}

// A user with no organizations must still produce a valid token — claims
// carry an empty (non-nil) slice, never panicking downstream membership checks.
func TestCreateAccessToken_NoOrganizations(t *testing.T) {
	svc, err := New("test-secret", time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	u := &user.Model{ID: 1, ContactID: 1, Name: "Sysadmin", Roles: []string{"sc"}}

	pair, err := svc.Create(u)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	claims, err := svc.Verify(pair.AccessToken)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(claims.OrganizationIDs) != 0 {
		t.Errorf("OrganizationIDs: want empty, got %v", claims.OrganizationIDs)
	}
}
