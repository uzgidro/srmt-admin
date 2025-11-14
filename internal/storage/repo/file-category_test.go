package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestFileCategoryRepository_GetCategoryByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves category", func(t *testing.T) {
		// Create category
		categoryID, err := repo.AddCategory(ctx, nil, "test-cat", "Test Category", strPtr("Test description"))
		require.NoError(t, err)

		category, err := repo.GetCategoryByID(ctx, categoryID)

		require.NoError(t, err)
		assert.Equal(t, categoryID, category.ID)
		assert.Equal(t, "test-cat", category.Name)
		assert.Equal(t, "Test Category", category.DisplayName)
		assert.Equal(t, "Test description", *category.Description)
	})

	t.Run("returns error for non-existent category", func(t *testing.T) {
		_, err := repo.GetCategoryByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestFileCategoryRepository_GetAllCategories(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("returns all categories", func(t *testing.T) {
		// Create multiple categories
		cat1ID, _ := repo.AddCategory(ctx, nil, "cat1", "Category 1", nil)
		cat2ID, _ := repo.AddCategory(ctx, nil, "cat2", "Category 2", nil)

		categories, err := repo.GetAllCategories(ctx)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(categories), 2)

		// Check our categories are in the list
		ids := make([]int64, len(categories))
		for i, c := range categories {
			ids[i] = c.ID
		}
		assert.Contains(t, ids, cat1ID)
		assert.Contains(t, ids, cat2ID)
	})

	t.Run("returns empty array when no categories exist", func(t *testing.T) {
		// Start fresh
		testDB.TruncateTable(t, "categories")

		categories, err := repo.GetAllCategories(ctx)

		require.NoError(t, err)
		assert.Equal(t, 0, len(categories))
	})
}

func TestFileCategoryRepository_AddCategory(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds category with all fields", func(t *testing.T) {
		categoryID, err := repo.AddCategory(ctx, nil, "test", "Test Category", strPtr("Description"))

		require.NoError(t, err)
		assert.Greater(t, categoryID, int64(0))

		// Verify
		category, err := repo.GetCategoryByID(ctx, categoryID)
		require.NoError(t, err)
		assert.Equal(t, "test", category.Name)
		assert.Equal(t, "Test Category", category.DisplayName)
	})

	t.Run("successfully adds category with parent", func(t *testing.T) {
		parentID, _ := repo.AddCategory(ctx, nil, "parent", "Parent", nil)
		childID, err := repo.AddCategory(ctx, &parentID, "child", "Child", nil)

		require.NoError(t, err)
		assert.Greater(t, childID, int64(0))
	})

	t.Run("returns error on duplicate name", func(t *testing.T) {
		_, err := repo.AddCategory(ctx, nil, "unique", "Unique", nil)
		require.NoError(t, err)

		_, err = repo.AddCategory(ctx, nil, "unique", "Unique 2", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})
}
