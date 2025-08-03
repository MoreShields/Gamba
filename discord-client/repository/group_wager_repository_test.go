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

func TestGroupWagerRepository_GetStats(t *testing.T) {
	t.Parallel()
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

		err := groupWagerRepo.CreateWithOptions(ctx, wager1, []*entities.GroupWagerOption{option1, option2})
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

		err := groupWagerRepo.CreateWithOptions(ctx, wager2, []*entities.GroupWagerOption{option3, option4})
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
			user1.DiscordID, 100000, 103000, 3000, entities.TransactionTypeGroupWagerWin)
		err := balanceHistoryRepo.Record(ctx, balanceHistory1)
		require.NoError(t, err)

		// Create an active group wager first
		wager3 := testutil.CreateTestGroupWager(user2.DiscordID, "Test wager 3")
		option5 := testutil.CreateTestGroupWagerOption(0, "Winning Option", 0)
		option6 := testutil.CreateTestGroupWagerOption(0, "Losing Option", 1)

		err = groupWagerRepo.CreateWithOptions(ctx, wager3, []*entities.GroupWagerOption{option5, option6})
		require.NoError(t, err)

		// Now resolve the wager by updating to resolved state
		wager3.State = entities.GroupWagerStateResolved
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
		err = groupWagerRepo.UpdateParticipantPayouts(ctx, []*entities.GroupWagerParticipant{winningParticipant})
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

		err := groupWagerRepo.CreateWithOptions(ctx, wager4, []*entities.GroupWagerOption{option7, option8})
		require.NoError(t, err)

		// Resolve the wager with option8 as winner
		wager4.State = entities.GroupWagerStateResolved
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
			user3.DiscordID, 100000, 105000, 5000, entities.TransactionTypeGroupWagerWin)
		err = balanceHistoryRepo.Record(ctx, balanceHistory2)
		require.NoError(t, err)

		winningParticipant := testutil.CreateTestGroupWagerParticipant(wager4.ID, user3.DiscordID, option8.ID, 2500)
		err = groupWagerRepo.SaveParticipant(ctx, winningParticipant)
		require.NoError(t, err)

		// Now update the participant with payout information
		winningParticipant.PayoutAmount = &[]int64{5000}[0]
		winningParticipant.BalanceHistoryID = &balanceHistory2.ID
		err = groupWagerRepo.UpdateParticipantPayouts(ctx, []*entities.GroupWagerParticipant{winningParticipant})
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

		err := groupWagerRepo.CreateWithOptions(ctx, wager5, []*entities.GroupWagerOption{option9, option10})
		require.NoError(t, err)

		// Resolve the wager with option9 as winner
		wager5.State = entities.GroupWagerStateResolved
		wager5.ResolverDiscordID = &user2.DiscordID
		wager5.WinningOptionID = &option9.ID
		resolvedAt := time.Now()
		wager5.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, wager5)
		require.NoError(t, err)

		// Create another balance history for second win
		balanceHistory3 := testutil.CreateTestBalanceHistoryWithAmounts(
			user1.DiscordID, 103000, 108000, 5000, entities.TransactionTypeGroupWagerWin)
		err = balanceHistoryRepo.Record(ctx, balanceHistory3)
		require.NoError(t, err)

		// Add user1 as winning participant again (without payout first)
		winningParticipant2 := testutil.CreateTestGroupWagerParticipant(wager5.ID, user1.DiscordID, option9.ID, 3000)
		err = groupWagerRepo.SaveParticipant(ctx, winningParticipant2)
		require.NoError(t, err)

		// Now update the participant with payout information
		winningParticipant2.PayoutAmount = &[]int64{5000}[0]
		winningParticipant2.BalanceHistoryID = &balanceHistory3.ID
		err = groupWagerRepo.UpdateParticipantPayouts(ctx, []*entities.GroupWagerParticipant{winningParticipant2})
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

