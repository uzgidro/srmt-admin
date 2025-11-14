package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestContactRepository_AddContact(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds contact with all fields", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddContactRequest{
			FIO:             "John Doe",
			Email:           strPtr("john.doe@example.com"),
			Phone:           strPtr("+1234567890"),
			IPPhone:         strPtr("1234"),
			ExternalOrgName: strPtr("External Corp"),
			OrganizationID:  &fixtures.OrgID,
			DepartmentID:    &fixtures.DeptID,
			PositionID:      &fixtures.PosID,
		}

		contactID, err := repo.AddContact(ctx, req)

		require.NoError(t, err)
		assert.Greater(t, contactID, int64(0))

		// Verify contact was created
		contact, err := repo.GetContactByID(ctx, contactID)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", contact.FIO)
		assert.Equal(t, "john.doe@example.com", *contact.Email)
		assert.Equal(t, "+1234567890", *contact.Phone)
	})

	t.Run("successfully adds minimal contact", func(t *testing.T) {
		req := dto.AddContactRequest{
			FIO: "Jane Smith",
		}

		contactID, err := repo.AddContact(ctx, req)

		require.NoError(t, err)
		assert.Greater(t, contactID, int64(0))

		// Verify
		contact, err := repo.GetContactByID(ctx, contactID)
		require.NoError(t, err)
		assert.Equal(t, "Jane Smith", contact.FIO)
		assert.Nil(t, contact.Email)
		assert.Nil(t, contact.Organization)
	})

	t.Run("returns error on duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"
		req1 := dto.AddContactRequest{FIO: "User 1", Email: &email}
		req2 := dto.AddContactRequest{FIO: "User 2", Email: &email}

		_, err := repo.AddContact(ctx, req1)
		require.NoError(t, err)

		_, err = repo.AddContact(ctx, req2)
		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})

	t.Run("returns error on invalid organization_id", func(t *testing.T) {
		invalidOrgID := int64(99999)
		req := dto.AddContactRequest{
			FIO:            "Test User",
			OrganizationID: &invalidOrgID,
		}

		_, err := repo.AddContact(ctx, req)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})
}

