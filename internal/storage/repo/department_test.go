package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestDepartmentRepository_AddDepartment(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds department", func(t *testing.T) {
		orgID, _ := repo.AddOrganization(ctx, "Test Org", nil, []int64{})
		deptID, err := repo.AddDepartment(ctx, "Engineering", strPtr("Engineering department"), orgID)

		require.NoError(t, err)
		assert.Greater(t, deptID, int64(0))
	})

	t.Run("returns error on duplicate name", func(t *testing.T) {
		orgID, _ := repo.AddOrganization(ctx, "Test Org", nil, []int64{})
		_, _ = repo.AddDepartment(ctx, "Unique", nil, orgID)
		_, err := repo.AddDepartment(ctx, "Unique", nil, orgID)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})
}

func TestDepartmentRepository_GetDepartmentByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves department", func(t *testing.T) {
		orgID, _ := repo.AddOrganization(ctx, "Test Org", nil, []int64{})
		deptID, _ := repo.AddDepartment(ctx, "IT", nil, orgID)

		dept, err := repo.GetDepartmentByID(ctx, deptID)

		require.NoError(t, err)
		assert.Equal(t, "IT", dept.Name)
	})

	t.Run("returns error for non-existent department", func(t *testing.T) {
		_, err := repo.GetDepartmentByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestDepartmentRepository_DeleteDepartment(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes department", func(t *testing.T) {
		orgID, _ := repo.AddOrganization(ctx, "Test Org", nil, []int64{})
		deptID, _ := repo.AddDepartment(ctx, "To Delete", nil, orgID)

		err := repo.DeleteDepartment(ctx, deptID)
		require.NoError(t, err)

		_, err = repo.GetDepartmentByID(ctx, deptID)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