func TestGroupWagerRepository_GetGroupWagerPredictions(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	groupWagerRepo := NewGroupWagerRepository(testDB.DB)
	userRepo := NewUserRepository(testDB.DB)
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

	t.Run("no resolved wagers - returns empty predictions", func(t *testing.T) {
		predictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, predictions)
	})

	t.Run("active wagers only - returns empty predictions", func(t *testing.T) {
		// Create an active wager (not resolved)
		activeWager := testutil.CreateTestGroupWager(user1.DiscordID, "Active wager")
		option1 := testutil.CreateTestGroupWagerOption(0, "Win", 0)
		option2 := testutil.CreateTestGroupWagerOption(0, "Loss", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, activeWager, []*entities.GroupWagerOption{option1, option2})
		require.NoError(t, err)

		// Add participant
		participant := testutil.CreateTestGroupWagerParticipant(activeWager.ID, user2.DiscordID, option1.ID, 1000)
		err = groupWagerRepo.SaveParticipant(ctx, participant)
		require.NoError(t, err)

		predictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, predictions)
	})

	t.Run("resolved wager without external system", func(t *testing.T) {
		// Create and resolve a wager
		wager := testutil.CreateTestGroupWager(user1.DiscordID, "Test resolved wager")
		option1 := testutil.CreateTestGroupWagerOption(0, "Win", 0)
		option2 := testutil.CreateTestGroupWagerOption(0, "Loss", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, wager, []*entities.GroupWagerOption{option1, option2})
		require.NoError(t, err)

		// Resolve the wager
		wager.State = entities.GroupWagerStateResolved
		wager.ResolverDiscordID = &user1.DiscordID
		wager.WinningOptionID = &option1.ID
		resolvedAt := time.Now()
		wager.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, wager)
		require.NoError(t, err)

		// Add participants
		participant1 := testutil.CreateTestGroupWagerParticipant(wager.ID, user1.DiscordID, option1.ID, 1000)
		participant2 := testutil.CreateTestGroupWagerParticipant(wager.ID, user2.DiscordID, option2.ID, 1500)
		err = groupWagerRepo.SaveParticipant(ctx, participant1)
		require.NoError(t, err)
		err = groupWagerRepo.SaveParticipant(ctx, participant2)
		require.NoError(t, err)

		predictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, nil)
		require.NoError(t, err)
		require.Len(t, predictions, 2)

		// Sort predictions by DiscordID for consistent testing
		if predictions[0].DiscordID > predictions[1].DiscordID {
			predictions[0], predictions[1] = predictions[1], predictions[0]
		}

		// Check user1's prediction (correct)
		assert.Equal(t, user1.DiscordID, predictions[0].DiscordID)
		assert.Equal(t, wager.ID, predictions[0].GroupWagerID)
		assert.Equal(t, option1.ID, predictions[0].OptionID)
		assert.Equal(t, "Win", predictions[0].OptionText)
		assert.Equal(t, option1.ID, predictions[0].WinningOptionID)
		assert.Equal(t, int64(1000), predictions[0].Amount)
		assert.True(t, predictions[0].WasCorrect)
		assert.Nil(t, predictions[0].ExternalSystem)
		assert.Nil(t, predictions[0].ExternalID)

		// Check user2's prediction (incorrect)
		assert.Equal(t, user2.DiscordID, predictions[1].DiscordID)
		assert.Equal(t, wager.ID, predictions[1].GroupWagerID)
		assert.Equal(t, option2.ID, predictions[1].OptionID)
		assert.Equal(t, "Loss", predictions[1].OptionText)
		assert.Equal(t, option1.ID, predictions[1].WinningOptionID)
		assert.Equal(t, int64(1500), predictions[1].Amount)
		assert.False(t, predictions[1].WasCorrect)
		assert.Nil(t, predictions[1].ExternalSystem)
		assert.Nil(t, predictions[1].ExternalID)
	})

	t.Run("resolved wager with external system - League of Legends", func(t *testing.T) {
		// Create a LoL wager
		lolWager := testutil.CreateTestGroupWager(user2.DiscordID, "LoL game wager")
		lolWager.ExternalRef = &entities.ExternalReference{
			System: entities.SystemLeagueOfLegends,
			ID:     "RIOT_12345",
		}
		option3 := testutil.CreateTestGroupWagerOption(0, "Win", 0)
		option4 := testutil.CreateTestGroupWagerOption(0, "Loss", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, lolWager, []*entities.GroupWagerOption{option3, option4})
		require.NoError(t, err)

		// Resolve the wager
		lolWager.State = entities.GroupWagerStateResolved
		lolWager.ResolverDiscordID = &user2.DiscordID
		lolWager.WinningOptionID = &option4.ID
		resolvedAt := time.Now()
		lolWager.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, lolWager)
		require.NoError(t, err)

		// Add participants
		participant3 := testutil.CreateTestGroupWagerParticipant(lolWager.ID, user2.DiscordID, option3.ID, 2000)
		participant4 := testutil.CreateTestGroupWagerParticipant(lolWager.ID, user3.DiscordID, option4.ID, 2500)
		err = groupWagerRepo.SaveParticipant(ctx, participant3)
		require.NoError(t, err)
		err = groupWagerRepo.SaveParticipant(ctx, participant4)
		require.NoError(t, err)

		// Test filtering by external system
		lolSystem := entities.SystemLeagueOfLegends
		predictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, &lolSystem)
		require.NoError(t, err)
		require.Len(t, predictions, 2)

		// Sort predictions by DiscordID for consistent testing
		if predictions[0].DiscordID > predictions[1].DiscordID {
			predictions[0], predictions[1] = predictions[1], predictions[0]
		}

		// Check user2's prediction (incorrect)
		assert.Equal(t, user2.DiscordID, predictions[0].DiscordID)
		assert.Equal(t, lolWager.ID, predictions[0].GroupWagerID)
		assert.Equal(t, option3.ID, predictions[0].OptionID)
		assert.Equal(t, "Win", predictions[0].OptionText)
		assert.Equal(t, option4.ID, predictions[0].WinningOptionID)
		assert.Equal(t, int64(2000), predictions[0].Amount)
		assert.False(t, predictions[0].WasCorrect)
		assert.NotNil(t, predictions[0].ExternalSystem)
		assert.Equal(t, entities.SystemLeagueOfLegends, *predictions[0].ExternalSystem)
		assert.NotNil(t, predictions[0].ExternalID)
		assert.Equal(t, "RIOT_12345", *predictions[0].ExternalID)

		// Check user3's prediction (correct)
		assert.Equal(t, user3.DiscordID, predictions[1].DiscordID)
		assert.Equal(t, lolWager.ID, predictions[1].GroupWagerID)
		assert.Equal(t, option4.ID, predictions[1].OptionID)
		assert.Equal(t, "Loss", predictions[1].OptionText)
		assert.Equal(t, option4.ID, predictions[1].WinningOptionID)
		assert.Equal(t, int64(2500), predictions[1].Amount)
		assert.True(t, predictions[1].WasCorrect)
		assert.NotNil(t, predictions[1].ExternalSystem)
		assert.Equal(t, entities.SystemLeagueOfLegends, *predictions[1].ExternalSystem)
		assert.NotNil(t, predictions[1].ExternalID)
		assert.Equal(t, "RIOT_12345", *predictions[1].ExternalID)
	})

	t.Run("filtering by external system excludes other systems", func(t *testing.T) {
		// Create a TFT wager
		tftWager := testutil.CreateTestGroupWager(user3.DiscordID, "TFT game wager")
		tftWager.ExternalRef = &entities.ExternalReference{
			System: entities.SystemTFT,
			ID:     "TFT_67890",
		}
		option5 := testutil.CreateTestGroupWagerOption(0, "Top 4", 0)
		option6 := testutil.CreateTestGroupWagerOption(0, "Bottom 4", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, tftWager, []*entities.GroupWagerOption{option5, option6})
		require.NoError(t, err)

		// Resolve the TFT wager
		tftWager.State = entities.GroupWagerStateResolved
		tftWager.ResolverDiscordID = &user3.DiscordID
		tftWager.WinningOptionID = &option5.ID
		resolvedAt := time.Now()
		tftWager.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, tftWager)
		require.NoError(t, err)

		// Add participant
		participant5 := testutil.CreateTestGroupWagerParticipant(tftWager.ID, user1.DiscordID, option5.ID, 3000)
		err = groupWagerRepo.SaveParticipant(ctx, participant5)
		require.NoError(t, err)

		// Test filtering by LoL should not return TFT predictions
		lolSystem := entities.SystemLeagueOfLegends
		lolPredictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, &lolSystem)
		require.NoError(t, err)
		require.Len(t, lolPredictions, 2) // Only the LoL predictions from previous test

		// Test filtering by TFT should only return TFT predictions
		tftSystem := entities.SystemTFT
		tftPredictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, &tftSystem)
		require.NoError(t, err)
		require.Len(t, tftPredictions, 1)

		// Verify TFT prediction
		assert.Equal(t, user1.DiscordID, tftPredictions[0].DiscordID)
		assert.Equal(t, tftWager.ID, tftPredictions[0].GroupWagerID)
		assert.Equal(t, option5.ID, tftPredictions[0].OptionID)
		assert.Equal(t, "Top 4", tftPredictions[0].OptionText)
		assert.Equal(t, option5.ID, tftPredictions[0].WinningOptionID)
		assert.True(t, tftPredictions[0].WasCorrect)
		assert.NotNil(t, tftPredictions[0].ExternalSystem)
		assert.Equal(t, entities.SystemTFT, *tftPredictions[0].ExternalSystem)
		assert.NotNil(t, tftPredictions[0].ExternalID)
		assert.Equal(t, "TFT_67890", *tftPredictions[0].ExternalID)

		// Test no filter should return all predictions
		allPredictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, nil)
		require.NoError(t, err)
		require.Len(t, allPredictions, 5) // All predictions from all tests
	})

	t.Run("resolved wager without winning option - excluded from results", func(t *testing.T) {
		// Create a wager that is resolved but has no winning option (edge case)
		cancelledWager := testutil.CreateTestGroupWager(user1.DiscordID, "Cancelled wager")
		option7 := testutil.CreateTestGroupWagerOption(0, "Option A", 0)
		option8 := testutil.CreateTestGroupWagerOption(0, "Option B", 1)

		err := groupWagerRepo.CreateWithOptions(ctx, cancelledWager, []*entities.GroupWagerOption{option7, option8})
		require.NoError(t, err)

		// Resolve without winner (simulates cancellation)
		cancelledWager.State = entities.GroupWagerStateResolved
		cancelledWager.ResolverDiscordID = &user1.DiscordID
		// Don't set WinningOptionID - leave it nil
		resolvedAt := time.Now()
		cancelledWager.ResolvedAt = &resolvedAt
		err = groupWagerRepo.Update(ctx, cancelledWager)
		require.NoError(t, err)

		// Add participant
		participant6 := testutil.CreateTestGroupWagerParticipant(cancelledWager.ID, user2.DiscordID, option7.ID, 1000)
		err = groupWagerRepo.SaveParticipant(ctx, participant6)
		require.NoError(t, err)

		// Should still return the same number of predictions as before (cancelled wager excluded)
		predictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, nil)
		require.NoError(t, err)
		require.Len(t, predictions, 5) // Same as before, cancelled wager not included

		// Verify none of the predictions belong to the cancelled wager
		for _, pred := range predictions {
			assert.NotEqual(t, cancelledWager.ID, pred.GroupWagerID)
		}
	})

	t.Run("sorting is consistent", func(t *testing.T) {
		// Test that results are consistently ordered by discord_id, created_at
		predictions, err := groupWagerRepo.GetGroupWagerPredictions(ctx, nil)
		require.NoError(t, err)
		require.Greater(t, len(predictions), 0)

		// Verify sorting order (should be by discord_id, then created_at)
		for i := 1; i < len(predictions); i++ {
			if predictions[i-1].DiscordID == predictions[i].DiscordID {
				// Same user, can't easily test created_at ordering without more setup
				// but we know it's ordered by created_at for same user
				continue
			}
			assert.LessOrEqual(t, predictions[i-1].DiscordID, predictions[i].DiscordID)
		}
	})
}
