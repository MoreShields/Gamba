package repository

import (
	"context"
	"testing"

	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_GetByDiscordID(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	t.Run("user not found", func(t *testing.T) {
		user, err := repo.GetByDiscordID(ctx, 999999)
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("user found", func(t *testing.T) {
		// Create a test user first
		testUser := testutil.CreateTestUser(123456, "testuser")
		createdUser, err := repo.Create(ctx, testUser.DiscordID, testUser.Username, testUser.Balance)
		require.NoError(t, err)

		// Now retrieve it
		user, err := repo.GetByDiscordID(ctx, 123456)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, testUser.DiscordID, user.DiscordID)
		assert.Equal(t, testUser.Username, user.Username)
		assert.Equal(t, testUser.Balance, user.Balance)
		assert.Equal(t, createdUser.CreatedAt, user.CreatedAt)
		assert.Equal(t, createdUser.UpdatedAt, user.UpdatedAt)
	})
}

func TestUserRepository_Create(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		testUser := testutil.CreateTestUser(123456, "testuser")

		user, err := repo.Create(ctx, testUser.DiscordID, testUser.Username, testUser.Balance)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, testUser.DiscordID, user.DiscordID)
		assert.Equal(t, testUser.Username, user.Username)
		assert.Equal(t, testUser.Balance, user.Balance)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
	})

	t.Run("duplicate discord ID", func(t *testing.T) {
		testUser := testutil.CreateTestUser(789012, "testuser2")

		// Create user first time
		_, err := repo.Create(ctx, testUser.DiscordID, testUser.Username, testUser.Balance)
		require.NoError(t, err)

		// Try to create again with same Discord ID
		_, err = repo.Create(ctx, testUser.DiscordID, "different_name", testUser.Balance)
		assert.Error(t, err)
	})

	t.Run("different users", func(t *testing.T) {
		user1 := testutil.CreateTestUser(111111, "user1")
		user2 := testutil.CreateTestUser(222222, "user2")

		createdUser1, err := repo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
		require.NoError(t, err)

		createdUser2, err := repo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
		require.NoError(t, err)

		assert.NotEqual(t, createdUser1.DiscordID, createdUser2.DiscordID)
		assert.NotEqual(t, createdUser1.Username, createdUser2.Username)
	})
}

func TestUserRepository_UpdateBalance(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		// Create a test user
		testUser := testutil.CreateTestUser(123456, "testuser")
		_, err := repo.Create(ctx, testUser.DiscordID, testUser.Username, testUser.Balance)
		require.NoError(t, err)

		// Update balance
		newBalance := int64(50000)
		err = repo.UpdateBalance(ctx, testUser.DiscordID, newBalance)
		require.NoError(t, err)

		// Verify the update
		user, err := repo.GetByDiscordID(ctx, testUser.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, newBalance, user.Balance)
	})

	t.Run("user not found", func(t *testing.T) {
		err := repo.UpdateBalance(ctx, 999999, 50000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("zero balance", func(t *testing.T) {
		// Create a test user
		testUser := testutil.CreateTestUser(345678, "testuser3")
		_, err := repo.Create(ctx, testUser.DiscordID, testUser.Username, testUser.Balance)
		require.NoError(t, err)

		// Update to zero balance
		err = repo.UpdateBalance(ctx, testUser.DiscordID, 0)
		require.NoError(t, err)

		// Verify the update
		user, err := repo.GetByDiscordID(ctx, testUser.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), user.Balance)
	})
}

func TestUserRepository_GetUsersWithPositiveBalance(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	t.Run("no users with positive balance", func(t *testing.T) {
		users, err := repo.GetUsersWithPositiveBalance(ctx)
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("mixed balances", func(t *testing.T) {
		// Create users with different balances
		user1 := testutil.CreateTestUserWithBalance(111111, "user1", 50000)
		user2 := testutil.CreateTestUserWithBalance(222222, "user2", 0)
		user3 := testutil.CreateTestUserWithBalance(333333, "user3", 75000)

		_, err := repo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
		require.NoError(t, err)
		_, err = repo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
		require.NoError(t, err)
		_, err = repo.Create(ctx, user3.DiscordID, user3.Username, user3.Balance)
		require.NoError(t, err)

		users, err := repo.GetUsersWithPositiveBalance(ctx)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// Should be sorted by balance DESC
		assert.Equal(t, user3.DiscordID, users[0].DiscordID) // 75000
		assert.Equal(t, user1.DiscordID, users[1].DiscordID) // 50000
	})
}

func TestUserRepository_GetAll(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	t.Run("no users", func(t *testing.T) {
		users, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("multiple users", func(t *testing.T) {
		// Create multiple users
		user1 := testutil.CreateTestUser(111111, "user1")
		user2 := testutil.CreateTestUser(222222, "user2")
		user3 := testutil.CreateTestUser(333333, "user3")

		createdUser1, err := repo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
		require.NoError(t, err)
		createdUser2, err := repo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
		require.NoError(t, err)
		createdUser3, err := repo.Create(ctx, user3.DiscordID, user3.Username, user3.Balance)
		require.NoError(t, err)

		users, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, users, 3)

		// Should be sorted by created_at DESC (newest first)
		assert.Equal(t, createdUser3.DiscordID, users[0].DiscordID)
		assert.Equal(t, createdUser2.DiscordID, users[1].DiscordID)
		assert.Equal(t, createdUser1.DiscordID, users[2].DiscordID)
	})
}
