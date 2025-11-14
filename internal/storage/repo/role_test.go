package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestRoleRepository_AddRole(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds role", func(t *testing.T) {
		roleID, err := repo.AddRole(ctx, "admin", "Administrator role")

		require.NoError(t, err)
		assert.Greater(t, roleID, int64(0))
	})

	t.Run("returns error on duplicate name", func(t *testing.T) {
		_, _ = repo.AddRole(ctx, "unique-role", "")
		_, err := repo.AddRole(ctx, "unique-role", "")

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})
}

func TestRoleRepository_GetRoleByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves role", func(t *testing.T) {
		roleID, _ := repo.AddRole(ctx, "editor", "")

		role, err := repo.GetRoleByID(ctx, roleID)

		require.NoError(t, err)
		assert.Equal(t, "editor", role.Name)
	})

	t.Run("returns error for non-existent role", func(t *testing.T) {
		_, err := repo.GetRoleByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestRoleRepository_DeleteRole(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes role", func(t *testing.T) {
		roleID, _ := repo.AddRole(ctx, "to-delete", "")

		err := repo.DeleteRole(ctx, roleID)
		require.NoError(t, err)

		_, err = repo.GetRoleByID(ctx, roleID)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
