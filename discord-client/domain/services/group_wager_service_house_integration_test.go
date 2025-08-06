package services_test

import (
	"gambler/discord-client/domain/testhelpers"
	"context"
	"testing"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository"
	"gambler/discord-client/repository/testutil"
	"gambler/discord-client/domain/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHouseWager_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()

	// Create repositories
	userRepo := repository.NewUserRepository(testDB.DB)
	groupWagerRepo := repository.NewGroupWagerRepository(testDB.DB)
	balanceHistoryRepo := repository.NewBalanceHistoryRepository(testDB.DB)
	eventPublisher := &testhelpers.MockEventPublisher{}
	eventPublisher.On("Publish", mock.Anything).Return(nil)

	// Verify config is set up properly by TestMain
	// The test config already includes 999999 as a resolver ID
	cfg := config.Get()
	require.NotNil(t, cfg)
	require.Contains(t, cfg.ResolverDiscordIDs, int64(999999), "Test config should include 999999 as a resolver")

	// Create service
	// Note: This integration test will need to be updated once the repository
	// methods are fully implemented in the database layer
	groupWagerService := services.NewGroupWagerService(
		groupWagerRepo,
		userRepo,
		balanceHistoryRepo,
		eventPublisher,
	)

	t.Run("complete house wager lifecycle", func(t *testing.T) {
		// Create users
		creator, err := userRepo.Create(ctx, 999999, "creator", 100000)
		require.NoError(t, err)
		user1, err := userRepo.Create(ctx, 111111, "user1", 100000)
		require.NoError(t, err)
		user2, err := userRepo.Create(ctx, 222222, "user2", 100000)
		require.NoError(t, err)
		user3, err := userRepo.Create(ctx, 333333, "user3", 100000)
		require.NoError(t, err)

		// Create house wager with fixed odds
		wagerDetail, err := groupWagerService.CreateGroupWager(
			ctx,
			&creator.DiscordID,
			"Who will win the championship?",
			[]string{"Team Alpha", "Team Beta", "Team Gamma"},
			1440, // 24 hours
			123456,
			789012,
			entities.GroupWagerTypeHouse,
			[]float64{1.5, 2.5, 4.0}, // Fixed odds for each team
		)
		require.NoError(t, err)
		require.NotNil(t, wagerDetail)

		// Verify wager creation
		assert.Equal(t, entities.GroupWagerTypeHouse, wagerDetail.Wager.WagerType)
		assert.Len(t, wagerDetail.Options, 3)
		assert.Equal(t, 1.5, wagerDetail.Options[0].OddsMultiplier)
		assert.Equal(t, 2.5, wagerDetail.Options[1].OddsMultiplier)
		assert.Equal(t, 4.0, wagerDetail.Options[2].OddsMultiplier)

		// Place bets
		// User1 bets 10000 on favorite (Team Alpha)
		participant1, err := groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user1.DiscordID, wagerDetail.Options[0].ID, 10000)
		require.NoError(t, err)
		assert.Equal(t, int64(10000), participant1.Amount)

		// User2 bets 5000 on middle odds (Team Beta)
		participant2, err := groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user2.DiscordID, wagerDetail.Options[1].ID, 5000)
		require.NoError(t, err)
		assert.Equal(t, int64(5000), participant2.Amount)

		// User3 bets 2500 on underdog (Team Gamma)
		participant3, err := groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user3.DiscordID, wagerDetail.Options[2].ID, 2500)
		require.NoError(t, err)
		assert.Equal(t, int64(2500), participant3.Amount)

		// Verify balances are unchanged (bets only reserve funds, don't deduct immediately)
		user1Updated, err := userRepo.GetByDiscordID(ctx, user1.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000), user1Updated.Balance)         // Balance unchanged
		assert.Equal(t, int64(90000), user1Updated.AvailableBalance) // Available reduced by bet

		user2Updated, err := userRepo.GetByDiscordID(ctx, user2.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000), user2Updated.Balance)         // Balance unchanged
		assert.Equal(t, int64(95000), user2Updated.AvailableBalance) // Available reduced by bet

		user3Updated, err := userRepo.GetByDiscordID(ctx, user3.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000), user3Updated.Balance)         // Balance unchanged
		assert.Equal(t, int64(97500), user3Updated.AvailableBalance) // Available reduced by bet

		// Verify odds didn't change after bets (house wager specific)
		updatedDetail, err := groupWagerService.GetGroupWagerDetail(ctx, wagerDetail.Wager.ID)
		require.NoError(t, err)
		assert.Equal(t, 1.5, updatedDetail.Options[0].OddsMultiplier)
		assert.Equal(t, 2.5, updatedDetail.Options[1].OddsMultiplier)
		assert.Equal(t, 4.0, updatedDetail.Options[2].OddsMultiplier)

		// Resolve wager - underdog wins!
		creatorID := creator.DiscordID
		result, err := groupWagerService.ResolveGroupWager(
			ctx,
			wagerDetail.Wager.ID,
			&creatorID,
			wagerDetail.Options[2].ID, // Team Gamma wins
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify resolution
		assert.Equal(t, entities.GroupWagerStateResolved, result.GroupWager.State)
		assert.Len(t, result.Winners, 1)
		assert.Len(t, result.Losers, 2)

		// Verify payouts
		assert.Equal(t, user3.DiscordID, result.Winners[0].DiscordID)
		assert.Equal(t, int64(10000), *result.Winners[0].PayoutAmount) // 2500 * 4.0

		// Verify final balances after resolution
		user1Final, err := userRepo.GetByDiscordID(ctx, user1.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(90000), user1Final.Balance) // Lost: 100000 - 10000

		user2Final, err := userRepo.GetByDiscordID(ctx, user2.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(95000), user2Final.Balance) // Lost: 100000 - 5000

		user3Final, err := userRepo.GetByDiscordID(ctx, user3.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(107500), user3Final.Balance) // Won: 100000 - 2500 + 10000 = 107500
	})

	t.Run("house wager with bet updates", func(t *testing.T) {
		// Create users (need at least 3 participants)
		user, err := userRepo.Create(ctx, 444444, "bettor", 50000)
		require.NoError(t, err)
		user2, err := userRepo.Create(ctx, 444445, "bettor2", 50000)
		require.NoError(t, err)
		user3, err := userRepo.Create(ctx, 444446, "bettor3", 50000)
		require.NoError(t, err)

		// Create house wager
		testCreatorID := int64(999999)
		wagerDetail, err := groupWagerService.CreateGroupWager(
			ctx,
			&testCreatorID,
			"Coin flip",
			[]string{"Heads", "Tails"},
			1440, // 24 hours
			234567,
			890123,
			entities.GroupWagerTypeHouse,
			[]float64{2.0, 2.0}, // Even odds
		)
		require.NoError(t, err)

		// Initial bet on Heads
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user.DiscordID, wagerDetail.Options[0].ID, 1000)
		require.NoError(t, err)

		// Add more participants to meet minimum requirement (bet on Heads so they lose)
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user2.DiscordID, wagerDetail.Options[0].ID, 2000)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user3.DiscordID, wagerDetail.Options[0].ID, 1500)
		require.NoError(t, err)

		// Check balance after first bet (balance unchanged, available balance reduced)
		userAfterBet1, err := userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(50000), userAfterBet1.Balance)
		assert.Equal(t, int64(49000), userAfterBet1.AvailableBalance)

		// Update bet to higher amount
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user.DiscordID, wagerDetail.Options[0].ID, 5000)
		require.NoError(t, err)

		// Check balance after update (balance unchanged, available balance reduced by additional 4000)
		userAfterBet2, err := userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(50000), userAfterBet2.Balance)
		assert.Equal(t, int64(45000), userAfterBet2.AvailableBalance)

		// Change to different option
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, user.DiscordID, wagerDetail.Options[1].ID, 3000)
		require.NoError(t, err)

		// Check balance (balance unchanged, available balance: 50000 - 3000 = 47000)
		userAfterBet3, err := userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(50000), userAfterBet3.Balance)
		assert.Equal(t, int64(47000), userAfterBet3.AvailableBalance)

		// Resolve with Tails winning
		resolverID := int64(999999)
		result, err := groupWagerService.ResolveGroupWager(
			ctx,
			wagerDetail.Wager.ID,
			&resolverID,
			wagerDetail.Options[1].ID,
		)
		require.NoError(t, err)

		// Verify payout (only user who bet on Tails wins)
		assert.Len(t, result.Winners, 1)
		assert.Equal(t, user.DiscordID, result.Winners[0].DiscordID)
		assert.Equal(t, int64(6000), *result.Winners[0].PayoutAmount) // 3000 * 2.0

		// Verify final balance after resolution
		userFinal, err := userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(53000), userFinal.Balance) // 50000 - 3000 + 6000 = 53000
	})

	t.Run("house wager all participants lose", func(t *testing.T) {
		// Create users (need at least 3 participants)
		loser1, err := userRepo.Create(ctx, 555555, "loser1", 20000)
		require.NoError(t, err)
		loser2, err := userRepo.Create(ctx, 666666, "loser2", 20000)
		require.NoError(t, err)
		loser3, err := userRepo.Create(ctx, 777777, "loser3", 20000)
		require.NoError(t, err)

		// Create house wager
		testCreatorID2 := int64(999999)
		wagerDetail, err := groupWagerService.CreateGroupWager(
			ctx,
			&testCreatorID2,
			"Pick the winner",
			[]string{"Option A", "Option B", "Option C"},
			1440, // 24 hours
			345678,
			901234,
			entities.GroupWagerTypeHouse,
			[]float64{3.0, 2.0, 1.5},
		)
		require.NoError(t, err)

		// Two users bet on Option B, one on Option C (Option A wins)
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, loser1.DiscordID, wagerDetail.Options[1].ID, 5000)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, loser2.DiscordID, wagerDetail.Options[1].ID, 8000)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, wagerDetail.Wager.ID, loser3.DiscordID, wagerDetail.Options[2].ID, 3000)
		require.NoError(t, err)

		// Verify balances after bets (balance unchanged, available balance reduced)
		loser1After, err := userRepo.GetByDiscordID(ctx, loser1.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(20000), loser1After.Balance)
		assert.Equal(t, int64(15000), loser1After.AvailableBalance)

		loser2After, err := userRepo.GetByDiscordID(ctx, loser2.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(20000), loser2After.Balance)
		assert.Equal(t, int64(12000), loser2After.AvailableBalance)

		loser3After, err := userRepo.GetByDiscordID(ctx, loser3.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(20000), loser3After.Balance)
		assert.Equal(t, int64(17000), loser3After.AvailableBalance)

		// Resolve with Option A winning (no one bet on it)
		resolverID := int64(999999)
		result, err := groupWagerService.ResolveGroupWager(
			ctx,
			wagerDetail.Wager.ID,
			&resolverID,
			wagerDetail.Options[0].ID,
		)
		require.NoError(t, err)

		// Verify no winners
		assert.Len(t, result.Winners, 0)
		assert.Len(t, result.Losers, 3)

		// Verify final balances (bet amounts deducted at resolution)
		loser1Final, err := userRepo.GetByDiscordID(ctx, loser1.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(15000), loser1Final.Balance) // 20000 - 5000

		loser2Final, err := userRepo.GetByDiscordID(ctx, loser2.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(12000), loser2Final.Balance) // 20000 - 8000

		loser3Final, err := userRepo.GetByDiscordID(ctx, loser3.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(17000), loser3Final.Balance) // 20000 - 3000
	})
}
