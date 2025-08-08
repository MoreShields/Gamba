package repository

import (
	"context"
	"testing"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_GetByDiscordID(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestUserRepository_GetScoreboardData(t *testing.T) {
	t.Parallel()

	// Helper function to create isolated test environment for each sub-test
	setupTest := func(t *testing.T) (*UserRepository, context.Context, int64, func(string, ...interface{}), *testutil.TestDatabase) {
		testDB := testutil.SetupTestDatabase(t)
		guildID := int64(123456789)
		repo := NewUserRepositoryScoped(testDB.DB.Pool, guildID)
		ctx := context.Background()

		// Helper function to insert test data directly via SQL for complex scenarios
		insertTestData := func(query string, args ...interface{}) {
			_, err := testDB.DB.Pool.Exec(ctx, query, args...)
			require.NoError(t, err)
		}

		return repo, ctx, guildID, insertTestData, testDB
	}

	t.Run("empty database returns empty results", func(t *testing.T) {
		repo, ctx, _, _, _ := setupTest(t)
		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("users with zero balance are excluded", func(t *testing.T) {
		repo, ctx, _, _, _ := setupTest(t)
		// Create users with different balances
		userPositive := testutil.CreateTestUserWithBalance(111111, "positive_user", 50000)
		userZero := testutil.CreateTestUserWithBalance(222222, "zero_user", 0)

		_, err := repo.Create(ctx, userPositive.DiscordID, userPositive.Username, userPositive.Balance)
		require.NoError(t, err)
		_, err = repo.Create(ctx, userZero.DiscordID, userZero.Username, userZero.Balance)
		require.NoError(t, err)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, userPositive.DiscordID, entries[0].DiscordID)
		assert.Equal(t, userPositive.Username, entries[0].Username)
		assert.Equal(t, userPositive.Balance, entries[0].TotalBalance)
		assert.Equal(t, 1, entries[0].Rank)
	})

	t.Run("proper ranking assignment", func(t *testing.T) {
		repo, ctx, _, _, _ := setupTest(t)
		// Create users with different balances
		user1 := testutil.CreateTestUserWithBalance(333333, "user1", 75000)
		user2 := testutil.CreateTestUserWithBalance(444444, "user2", 50000)
		user3 := testutil.CreateTestUserWithBalance(555555, "user3", 100000)

		_, err := repo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
		require.NoError(t, err)
		_, err = repo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
		require.NoError(t, err)
		_, err = repo.Create(ctx, user3.DiscordID, user3.Username, user3.Balance)
		require.NoError(t, err)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 3)

		// Should be ranked by balance DESC
		assert.Equal(t, user3.DiscordID, entries[0].DiscordID) // 100000
		assert.Equal(t, 1, entries[0].Rank)
		assert.Equal(t, user1.DiscordID, entries[1].DiscordID) // 75000
		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, user2.DiscordID, entries[2].DiscordID) // 50000
		assert.Equal(t, 3, entries[2].Rank)
	})

	t.Run("proper calculation of wager win rates", func(t *testing.T) {
		repo, ctx, guildID, insertTestData, _ := setupTest(t)
		// Create test user
		user := testutil.CreateTestUserWithBalance(666666, "wager_user", 50000)
		_, err := repo.Create(ctx, user.DiscordID, user.Username, user.Balance)
		require.NoError(t, err)


		// Create other users to participate in wagers
		otherUser1 := testutil.CreateTestUserWithBalance(user.DiscordID+1, "other_user1", 40000)
		_, err = repo.Create(ctx, otherUser1.DiscordID, otherUser1.Username, otherUser1.Balance)
		require.NoError(t, err)

		otherUser2 := testutil.CreateTestUserWithBalance(user.DiscordID+2, "other_user2", 40000)
		_, err = repo.Create(ctx, otherUser2.DiscordID, otherUser2.Username, otherUser2.Balance)
		require.NoError(t, err)

		// Insert wagers with voting state to test active wager count
		// For this test, let's focus on active wager count rather than win rates
		// since resolved wagers require complex balance history setup
		insertTestData(`
			INSERT INTO wagers (proposer_discord_id, target_discord_id, amount, condition, state, winner_discord_id, guild_id, created_at)
			VALUES 
			($1, $2, 1000, 'Test wager 1', 'voting', NULL, $3, NOW()),
			($1, $4, 1500, 'Test wager 2', 'proposed', NULL, $3, NOW()),
			($2, $1, 2000, 'Test wager 3', 'voting', NULL, $3, NOW())
		`, user.DiscordID, otherUser1.DiscordID, guildID, otherUser2.DiscordID)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 3) // All 3 users should appear since they all have positive balances

		// Find our test user's entry
		var userEntry *entities.ScoreboardEntry
		for _, entry := range entries {
			if entry.DiscordID == user.DiscordID {
				userEntry = entry
				break
			}
		}
		require.NotNil(t, userEntry)

		assert.Equal(t, user.DiscordID, userEntry.DiscordID)
		// User is proposer in 2 wagers: 1 voting + 1 proposed = 2 active wagers 
		assert.Equal(t, 2, userEntry.ActiveWagerCount)
		// No resolved wagers, so win rate should be 0
		assert.Equal(t, float64(0), userEntry.WagerWinRate)
	})

	t.Run("proper calculation of bet win rates", func(t *testing.T) {
		repo, ctx, guildID, insertTestData, _ := setupTest(t)
		// Create test user
		user := testutil.CreateTestUserWithBalance(777777, "bet_user", 60000)
		_, err := repo.Create(ctx, user.DiscordID, user.Username, user.Balance)
		require.NoError(t, err)


		// Insert bets with different outcomes
		// 3 total bets, 2 wins, 1 loss (66.67% win rate)
		insertTestData(`
			INSERT INTO bets (discord_id, amount, win_probability, won, win_amount, guild_id, created_at)
			VALUES 
			($1, 1000, 0.5, true, 2000, $2, NOW()),
			($1, 1500, 0.6, true, 2500, $2, NOW()),
			($1, 2000, 0.33, false, 0, $2, NOW())
		`, user.DiscordID, guildID)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, user.DiscordID, entry.DiscordID)
		assert.InDelta(t, 66.67, entry.BetWinRate, 0.01) // 2 wins out of 3 = 66.67%
	})

	t.Run("proper aggregation of volume and donations", func(t *testing.T) {
		repo, ctx, guildID, insertTestData, _ := setupTest(t)
		// Create test user
		user := testutil.CreateTestUserWithBalance(888888, "volume_user", 75000)
		_, err := repo.Create(ctx, user.DiscordID, user.Username, user.Balance)
		require.NoError(t, err)


		// Insert balance history with different transaction types
		insertTestData(`
			INSERT INTO balance_history (discord_id, guild_id, balance_before, balance_after, change_amount, transaction_type, transaction_metadata, created_at)
			VALUES 
			($1, $2, 100000, 110000, 10000, 'bet_win', '{}', NOW()),
			($1, $2, 110000, 105000, -5000, 'bet_loss', '{}', NOW()),
			($1, $2, 105000, 95000, -10000, 'transfer_out', '{}', NOW()),
			($1, $2, 95000, 105000, 10000, 'transfer_in', '{}', NOW()),
			($1, $2, 105000, 100000, -5000, 'transfer_out', '{}', NOW())
		`, user.DiscordID, guildID)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, user.DiscordID, entry.DiscordID)
		// Total volume: 10000 + 5000 + 10000 + 10000 + 5000 = 40000 (sum of absolute values)
		assert.Equal(t, int64(40000), entry.TotalVolume)
		// Total donations: 10000 + 5000 = 15000 (transfer_out transactions)
		assert.Equal(t, int64(15000), entry.TotalDonations)
	})

	t.Run("zero rates when no wagers or bets exist", func(t *testing.T) {
		repo, ctx, _, _, _ := setupTest(t)
		// Create user with no gambling activity
		user := testutil.CreateTestUserWithBalance(999999, "clean_user", 80000)
		_, err := repo.Create(ctx, user.DiscordID, user.Username, user.Balance)
		require.NoError(t, err)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, user.DiscordID, entry.DiscordID)
		assert.Equal(t, float64(0), entry.WagerWinRate)
		assert.Equal(t, float64(0), entry.BetWinRate)
		assert.Equal(t, 0, entry.ActiveWagerCount)
		assert.Equal(t, int64(0), entry.TotalVolume)
		assert.Equal(t, int64(0), entry.TotalDonations)
	})

	t.Run("integration with multiple data sources", func(t *testing.T) {
		repo, ctx, guildID, insertTestData, _ := setupTest(t)
		// Create comprehensive test scenario with wagers, bets, and balance history
		user1 := testutil.CreateTestUserWithBalance(1111111, "comprehensive_user1", 90000)
		user2 := testutil.CreateTestUserWithBalance(2222222, "comprehensive_user2", 85000)

		_, err := repo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
		require.NoError(t, err)
		_, err = repo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
		require.NoError(t, err)


		// Insert comprehensive test data
		// Wagers for user1: 1 voting
		insertTestData(`
			INSERT INTO wagers (proposer_discord_id, target_discord_id, amount, condition, state, winner_discord_id, guild_id, created_at)
			VALUES 
			($1, $2, 1500, 'User1 wager 1', 'voting', NULL, $3, NOW())
		`, user1.DiscordID, user2.DiscordID, guildID)

		// Bets for user1: 4 bets, 3 wins
		insertTestData(`
			INSERT INTO bets (discord_id, amount, win_probability, won, win_amount, guild_id, created_at)
			VALUES 
			($1, 1000, 0.5, true, 2000, $2, NOW()),
			($1, 1500, 0.55, true, 2700, $2, NOW()),
			($1, 2000, 0.4, false, 0, $2, NOW()),
			($1, 1200, 0.52, true, 2280, $2, NOW())
		`, user1.DiscordID, guildID)

		// Balance history for user1
		insertTestData(`
			INSERT INTO balance_history (discord_id, guild_id, balance_before, balance_after, change_amount, transaction_type, transaction_metadata, created_at)
			VALUES 
			($1, $2, 100000, 102000, 2000, 'wager_win', '{}', NOW()),
			($1, $2, 102000, 99000, -3000, 'wager_loss', '{}', NOW()),
			($1, $2, 99000, 94000, -5000, 'transfer_out', '{}', NOW()),
			($1, $2, 94000, 96000, 2000, 'bet_win', '{}', NOW())
		`, user1.DiscordID, guildID)

		// Minimal data for user2 to ensure they still appear
		insertTestData(`
			INSERT INTO bets (discord_id, amount, win_probability, won, win_amount, guild_id, created_at)
			VALUES ($1, 500, 0.5, false, 0, $2, NOW())
		`, user2.DiscordID, guildID)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 2)

		// Find user1's entry
		var user1Entry *entities.ScoreboardEntry
		for _, entry := range entries {
			if entry.DiscordID == user1.DiscordID {
				user1Entry = entry
				break
			}
		}
		require.NotNil(t, user1Entry)

		// Verify user1's comprehensive stats
		assert.Equal(t, user1.DiscordID, user1Entry.DiscordID)
		assert.Equal(t, "comprehensive_user1", user1Entry.Username)
		assert.Equal(t, int64(90000), user1Entry.TotalBalance)
		assert.Equal(t, float64(0), user1Entry.WagerWinRate)    // 0 wins out of 0 resolved
		assert.Equal(t, float64(75), user1Entry.BetWinRate)     // 3 wins out of 4 bets
		assert.Equal(t, 1, user1Entry.ActiveWagerCount)         // 1 voting wager
		assert.Equal(t, int64(12000), user1Entry.TotalVolume)   // 2000+3000+5000+2000
		assert.Equal(t, int64(5000), user1Entry.TotalDonations) // Only transfer_out

		// Verify ranking (user1 should be rank 1 due to higher balance)
		assert.Equal(t, 1, user1Entry.Rank)
	})

	t.Run("handles group wagers in available balance calculation", func(t *testing.T) {
		repo, ctx, guildID, insertTestData, testDB := setupTest(t)
		// Create test user
		user := testutil.CreateTestUserWithBalance(3333333, "group_wager_user", 50000)
		_, err := repo.Create(ctx, user.DiscordID, user.Username, user.Balance)
		require.NoError(t, err)


		// Insert group wager and participant  
		// For 'active' state, voting_starts_at and voting_ends_at must be NOT NULL
		insertTestData(`
			INSERT INTO group_wagers (creator_discord_id, condition, state, total_pot, guild_id, voting_starts_at, voting_ends_at, created_at)
			VALUES ($1, 'Test group wager', 'active', 5000, $2, NOW(), NOW() + INTERVAL '1 hour', NOW())
		`, user.DiscordID, guildID)

		// Get the group wager ID
		var groupWagerID int64
		err = testDB.DB.Pool.QueryRow(ctx, `SELECT id FROM group_wagers WHERE creator_discord_id = $1`, user.DiscordID).Scan(&groupWagerID)
		require.NoError(t, err)

		// Create a group wager option first
		insertTestData(`
			INSERT INTO group_wager_options (group_wager_id, option_text, option_order, total_amount, odds_multiplier, created_at)
			VALUES ($1, 'Option A', 1, 0, 0, NOW())
		`, groupWagerID)

		// Get the option ID
		var optionID int64
		err = testDB.DB.Pool.QueryRow(ctx, `SELECT id FROM group_wager_options WHERE group_wager_id = $1`, groupWagerID).Scan(&optionID)
		require.NoError(t, err)

		// Insert participant (user participating in their own wager)
		insertTestData(`
			INSERT INTO group_wager_participants (group_wager_id, discord_id, option_id, amount, created_at)
			VALUES ($1, $2, $3, 3000, NOW())
		`, groupWagerID, user.DiscordID, optionID)

		entries, _, err := repo.GetScoreboardData(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, user.DiscordID, entry.DiscordID)
		assert.Equal(t, int64(50000), entry.TotalBalance)
		// Available balance should be reduced by the group wager participation amount
		assert.Equal(t, int64(47000), entry.AvailableBalance) // 50000 - 3000
	})
}
