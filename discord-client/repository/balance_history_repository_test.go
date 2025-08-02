package repository

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBalanceHistoryRepository_Record(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewBalanceHistoryRepository(testDB.DB)
	userRepo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	// Create a test user first
	testUser := testutil.CreateTestUser(123456, "testuser")
	_, err := userRepo.Create(ctx, testUser.DiscordID, testUser.Username, testUser.Balance)
	require.NoError(t, err)

	t.Run("successful record creation", func(t *testing.T) {
		history := testutil.CreateTestBalanceHistory(testUser.DiscordID, entities.TransactionTypeBetLoss)

		err := repo.Record(ctx, history)
		require.NoError(t, err)
		assert.NotZero(t, history.ID)
		assert.False(t, history.CreatedAt.IsZero())
	})

	t.Run("record with metadata", func(t *testing.T) {
		history := testutil.CreateTestBalanceHistoryWithAmounts(
			testUser.DiscordID, 100000, 150000, 50000, entities.TransactionTypeBetWin)
		history.TransactionMetadata = map[string]interface{}{
			"bet_amount":     10000,
			"bet_odds":       0.5,
			"bet_id":         "bet_123",
			"win_multiplier": 5.0,
		}

		err := repo.Record(ctx, history)
		require.NoError(t, err)
		assert.NotZero(t, history.ID)
		assert.NotNil(t, history.TransactionMetadata)
	})

	t.Run("record with nil metadata", func(t *testing.T) {
		history := testutil.CreateTestBalanceHistory(testUser.DiscordID, entities.TransactionTypeBetLoss)
		history.TransactionMetadata = nil

		err := repo.Record(ctx, history)
		require.NoError(t, err)
		assert.NotZero(t, history.ID)
	})
}

func TestBalanceHistoryRepository_GetByUser(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewBalanceHistoryRepository(testDB.DB)
	userRepo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	// Create test users
	user1 := testutil.CreateTestUser(123456, "user1")
	user2 := testutil.CreateTestUser(789012, "user2")
	_, err := userRepo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
	require.NoError(t, err)
	_, err = userRepo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
	require.NoError(t, err)

	t.Run("no history for user", func(t *testing.T) {
		histories, err := repo.GetByUser(ctx, user1.DiscordID, 10)
		require.NoError(t, err)
		assert.Empty(t, histories)
	})

	t.Run("multiple history entries", func(t *testing.T) {
		// Create multiple history entries for user1
		for i := 0; i < 5; i++ {
			history := testutil.CreateTestBalanceHistoryWithAmounts(
				user1.DiscordID,
				int64(100000-i*10000),
				int64(90000-i*10000),
				-10000,
				entities.TransactionTypeBetLoss,
			)
			err := repo.Record(ctx, history)
			require.NoError(t, err)

			// Small delay to ensure different timestamps
			time.Sleep(time.Millisecond)
		}

		// Create one entry for user2 to ensure isolation
		history2 := testutil.CreateTestBalanceHistory(user2.DiscordID, entities.TransactionTypeBetWin)
		err := repo.Record(ctx, history2)
		require.NoError(t, err)

		// Get history for user1
		histories, err := repo.GetByUser(ctx, user1.DiscordID, 10)
		require.NoError(t, err)
		assert.Len(t, histories, 5)

		// Should be ordered by created_at DESC (newest first)
		for i := 1; i < len(histories); i++ {
			assert.True(t, histories[i-1].CreatedAt.After(histories[i].CreatedAt) ||
				histories[i-1].CreatedAt.Equal(histories[i].CreatedAt))
		}
	})

	t.Run("limit results", func(t *testing.T) {
		// Clear existing data by creating a new user
		user3 := testutil.CreateTestUser(555555, "user3")
		_, err := userRepo.Create(ctx, user3.DiscordID, user3.Username, user3.Balance)
		require.NoError(t, err)

		// Create more entries than limit
		for i := 0; i < 10; i++ {
			history := testutil.CreateTestBalanceHistory(user3.DiscordID, entities.TransactionTypeTransferIn)
			err := repo.Record(ctx, history)
			require.NoError(t, err)
			time.Sleep(time.Millisecond)
		}

		// Get with limit
		histories, err := repo.GetByUser(ctx, user3.DiscordID, 3)
		require.NoError(t, err)
		assert.Len(t, histories, 3)
	})

	t.Run("metadata preservation", func(t *testing.T) {
		user4 := testutil.CreateTestUser(666666, "user4")
		_, err := userRepo.Create(ctx, user4.DiscordID, user4.Username, user4.Balance)
		require.NoError(t, err)

		originalMetadata := map[string]interface{}{
			"bet_id":     "test_bet_123",
			"bet_amount": 5000,
			"bet_odds":   0.75,
			"nested_data": map[string]interface{}{
				"player_count": 3,
				"game_type":    "roulette",
			},
		}

		history := testutil.CreateTestBalanceHistory(user4.DiscordID, entities.TransactionTypeBetWin)
		history.TransactionMetadata = originalMetadata
		err = repo.Record(ctx, history)
		require.NoError(t, err)

		// Retrieve and verify metadata
		histories, err := repo.GetByUser(ctx, user4.DiscordID, 1)
		require.NoError(t, err)
		require.Len(t, histories, 1)

		retrieved := histories[0]
		assert.Equal(t, "test_bet_123", retrieved.TransactionMetadata["bet_id"])
		assert.Equal(t, float64(5000), retrieved.TransactionMetadata["bet_amount"])
		assert.Equal(t, 0.75, retrieved.TransactionMetadata["bet_odds"])

		nestedData := retrieved.TransactionMetadata["nested_data"].(map[string]interface{})
		assert.Equal(t, float64(3), nestedData["player_count"])
		assert.Equal(t, "roulette", nestedData["game_type"])
	})
}

