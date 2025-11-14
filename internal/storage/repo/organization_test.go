package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestOrganizationRepository_AddOrganization(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds organization with types", func(t *testing.T) {
		// Create organization types
		type1ID, _ := repo.AddOrganizationType(ctx, "Cascade", nil)
		type2ID, _ := repo.AddOrganizationType(ctx, "HPP", nil)

		orgID, err := repo.AddOrganization(ctx, "Test Cascade", nil, []int64{type1ID, type2ID})

		require.NoError(t, err)
		assert.Greater(t, orgID, int64(0))

		// Verify organization and its types
		orgs, err := repo.GetAllOrganizations(ctx, nil)
		require.NoError(t, err)

		found := false
		for _, org := range orgs {
			if org.ID == orgID {
				found = true
				assert.Equal(t, "Test Cascade", org.Name)
				assert.ElementsMatch(t, []string{"Cascade", "HPP"}, org.Types)
			}
		}
		assert.True(t, found, "organization not found in results")
	})

	t.Run("successfully adds organization without types", func(t *testing.T) {
		orgID, err := repo.AddOrganization(ctx, "Simple Org", nil, []int64{})

		require.NoError(t, err)
		assert.Greater(t, orgID, int64(0))
	})

	t.Run("successfully adds child organization", func(t *testing.T) {
		parentID, _ := repo.AddOrganization(ctx, "Parent Org", nil, []int64{})
		childID, err := repo.AddOrganization(ctx, "Child Org", &parentID, []int64{})

		require.NoError(t, err)
		assert.Greater(t, childID, int64(0))
	})

	t.Run("rolls back transaction on invalid type_id", func(t *testing.T) {
		// Try to add org with non-existent type
		_, err := repo.AddOrganization(ctx, "Bad Org", nil, []int64{99999})

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)

		// Verify organization was NOT created (rollback worked)
		orgs, _ := repo.GetAllOrganizations(ctx, nil)
		for _, org := range orgs {
			assert.NotEqual(t, "Bad Org", org.Name)
		}
	})
}

func TestOrganizationRepository_GetAllOrganizations(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("returns all organizations", func(t *testing.T) {
		// Create organizations
		org1ID, _ := repo.AddOrganization(ctx, "Org 1", nil, []int64{})
		org2ID, _ := repo.AddOrganization(ctx, "Org 2", nil, []int64{})

		orgs, err := repo.GetAllOrganizations(ctx, nil)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orgs), 2)

		// Check our orgs are included
		ids := make([]int64, len(orgs))
		for i, o := range orgs {
			ids[i] = o.ID
		}
		assert.Contains(t, ids, org1ID)
		assert.Contains(t, ids, org2ID)
	})

	t.Run("filters by organization type", func(t *testing.T) {
		typeID, _ := repo.AddOrganizationType(ctx, "FilterType", nil)
		orgWithType, _ := repo.AddOrganization(ctx, "Org With Type", nil, []int64{typeID})
		_, _ = repo.AddOrganization(ctx, "Org Without Type", nil, []int64{})

		orgs, err := repo.GetAllOrganizations(ctx, &typeID)

		require.NoError(t, err)
		// Should only include organization with the type
		found := false
		for _, org := range orgs {
			if org.ID == orgWithType {
				found = true
			}
		}
		assert.True(t, found)
	})
}

func TestOrganizationRepository_EditOrganization(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully updates organization name", func(t *testing.T) {
		orgID, _ := repo.AddOrganization(ctx, "Original Name", nil, []int64{})
		newName := "Updated Name"

		err := repo.EditOrganization(ctx, orgID, &newName, nil, nil)
		require.NoError(t, err)

		// Verify
		orgs, _ := repo.GetAllOrganizations(ctx, nil)
		for _, org := range orgs {
			if org.ID == orgID {
				assert.Equal(t, "Updated Name", org.Name)
			}
		}
	})

	t.Run("successfully updates organization types", func(t *testing.T) {
		type1ID, _ := repo.AddOrganizationType(ctx, "Type1", nil)
		type2ID, _ := repo.AddOrganizationType(ctx, "Type2", nil)

		orgID, _ := repo.AddOrganization(ctx, "Org", nil, []int64{type1ID})

		// Update to type2
		err := repo.EditOrganization(ctx, orgID, nil, nil, []int64{type2ID})
		require.NoError(t, err)

		// Verify
		orgs, _ := repo.GetAllOrganizations(ctx, nil)
		for _, org := range orgs {
			if org.ID == orgID {
				assert.Equal(t, []string{"Type2"}, org.Types)
			}
		}
	})

	t.Run("returns error for non-existent organization", func(t *testing.T) {
		newName := "Test"
		err := repo.EditOrganization(ctx, 99999, &newName, nil, nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestOrganizationRepository_DeleteOrganization(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes organization", func(t *testing.T) {
		orgID, _ := repo.AddOrganization(ctx, "To Delete", nil, []int64{})

		err := repo.DeleteOrganization(ctx, orgID)
		require.NoError(t, err)

		// Verify deletion
		orgs, _ := repo.GetAllOrganizations(ctx, nil)
		for _, org := range orgs {
			assert.NotEqual(t, orgID, org.ID)
		}
	})

	t.Run("returns error for non-existent organization", func(t *testing.T) {
		err := repo.DeleteOrganization(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
