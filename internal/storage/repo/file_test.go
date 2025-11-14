package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestFileRepository_AddFile(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds file with all fields", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		fileModel := file.Model{
			FileName:   "test.pdf",
			ObjectKey:  "category/2025/01/01/uuid-123.pdf",
			CategoryID: fixtures.CategoryID,
			MimeType:   "application/pdf",
			SizeBytes:  1024,
			CreatedAt:  time.Now(),
		}

		fileID, err := repo.AddFile(ctx, fileModel)

		require.NoError(t, err)
		assert.Greater(t, fileID, int64(0))

		// Verify file was created
		retrievedFile, err := repo.GetFileByID(ctx, fileID)
		require.NoError(t, err)
		assert.Equal(t, "test.pdf", retrievedFile.FileName)
		assert.Equal(t, "application/pdf", retrievedFile.MimeType)
		assert.Equal(t, int64(1024), retrievedFile.SizeBytes)
	})

	t.Run("returns error on invalid category_id", func(t *testing.T) {
		fileModel := file.Model{
			FileName:   "test.pdf",
			ObjectKey:  "category/unique-key.pdf",
			CategoryID: 99999, // Non-existent
			CreatedAt:  time.Now(),
		}

		_, err := repo.AddFile(ctx, fileModel)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})

	t.Run("returns error on duplicate object_key", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		file1 := file.Model{
			FileName:   "file1.pdf",
			ObjectKey:  "unique-key.pdf",
			CategoryID: fixtures.CategoryID,
			CreatedAt:  time.Now(),
		}
		file2 := file.Model{
			FileName:   "file2.pdf",
			ObjectKey:  "unique-key.pdf", // Duplicate
			CategoryID: fixtures.CategoryID,
			CreatedAt:  time.Now(),
		}

		_, err := repo.AddFile(ctx, file1)
		require.NoError(t, err)

		_, err = repo.AddFile(ctx, file2)
		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})
}

func TestFileRepository_GetFileByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves file", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		fileModel := file.Model{
			FileName:   "document.pdf",
			ObjectKey:  "docs/document.pdf",
			CategoryID: fixtures.CategoryID,
			MimeType:   "application/pdf",
			SizeBytes:  2048,
			CreatedAt:  time.Now(),
		}
		fileID, _ := repo.AddFile(ctx, fileModel)

		retrievedFile, err := repo.GetFileByID(ctx, fileID)

		require.NoError(t, err)
		assert.Equal(t, fileID, retrievedFile.ID)
		assert.Equal(t, "document.pdf", retrievedFile.FileName)
		assert.Equal(t, "docs/document.pdf", retrievedFile.ObjectKey)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := repo.GetFileByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestFileRepository_DeleteFile(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes file", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		fileModel := file.Model{
			FileName:   "to-delete.pdf",
			ObjectKey:  "trash/to-delete.pdf",
			CategoryID: fixtures.CategoryID,
			CreatedAt:  time.Now(),
		}
		fileID, _ := repo.AddFile(ctx, fileModel)

		err := repo.DeleteFile(ctx, fileID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetFileByID(ctx, fileID)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		err := repo.DeleteFile(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestFileRepository_GetLatestFilesByCategory(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	fixtures := repotest.LoadFixtures(t, repo)

	t.Run("returns latest files in descending order", func(t *testing.T) {
		// Create files with different timestamps
		now := time.Now()
		file1Model := file.Model{
			FileName:   "old.pdf",
			ObjectKey:  "files/old.pdf",
			CategoryID: fixtures.CategoryID,
			CreatedAt:  now.Add(-2 * time.Hour),
		}
		file2Model := file.Model{
			FileName:   "new.pdf",
			ObjectKey:  "files/new.pdf",
			CategoryID: fixtures.CategoryID,
			CreatedAt:  now,
		}

		repo.AddFile(ctx, file1Model)
		file2ID, _ := repo.AddFile(ctx, file2Model)

		files, err := repo.GetLatestFilesByCategory(ctx, fixtures.CategoryID, 10)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 2)
		// Most recent first
		assert.Equal(t, file2ID, files[0].ID)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		files, err := repo.GetLatestFilesByCategory(ctx, fixtures.CategoryID, 1)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(files), 1)
	})
}
