package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestPositionRepository_AddPosition(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds position", func(t *testing.T) {
		posID, err := repo.AddPosition(ctx, "Manager", strPtr("Management position"))

		require.NoError(t, err)
		assert.Greater(t, posID, int64(0))
	})

	t.Run("returns error on duplicate name", func(t *testing.T) {
		_, _ = repo.AddPosition(ctx, "Unique", nil)
		_, err := repo.AddPosition(ctx, "Unique", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})
}

func TestPositionRepository_GetPositionByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves position", func(t *testing.T) {
		posID, _ := repo.AddPosition(ctx, "Developer", nil)

		pos, err := repo.GetPositionByID(ctx, posID)

		require.NoError(t, err)
		assert.Equal(t, "Developer", pos.Name)
	})

	t.Run("returns error for non-existent position", func(t *testing.T) {
		_, err := repo.GetPositionByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestPositionRepository_DeletePosition(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes position", func(t *testing.T) {
		posID, _ := repo.AddPosition(ctx, "To Delete", nil)

		err := repo.DeletePosition(ctx, posID)
		require.NoError(t, err)

		_, err = repo.GetPositionByID(ctx, posID)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
