package repository

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/models"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerRepository_GetStats(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	groupWagerRepo := NewGroupWagerRepository(testDB.DB)
	userRepo := NewUserRepository(testDB.DB)
	balanceHistoryRepo := NewBalanceHistoryRepository(testDB.DB)
	ctx := context.Background()

	// Create test users
	user1 := testutil.CreateTestUser(111111, "user1")
	user2 := testutil.CreateTestUser(222222, "user2")
	user3 := testutil.CreateTestUser(333333, "user3")

	_, err := userRepo.Create(ctx, user1.DiscordID, user1.Username, user1.Balance)
	require.NoError(t, err)
	_, err = userRepo.Create(ctx, user2.DiscordID, user2.Username, user2.Balance)
	require.NoError(t, err)
	_, err = userRepo.Create(ctx, user3.DiscordID, user3.Username, user3.Balance)
	require.NoError(t, err)

	t.Run("no group wagers - returns zero stats", func(t *testing.T) {
		stats, err := groupWagerRepo.GetStats(ctx, user1.DiscordID)
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, 0, stats.TotalGroupWagers)
		assert.Equal(t, 0, stats.TotalProposed)
		assert.Equal(t, 0, stats.TotalWon)
		assert.Equal(t, int64(0), stats.TotalWonAmount)
	})

	t.Run("user as creator only", func(t *testing.T) {
		// Create a group wager where user1 is creator but doesn't participate
		wager1 := testutil.CreateTestGroupWager(user1.DiscordID, "Test wager 1")
		option1 := testutil.CreateTestGroupWagerOption(0, "Option A", 0)
		option2 := testutil.CreateTestGroupWagerOption(0, "Option B", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, wager1, []*models.GroupWagerOption{option1, option2})
		require.NoError(t, err)

		// Add participants (not including user1)
		participant1 := testutil.CreateTestGroupWagerParticipant(wager1.ID, user2.DiscordID, option1.ID, 1000)
		err = groupWagerRepo.SaveParticipant(ctx, participant1)
		require.NoError(t, err)

		stats, err := groupWagerRepo.GetStats(ctx, user1.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 0, stats.TotalGroupWagers) // Only counts participations, not creations
		assert.Equal(t, 1, stats.TotalProposed)    // Created 1 wager
		assert.Equal(t, 0, stats.TotalWon)         // Didn't participate, so can't win
		assert.Equal(t, int64(0), stats.TotalWonAmount)
	})

	t.Run("user as participant only", func(t *testing.T) {
		// Create a group wager where user1 participates but doesn't create
		wager2 := testutil.CreateTestGroupWager(user2.DiscordID, "Test wager 2")
		option3 := testutil.CreateTestGroupWagerOption(0, "Option A", 0)
		option4 := testutil.CreateTestGroupWagerOption(0, "Option B", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, wager2, []*models.GroupWagerOption{option3, option4})
		require.NoError(t, err)

		// Add user1 as participant
		participant2 := testutil.CreateTestGroupWagerParticipant(wager2.ID, user1.DiscordID, option3.ID, 2000)
		err = groupWagerRepo.SaveParticipant(ctx, participant2)
		require.NoError(t, err)

		stats, err := groupWagerRepo.GetStats(ctx, user1.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 1, stats.TotalGroupWagers) // Now participates in 1 wager
		assert.Equal(t, 1, stats.TotalProposed)    // Still created 1 wager from previous test
		assert.Equal(t, 0, stats.TotalWon)         // Wager not resolved yet
		assert.Equal(t, int64(0), stats.TotalWonAmount)
	})

	t.Run("resolved wager - user wins", func(t *testing.T) {
		// Create balance history entries for payout tracking
		balanceHistory1 := testutil.CreateTestBalanceHistoryWithAmounts(
			user1.DiscordID, 100000, 103000, 3000, models.TransactionTypeGroupWagerWin)
		err := balanceHistoryRepo.Record(ctx, balanceHistory1)
		require.NoError(t, err)

		// Create an active group wager first
		wager3 := testutil.CreateTestGroupWager(user2.DiscordID, "Test wager 3")
		option5 := testutil.CreateTestGroupWagerOption(0, "Winning Option", 0)
		option6 := testutil.CreateTestGroupWagerOption(0, "Losing Option", 1)

		err = groupWagerRepo.CreateWithOptions(ctx, wager3, []*models.GroupWagerOption{option5, option6})
		require.NoError(t, err)

		// Now resolve the wager by updating to resolved state
		wager3.State = models.GroupWagerStateResolved
		wager3.ResolverDiscordID = &user2.DiscordID
		wager3.WinningOptionID = &option5.ID
		resolvedAt := time.Now()
		wager3.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, wager3)
		require.NoError(t, err)

		// Add user1 as winning participant first (without payout)
		winningParticipant := testutil.CreateTestGroupWagerParticipant(wager3.ID, user1.DiscordID, option5.ID, 1500)
		err = groupWagerRepo.SaveParticipant(ctx, winningParticipant)
		require.NoError(t, err)

		// Now update the participant with payout information
		winningParticipant.PayoutAmount = &[]int64{3000}[0]
		winningParticipant.BalanceHistoryID = &balanceHistory1.ID
		err = groupWagerRepo.UpdateParticipantPayouts(ctx, []*models.GroupWagerParticipant{winningParticipant})
		require.NoError(t, err)

		// Add losing participant
		losingParticipant := testutil.CreateTestGroupWagerParticipant(wager3.ID, user3.DiscordID, option6.ID, 1500)
		err = groupWagerRepo.SaveParticipant(ctx, losingParticipant)
		require.NoError(t, err)

		stats, err := groupWagerRepo.GetStats(ctx, user1.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 2, stats.TotalGroupWagers)         // Participates in 2 wagers total (from previous tests)
		assert.Equal(t, 1, stats.TotalProposed)            // Created 1 wager in "user as creator only" test
		assert.Equal(t, 1, stats.TotalWon)                 // Won 1 resolved wager
		assert.Equal(t, int64(3000), stats.TotalWonAmount) // Won 3000 bits
	})

	t.Run("resolved wager - user loses", func(t *testing.T) {
		// Create another active group wager where user1 loses
		wager4 := testutil.CreateTestGroupWager(user3.DiscordID, "Test wager 4")
		option7 := testutil.CreateTestGroupWagerOption(0, "Losing Option", 0)
		option8 := testutil.CreateTestGroupWagerOption(0, "Winning Option", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, wager4, []*models.GroupWagerOption{option7, option8})
		require.NoError(t, err)

		// Resolve the wager with option8 as winner
		wager4.State = models.GroupWagerStateResolved
		wager4.ResolverDiscordID = &user3.DiscordID
		wager4.WinningOptionID = &option8.ID
		resolvedAt := time.Now()
		wager4.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, wager4)
		require.NoError(t, err)

		// Add user1 as losing participant
		losingParticipant := testutil.CreateTestGroupWagerParticipant(wager4.ID, user1.DiscordID, option7.ID, 2500)
		err = groupWagerRepo.SaveParticipant(ctx, losingParticipant)
		require.NoError(t, err)

		// Add winning participant (user3)
		balanceHistory2 := testutil.CreateTestBalanceHistoryWithAmounts(
			user3.DiscordID, 100000, 105000, 5000, models.TransactionTypeGroupWagerWin)
		err = balanceHistoryRepo.Record(ctx, balanceHistory2)
		require.NoError(t, err)

		winningParticipant := testutil.CreateTestGroupWagerParticipant(wager4.ID, user3.DiscordID, option8.ID, 2500)
		err = groupWagerRepo.SaveParticipant(ctx, winningParticipant)
		require.NoError(t, err)

		// Now update the participant with payout information
		winningParticipant.PayoutAmount = &[]int64{5000}[0]
		winningParticipant.BalanceHistoryID = &balanceHistory2.ID
		err = groupWagerRepo.UpdateParticipantPayouts(ctx, []*models.GroupWagerParticipant{winningParticipant})
		require.NoError(t, err)

		stats, err := groupWagerRepo.GetStats(ctx, user1.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 3, stats.TotalGroupWagers)         // Now participates in 3 wagers
		assert.Equal(t, 1, stats.TotalProposed)            // Still created 1 wager
		assert.Equal(t, 1, stats.TotalWon)                 // Still won only 1 wager
		assert.Equal(t, int64(3000), stats.TotalWonAmount) // Still won 3000 total
	})

	t.Run("multiple wins - accumulates winnings", func(t *testing.T) {
		// Create another active group wager where user1 wins again
		wager5 := testutil.CreateTestGroupWager(user2.DiscordID, "Test wager 5")
		option9 := testutil.CreateTestGroupWagerOption(0, "Winning Option", 0)
		option10 := testutil.CreateTestGroupWagerOption(0, "Losing Option", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, wager5, []*models.GroupWagerOption{option9, option10})
		require.NoError(t, err)

		// Resolve the wager with option9 as winner
		wager5.State = models.GroupWagerStateResolved
		wager5.ResolverDiscordID = &user2.DiscordID
		wager5.WinningOptionID = &option9.ID
		resolvedAt := time.Now()
		wager5.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, wager5)
		require.NoError(t, err)

		// Create another balance history for second win
		balanceHistory3 := testutil.CreateTestBalanceHistoryWithAmounts(
			user1.DiscordID, 103000, 108000, 5000, models.TransactionTypeGroupWagerWin)
		err = balanceHistoryRepo.Record(ctx, balanceHistory3)
		require.NoError(t, err)

		// Add user1 as winning participant again (without payout first)
		winningParticipant2 := testutil.CreateTestGroupWagerParticipant(wager5.ID, user1.DiscordID, option9.ID, 3000)
		err = groupWagerRepo.SaveParticipant(ctx, winningParticipant2)
		require.NoError(t, err)

		// Now update the participant with payout information
		winningParticipant2.PayoutAmount = &[]int64{5000}[0]
		winningParticipant2.BalanceHistoryID = &balanceHistory3.ID
		err = groupWagerRepo.UpdateParticipantPayouts(ctx, []*models.GroupWagerParticipant{winningParticipant2})
		require.NoError(t, err)

		stats, err := groupWagerRepo.GetStats(ctx, user1.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 4, stats.TotalGroupWagers)         // Now participates in 4 wagers
		assert.Equal(t, 1, stats.TotalProposed)            // Still created 1 wager
		assert.Equal(t, 2, stats.TotalWon)                 // Won 2 wagers now
		assert.Equal(t, int64(8000), stats.TotalWonAmount) // Won 3000 + 5000 = 8000 total
	})

	t.Run("stats for different user", func(t *testing.T) {
		// Check stats for user3 who won one wager and lost others
		stats, err := groupWagerRepo.GetStats(ctx, user3.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 2, stats.TotalGroupWagers)         // Participated in 2 wagers
		assert.Equal(t, 1, stats.TotalProposed)            // Created 1 wager
		assert.Equal(t, 1, stats.TotalWon)                 // Won 1 wager
		assert.Equal(t, int64(5000), stats.TotalWonAmount) // Won 5000 bits
	})

	t.Run("user with no activity", func(t *testing.T) {
		// Create a user that has no group wager activity
		user4 := testutil.CreateTestUser(444444, "user4")
		_, err := userRepo.Create(ctx, user4.DiscordID, user4.Username, user4.Balance)
		require.NoError(t, err)

		stats, err := groupWagerRepo.GetStats(ctx, user4.DiscordID)
		require.NoError(t, err)

		assert.Equal(t, 0, stats.TotalGroupWagers)
		assert.Equal(t, 0, stats.TotalProposed)
		assert.Equal(t, 0, stats.TotalWon)
		assert.Equal(t, int64(0), stats.TotalWonAmount)
	})
}
