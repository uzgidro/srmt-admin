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

func TestUserRepository_AddUser(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds user", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)
		contactID := repotest.CreateMinimalContact(t, repo, "New User")

		userID, err := repo.AddUser(ctx, "newuser", []byte("$2a$10$hashedpassword"), contactID)

		require.NoError(t, err)
		assert.Greater(t, userID, int64(0))

		// Verify user was created
		user, err := repo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "newuser", user.Login)
		assert.Equal(t, "New User", user.FIO)
		assert.Equal(t, contactID, user.ContactID)
		assert.True(t, user.IsActive) // Default is active
	})

	t.Run("returns error on duplicate login", func(t *testing.T) {
		contactID1 := repotest.CreateMinimalContact(t, repo, "User 1")
		contactID2 := repotest.CreateMinimalContact(t, repo, "User 2")

		_, err := repo.AddUser(ctx, "duplicateuser", []byte("pass1"), contactID1)
		require.NoError(t, err)

		_, err = repo.AddUser(ctx, "duplicateuser", []byte("pass2"), contactID2)
		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrDuplicate)
	})

	t.Run("returns error on invalid contact_id", func(t *testing.T) {
		_, err := repo.AddUser(ctx, "testuser", []byte("pass"), 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves user with relationships", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		user, err := repo.GetUserByID(ctx, fixtures.UserID)

		require.NoError(t, err)
		assert.Equal(t, fixtures.UserID, user.ID)
		assert.Equal(t, "testuser", user.Login)
		assert.Equal(t, "Test User", user.FIO)
		assert.NotNil(t, user.Organization)
		assert.Equal(t, "Test Organization", user.Organization.Name)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		_, err := repo.GetUserByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestUserRepository_EditUser(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully updates user login", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)
		newLogin := "updatedlogin"
		req := dto.EditUserRequest{Login: &newLogin}

		err := repo.EditUser(ctx, fixtures.UserID, nil, req)
		require.NoError(t, err)

		user, _ := repo.GetUserByID(ctx, fixtures.UserID)
		assert.Equal(t, "updatedlogin", user.Login)
	})

	t.Run("successfully changes active status", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)
		inactive := false
		req := dto.EditUserRequest{IsActive: &inactive}

		err := repo.EditUser(ctx, fixtures.UserID, nil, req)
		require.NoError(t, err)

		user, _ := repo.GetUserByID(ctx, fixtures.UserID)
		assert.False(t, user.IsActive)
	})
}

func TestUserRepository_AssignRevokeRole(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully assigns role to user", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		err := repo.AssignRole(ctx, fixtures.UserID, fixtures.RoleID)
		require.NoError(t, err)

		// Verify role is assigned
		user, _ := repo.GetUserByID(ctx, fixtures.UserID)
		assert.Contains(t, user.Roles, "TestRole")
	})

	t.Run("successfully revokes role from user", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		// Assign first
		_ = repo.AssignRole(ctx, fixtures.UserID, fixtures.RoleID)

		// Revoke
		err := repo.RevokeRole(ctx, fixtures.UserID, fixtures.RoleID)
		require.NoError(t, err)

		// Verify role is removed
		user, _ := repo.GetUserByID(ctx, fixtures.UserID)
		assert.NotContains(t, user.Roles, "TestRole")
	})
}