func TestContactRepository_GetContactByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves contact with all relationships", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		contact, err := repo.GetContactByID(ctx, fixtures.ContactID)

		require.NoError(t, err)
		assert.Equal(t, fixtures.ContactID, contact.ID)
		assert.Equal(t, "Test User", contact.FIO)
		assert.NotNil(t, contact.Organization)
		assert.Equal(t, "Test Organization", contact.Organization.Name)
		assert.NotNil(t, contact.Department)
		assert.Equal(t, "Test Department", contact.Department.Name)
		assert.NotNil(t, contact.Position)
		assert.Equal(t, "Test Position", contact.Position.Name)
	})

	t.Run("returns error for non-existent contact", func(t *testing.T) {
		_, err := repo.GetContactByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestContactRepository_GetAllContacts(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	fixtures := repotest.LoadFixtures(t, repo)

	// Create additional contacts
	org2ID, _ := repo.AddOrganization(ctx, "Org 2", nil, []int64{})
	dept2ID, _ := repo.AddDepartment(ctx, "Dept 2", nil, org2ID)

	contact2ID, _ := repo.AddContact(ctx, dto.AddContactRequest{FIO: "Contact 2", OrganizationID: &org2ID})
	contact3ID, _ := repo.AddContact(ctx, dto.AddContactRequest{FIO: "Contact 3", DepartmentID: &dept2ID})

	t.Run("returns all contacts with no filters", func(t *testing.T) {
		contacts, err := repo.GetAllContacts(ctx, dto.GetAllContactsFilters{})

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(contacts), 3)

		// Find our contacts
		ids := make([]int64, len(contacts))
		for i, c := range contacts {
			ids[i] = c.ID
		}
		assert.Contains(t, ids, fixtures.ContactID)
		assert.Contains(t, ids, contact2ID)
		assert.Contains(t, ids, contact3ID)
	})

	t.Run("filters by organization_id", func(t *testing.T) {
		filters := dto.GetAllContactsFilters{OrganizationID: &org2ID}
		contacts, err := repo.GetAllContacts(ctx, filters)

		require.NoError(t, err)
		assert.Equal(t, 1, len(contacts))
		assert.Equal(t, contact2ID, contacts[0].ID)
		assert.Equal(t, "Contact 2", contacts[0].FIO)
	})

	t.Run("filters by department_id", func(t *testing.T) {
		filters := dto.GetAllContactsFilters{DepartmentID: &dept2ID}
		contacts, err := repo.GetAllContacts(ctx, filters)

		require.NoError(t, err)
		assert.Equal(t, 1, len(contacts))
		assert.Equal(t, contact3ID, contacts[0].ID)
	})

	t.Run("returns empty array when no matches", func(t *testing.T) {
		nonExistentID := int64(99999)
		filters := dto.GetAllContactsFilters{OrganizationID: &nonExistentID}
		contacts, err := repo.GetAllContacts(ctx, filters)

		require.NoError(t, err)
		assert.Equal(t, 0, len(contacts))
	})
}

func TestContactRepository_EditContact(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully updates FIO", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)
		newFIO := "Updated Name"
		req := dto.EditContactRequest{FIO: &newFIO}

		err := repo.EditContact(ctx, fixtures.ContactID, req)
		require.NoError(t, err)

		// Verify
		contact, err := repo.GetContactByID(ctx, fixtures.ContactID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", contact.FIO)
	})

	t.Run("successfully updates email", func(t *testing.T) {
		contactID := repotest.CreateMinimalContact(t, repo, "Test Contact")
		newEmail := "newemail@example.com"
		req := dto.EditContactRequest{Email: &newEmail}

		err := repo.EditContact(ctx, contactID, req)
		require.NoError(t, err)

		contact, err := repo.GetContactByID(ctx, contactID)
		require.NoError(t, err)
		assert.Equal(t, "newemail@example.com", *contact.Email)
	})

	t.Run("successfully updates organization", func(t *testing.T) {
		contactID := repotest.CreateMinimalContact(t, repo, "Test Contact")
		newOrgID, _ := repo.AddOrganization(ctx, "New Org", nil, []int64{})
		req := dto.EditContactRequest{OrganizationID: &newOrgID}

		err := repo.EditContact(ctx, contactID, req)
		require.NoError(t, err)

		contact, err := repo.GetContactByID(ctx, contactID)
		require.NoError(t, err)
		assert.NotNil(t, contact.Organization)
		assert.Equal(t, "New Org", contact.Organization.Name)
	})

	t.Run("returns error for non-existent contact", func(t *testing.T) {
		newFIO := "Test"
		req := dto.EditContactRequest{FIO: &newFIO}

		err := repo.EditContact(ctx, 99999, req)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("returns error on invalid organization_id", func(t *testing.T) {
		contactID := repotest.CreateMinimalContact(t, repo, "Test Contact")
		invalidOrgID := int64(99999)
		req := dto.EditContactRequest{OrganizationID: &invalidOrgID}

		err := repo.EditContact(ctx, contactID, req)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})

	t.Run("handles no updates gracefully", func(t *testing.T) {
		contactID := repotest.CreateMinimalContact(t, repo, "Test Contact")
		req := dto.EditContactRequest{} // No fields to update

		err := repo.EditContact(ctx, contactID, req)

		require.NoError(t, err)
	})
}

func TestContactRepository_DeleteContact(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes contact", func(t *testing.T) {
		contactID := repotest.CreateMinimalContact(t, repo, "To Delete")

		err := repo.DeleteContact(ctx, contactID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetContactByID(ctx, contactID)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("returns error for non-existent contact", func(t *testing.T) {
		err := repo.DeleteContact(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

// Helper functions

func strPtr(s string) *string {
	return &s
}
