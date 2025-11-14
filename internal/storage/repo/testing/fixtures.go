package testing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/storage/repo"
)

// FixtureData holds IDs of commonly created test entities
// Use this to avoid recreating the same test data across multiple tests
type FixtureData struct {
	// Organization
	OrgID     int64
	OrgTypeID int64

	// Department and Position
	DeptID int64
	PosID  int64

	// Contact
	ContactID int64

	// User and Role
	UserID int64
	RoleID int64

	// File categories
	CategoryID       int64
	EventsCategoryID int64

	// Event types and statuses
	EventTypeID   int
	EventStatusID int
}

// LoadFixtures creates a complete set of standard test data
// This includes organization, department, position, contact, user, role, and file categories
func LoadFixtures(t *testing.T, r *repo.Repo) *FixtureData {
	t.Helper()

	ctx := context.Background()
	fixtures := &FixtureData{}

	// Create organization type
	orgTypeID, err := r.AddOrganizationType(ctx, "Test Organization Type", strPtr("Test type for fixtures"))
	require.NoError(t, err, "failed to create organization type")
	fixtures.OrgTypeID = orgTypeID

	// Create organization
	orgID, err := r.AddOrganization(ctx, "Test Organization", nil, []int64{orgTypeID})
	require.NoError(t, err, "failed to create organization")
	fixtures.OrgID = orgID

	// Create department
	deptID, err := r.AddDepartment(ctx, "Test Department", strPtr("Test department for fixtures"), fixtures.OrgID)
	require.NoError(t, err, "failed to create department")
	fixtures.DeptID = deptID

	// Create position
	posID, err := r.AddPosition(ctx, "Test Position", strPtr("Test position for fixtures"))
	require.NoError(t, err, "failed to create position")
	fixtures.PosID = posID

	// Create contact
	contactID, err := r.AddContact(ctx, dto.AddContactRequest{
		FIO:            "Test User",
		Email:          strPtr("testuser@example.com"),
		Phone:          strPtr("+1234567890"),
		OrganizationID: &orgID,
		DepartmentID:   &deptID,
		PositionID:     &posID,
	})
	require.NoError(t, err, "failed to create contact")
	fixtures.ContactID = contactID

	// Create user
	userID, err := r.AddUser(ctx, "testuser", []byte("$2a$10$hashedpassword"), contactID)
	require.NoError(t, err, "failed to create user")
	fixtures.UserID = userID

	// Create role
	roleID, err := r.AddRole(ctx, "TestRole", "Test role for fixtures")
	require.NoError(t, err, "failed to create role")
	fixtures.RoleID = roleID

	// Create file category
	testCat := category.Model{
		Name:        "test-category",
		DisplayName: "Test Category",
		Description: "Test category for fixtures",
	}
	categoryID, err := r.AddCategory(ctx, testCat)
	require.NoError(t, err, "failed to create file category")
	fixtures.CategoryID = categoryID

	// Create events category
	eventsCat := category.Model{
		Name:        "events",
		DisplayName: "Events",
		Description: "Category for event files",
	}
	eventsCategoryID, err := r.AddCategory(ctx, eventsCat)
	require.NoError(t, err, "failed to create events category")
	fixtures.EventsCategoryID = eventsCategoryID

	// Get default event type and status (created by migrations)
	// Meeting type (ID 1) and Active status (ID 3)
	fixtures.EventTypeID = 1   // Meeting
	fixtures.EventStatusID = 3 // Active

	return fixtures
}

// CreateMinimalContact creates a basic contact without dependencies
func CreateMinimalContact(t *testing.T, r *repo.Repo, fio string) int64 {
	t.Helper()

	contactID, err := r.AddContact(context.Background(), dto.AddContactRequest{
		FIO: fio,
	})
	require.NoError(t, err, "failed to create minimal contact")

	return contactID
}

// CreateContactWithOrg creates a contact linked to a specific organization
func CreateContactWithOrg(t *testing.T, r *repo.Repo, fio string, orgID int64) int64 {
	t.Helper()

	contactID, err := r.AddContact(context.Background(), dto.AddContactRequest{
		FIO:            fio,
		OrganizationID: &orgID,
	})
	require.NoError(t, err, "failed to create contact with organization")

	return contactID
}

// Helper functions for creating pointers to primitive types

func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