func TestBalanceHistoryRepository_GetByDateRange(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewBalanceHistoryRepository(testDB.DB)
	userRepo := NewUserRepository(testDB.DB)
	ctx := context.Background()

	// Create test user
	user := testutil.CreateTestUser(123456, "testuser")
	_, err := userRepo.Create(ctx, user.DiscordID, user.Username, user.Balance)
	require.NoError(t, err)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	t.Run("no entries in range", func(t *testing.T) {
		histories, err := repo.GetByDateRange(ctx, user.DiscordID, yesterday, now)
		require.NoError(t, err)
		assert.Empty(t, histories)
	})

	t.Run("entries within range", func(t *testing.T) {
		// Create entries at different times
		history1 := testutil.CreateTestBalanceHistory(user.DiscordID, entities.TransactionTypeBetLoss)
		err := repo.Record(ctx, history1)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		history2 := testutil.CreateTestBalanceHistory(user.DiscordID, entities.TransactionTypeBetWin)
		err = repo.Record(ctx, history2)
		require.NoError(t, err)

		// Get entries from yesterday to tomorrow (should include both)
		histories, err := repo.GetByDateRange(ctx, user.DiscordID, yesterday, tomorrow)
		require.NoError(t, err)
		assert.Len(t, histories, 2)

		// Should be ordered by created_at DESC
		assert.True(t, histories[0].CreatedAt.After(histories[1].CreatedAt) ||
			histories[0].CreatedAt.Equal(histories[1].CreatedAt))
	})

	t.Run("entries outside range", func(t *testing.T) {
		user2 := testutil.CreateTestUser(789012, "user2")
		_, err := userRepo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
		require.NoError(t, err)

		// Create an entry
		history := testutil.CreateTestBalanceHistory(user2.DiscordID, entities.TransactionTypeBetWin)
		err = repo.Record(ctx, history)
		require.NoError(t, err)

		// Query for a range that doesn't include the entry (far in the past)
		farPast := now.Add(-72 * time.Hour)
		pastRange := now.Add(-48 * time.Hour)

		histories, err := repo.GetByDateRange(ctx, user2.DiscordID, farPast, pastRange)
		require.NoError(t, err)
		assert.Empty(t, histories)
	})

	t.Run("exact boundary conditions", func(t *testing.T) {
		user3 := testutil.CreateTestUser(999999, "user3")
		_, err := userRepo.Create(ctx, user3.DiscordID, user3.Username, user3.Balance)
		require.NoError(t, err)

		// Create an entry with buffer time
		beforeCreate := time.Now().Add(-time.Second)
		history := testutil.CreateTestBalanceHistory(user3.DiscordID, entities.TransactionTypeTransferOut)
		err = repo.Record(ctx, history)
		require.NoError(t, err)
		afterCreate := time.Now().Add(time.Second)

		// Query should include the entry
		histories, err := repo.GetByDateRange(ctx, user3.DiscordID, beforeCreate, afterCreate)
		require.NoError(t, err)
		assert.Len(t, histories, 1)
	})
}
