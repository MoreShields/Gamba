package service_test

import (
	"context"
	"testing"
	"time"

	"gambler/config"
	"gambler/models"
	"gambler/repository"
	"gambler/repository/testutil"
	"gambler/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerResolution_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()

	// Create repositories
	userRepo := repository.NewUserRepository(testDB.DB)
	groupWagerRepo := repository.NewGroupWagerRepository(testDB.DB)
	balanceHistoryRepo := repository.NewBalanceHistoryRepository(testDB.DB)
	eventPublisher := &service.MockEventPublisher{}
	// Allow any publish calls
	eventPublisher.On("Publish", mock.Anything).Return()

	// Set up config with resolver IDs
	cfg := config.Get()
	originalResolvers := cfg.ResolverDiscordIDs
	cfg.ResolverDiscordIDs = []int64{999999, 999991}
	defer func() {
		cfg.ResolverDiscordIDs = originalResolvers
	}()

	// Create service
	groupWagerService := service.NewGroupWagerService(
		groupWagerRepo,
		userRepo,
		balanceHistoryRepo,
		eventPublisher,
	)

	t.Run("complete resolution workflow with multiple participants", func(t *testing.T) {
		// Create users with initial balances
		user1, err := userRepo.Create(ctx, 111111, "user1", 100000)
		require.NoError(t, err)
		user2, err := userRepo.Create(ctx, 222222, "user2", 100000)
		require.NoError(t, err)
		user3, err := userRepo.Create(ctx, 333333, "user3", 100000)
		require.NoError(t, err)
		user4, err := userRepo.Create(ctx, 444444, "user4", 100000)
		require.NoError(t, err)
		resolver, err := userRepo.Create(ctx, 999999, "resolver", 100000)
		require.NoError(t, err)

		// Create group wager
		votingEndsAt := time.Now().Add(24 * time.Hour)
		groupWager := &models.GroupWager{
			CreatorDiscordID:    resolver.DiscordID,
			Condition:           "Will team win the championship?",
			State:               models.GroupWagerStateActive,
			TotalPot:            0,
			MinParticipants:     2,
			VotingPeriodMinutes: 1440,
			VotingStartsAt:      &time.Time{},
			VotingEndsAt:        &votingEndsAt,
			MessageID:           123456,
			ChannelID:           789012,
		}

		options := []*models.GroupWagerOption{
			{OptionText: "Yes", OptionOrder: 0},
			{OptionText: "No", OptionOrder: 1},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		// Get the created options with IDs
		detail, err := groupWagerRepo.GetDetailByID(ctx, groupWager.ID)
		require.NoError(t, err)
		require.Len(t, detail.Options, 2)

		yesOptionID := detail.Options[0].ID
		noOptionID := detail.Options[1].ID

		// Place bets
		// Users 1 and 2 bet on "Yes"
		participant1, err := groupWagerService.PlaceBet(ctx, groupWager.ID, user1.DiscordID, yesOptionID, 30000)
		require.NoError(t, err)
		assert.Equal(t, int64(30000), participant1.Amount)

		participant2, err := groupWagerService.PlaceBet(ctx, groupWager.ID, user2.DiscordID, yesOptionID, 20000)
		require.NoError(t, err)
		assert.Equal(t, int64(20000), participant2.Amount)

		// Users 3 and 4 bet on "No"
		participant3, err := groupWagerService.PlaceBet(ctx, groupWager.ID, user3.DiscordID, noOptionID, 25000)
		require.NoError(t, err)
		assert.Equal(t, int64(25000), participant3.Amount)

		participant4, err := groupWagerService.PlaceBet(ctx, groupWager.ID, user4.DiscordID, noOptionID, 15000)
		require.NoError(t, err)
		assert.Equal(t, int64(15000), participant4.Amount)

		// Verify total pot
		updatedWager, err := groupWagerRepo.GetByID(ctx, groupWager.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(90000), updatedWager.TotalPot)

		// Resolve the wager with "Yes" as winner
		result, err := groupWagerService.ResolveGroupWager(ctx, groupWager.ID, resolver.DiscordID, yesOptionID)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify result structure
		assert.Equal(t, models.GroupWagerStateResolved, result.GroupWager.State)
		assert.Equal(t, yesOptionID, result.WinningOption.ID)
		assert.Len(t, result.Winners, 2)
		assert.Len(t, result.Losers, 2)
		assert.Equal(t, int64(90000), result.TotalPot)

		// Verify payout calculations
		// User1 should get 30000/50000 * 90000 = 54000
		assert.Equal(t, int64(54000), result.PayoutDetails[user1.DiscordID])
		// User2 should get 20000/50000 * 90000 = 36000
		assert.Equal(t, int64(36000), result.PayoutDetails[user2.DiscordID])
		// Losers get 0
		assert.Equal(t, int64(0), result.PayoutDetails[user3.DiscordID])
		assert.Equal(t, int64(0), result.PayoutDetails[user4.DiscordID])

		// Verify balances were updated correctly
		updatedUser1, err := userRepo.GetByDiscordID(ctx, user1.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000+54000-30000), updatedUser1.Balance) // Initial + payout - bet = 124000

		updatedUser2, err := userRepo.GetByDiscordID(ctx, user2.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000+36000-20000), updatedUser2.Balance) // Initial + payout - bet = 116000

		updatedUser3, err := userRepo.GetByDiscordID(ctx, user3.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000-25000), updatedUser3.Balance) // Initial - bet = 75000

		updatedUser4, err := userRepo.GetByDiscordID(ctx, user4.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000-15000), updatedUser4.Balance) // Initial - bet = 85000

		// Verify balance history was created
		user1History, err := balanceHistoryRepo.GetByUser(ctx, user1.DiscordID, 10)
		require.NoError(t, err)
		assert.Len(t, user1History, 1)
		assert.Equal(t, models.TransactionTypeGroupWagerWin, user1History[0].TransactionType)
		assert.Equal(t, int64(24000), user1History[0].ChangeAmount) // Net win

		user3History, err := balanceHistoryRepo.GetByUser(ctx, user3.DiscordID, 10)
		require.NoError(t, err)
		assert.Len(t, user3History, 1)
		assert.Equal(t, models.TransactionTypeGroupWagerLoss, user3History[0].TransactionType)
		assert.Equal(t, int64(-25000), user3History[0].ChangeAmount)

		// Verify participants were updated with payouts
		updatedDetail, err := groupWagerRepo.GetDetailByID(ctx, groupWager.ID)
		require.NoError(t, err)
		for _, p := range updatedDetail.Participants {
			require.NotNil(t, p.PayoutAmount)
			if p.OptionID == yesOptionID {
				assert.Greater(t, *p.PayoutAmount, int64(0))
				assert.NotNil(t, p.BalanceHistoryID)
			} else {
				assert.Equal(t, int64(0), *p.PayoutAmount)
				assert.NotNil(t, p.BalanceHistoryID)
			}
		}
	})

	t.Run("resolution with option switching before resolution", func(t *testing.T) {
		// Create users
		user5, err := userRepo.Create(ctx, 555555, "user5", 100000)
		require.NoError(t, err)
		user6, err := userRepo.Create(ctx, 666666, "user6", 100000)
		require.NoError(t, err)

		// Create group wager
		votingEndsAt := time.Now().Add(24 * time.Hour)
		groupWager := &models.GroupWager{
			CreatorDiscordID:    999999,
			Condition:           "Will it rain tomorrow?",
			State:               models.GroupWagerStateActive,
			TotalPot:            0,
			MinParticipants:     2,
			VotingPeriodMinutes: 1440,
			VotingStartsAt:      &time.Time{},
			VotingEndsAt:        &votingEndsAt,
			MessageID:           234567,
			ChannelID:           789012,
		}

		options := []*models.GroupWagerOption{
			{OptionText: "Rain", OptionOrder: 0},
			{OptionText: "No Rain", OptionOrder: 1},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		detail, err := groupWagerRepo.GetDetailByID(ctx, groupWager.ID)
		require.NoError(t, err)

		rainOptionID := detail.Options[0].ID
		noRainOptionID := detail.Options[1].ID

		// User5 bets on Rain initially
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user5.DiscordID, rainOptionID, 10000)
		require.NoError(t, err)

		// User6 bets on No Rain
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user6.DiscordID, noRainOptionID, 15000)
		require.NoError(t, err)

		// User5 switches to No Rain with higher amount
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user5.DiscordID, noRainOptionID, 20000)
		require.NoError(t, err)

		// Verify total pot is correct (20000 + 15000)
		updatedWager, err := groupWagerRepo.GetByID(ctx, groupWager.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(35000), updatedWager.TotalPot)

		// Add a third participant to meet the 2-option requirement
		user7, err := userRepo.Create(ctx, 777777, "user7", 100000)
		require.NoError(t, err)
		
		// User7 stays on Rain option
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user7.DiscordID, rainOptionID, 5000)
		require.NoError(t, err)

		// Resolve with No Rain as winner
		result, err := groupWagerService.ResolveGroupWager(ctx, groupWager.ID, 999999, noRainOptionID)
		require.NoError(t, err)

		// Two winners and one loser
		assert.Len(t, result.Winners, 2)
		assert.Len(t, result.Losers, 1)

		// Verify payouts
		// Total pot is 40000 (20k + 15k + 5k)
		// Winning option has 35000 (20k + 15k)
		// User5 gets 20000/35000 * 40000 = 22857
		// User6 gets 15000/35000 * 40000 = 17143
		assert.Equal(t, int64(22857), result.PayoutDetails[user5.DiscordID])
		assert.Equal(t, int64(17142), result.PayoutDetails[user6.DiscordID]) // Integer division rounding
		assert.Equal(t, int64(0), result.PayoutDetails[user7.DiscordID])

		// Verify balances
		updatedUser5, err := userRepo.GetByDiscordID(ctx, user5.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000+22857-20000), updatedUser5.Balance) // 102857

		updatedUser6, err := userRepo.GetByDiscordID(ctx, user6.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000+17142-15000), updatedUser6.Balance) // 102142

		updatedUser7, err := userRepo.GetByDiscordID(ctx, user7.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000-5000), updatedUser7.Balance) // 95000
	})

	t.Run("concurrent resolution attempts", func(t *testing.T) {
		// Create users
		user8, err := userRepo.Create(ctx, 888888, "user8", 100000)
		require.NoError(t, err)
		user9, err := userRepo.Create(ctx, 999111, "user9", 100000)
		require.NoError(t, err)

		// Create another resolver
		resolver2, err := userRepo.Create(ctx, 999998, "resolver2", 100000)
		require.NoError(t, err)
		cfg.ResolverDiscordIDs = append(cfg.ResolverDiscordIDs, resolver2.DiscordID)

		// Create group wager
		votingEndsAt := time.Now().Add(24 * time.Hour)
		groupWager := &models.GroupWager{
			CreatorDiscordID:    999999,
			Condition:           "Concurrent test wager",
			State:               models.GroupWagerStateActive,
			TotalPot:            0,
			MinParticipants:     2,
			VotingPeriodMinutes: 1440,
			VotingStartsAt:      &time.Time{},
			VotingEndsAt:        &votingEndsAt,
			MessageID:           345678,
			ChannelID:           789012,
		}

		options := []*models.GroupWagerOption{
			{OptionText: "Option A", OptionOrder: 0},
			{OptionText: "Option B", OptionOrder: 1},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		detail, err := groupWagerRepo.GetDetailByID(ctx, groupWager.ID)
		require.NoError(t, err)

		optionAID := detail.Options[0].ID
		optionBID := detail.Options[1].ID

		// Place bets
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user8.DiscordID, optionAID, 5000)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user9.DiscordID, optionBID, 5000)
		require.NoError(t, err)

		// First resolution should succeed
		result1, err1 := groupWagerService.ResolveGroupWager(ctx, groupWager.ID, 999999, optionAID)
		require.NoError(t, err1)
		require.NotNil(t, result1)

		// Second resolution attempt should fail
		result2, err2 := groupWagerService.ResolveGroupWager(ctx, groupWager.ID, resolver2.DiscordID, optionBID)
		assert.Error(t, err2)
		assert.Nil(t, result2)
		assert.Contains(t, err2.Error(), "cannot be resolved")
	})
}

func TestGroupWagerResolution_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()

	// Create repositories
	userRepo := repository.NewUserRepository(testDB.DB)
	groupWagerRepo := repository.NewGroupWagerRepository(testDB.DB)
	balanceHistoryRepo := repository.NewBalanceHistoryRepository(testDB.DB)
	eventPublisher := &service.MockEventPublisher{}
	// Allow any publish calls
	eventPublisher.On("Publish", mock.Anything).Return()

	// Set up config with resolver IDs
	cfg := config.Get()
	originalResolvers := cfg.ResolverDiscordIDs
	cfg.ResolverDiscordIDs = []int64{999999, 999991}
	defer func() {
		cfg.ResolverDiscordIDs = originalResolvers
	}()

	// Create service
	groupWagerService := service.NewGroupWagerService(
		groupWagerRepo,
		userRepo,
		balanceHistoryRepo,
		eventPublisher,
	)

	t.Run("resolution with single participant on winning option", func(t *testing.T) {
		// Create resolver user first
		resolver, err := userRepo.Create(ctx, 999999, "resolver", 100000)
		require.NoError(t, err)
		
		// Create users
		winner, err := userRepo.Create(ctx, 101010, "winner", 100000)
		require.NoError(t, err)
		loser1, err := userRepo.Create(ctx, 202020, "loser1", 100000)
		require.NoError(t, err)
		loser2, err := userRepo.Create(ctx, 303030, "loser2", 100000)
		require.NoError(t, err)

		// Create group wager
		votingEndsAt := time.Now().Add(24 * time.Hour)
		groupWager := &models.GroupWager{
			CreatorDiscordID:    resolver.DiscordID,
			Condition:           "Single winner test",
			State:               models.GroupWagerStateActive,
			TotalPot:            0,
			MinParticipants:     2,
			VotingPeriodMinutes: 1440,
			VotingStartsAt:      &time.Time{},
			VotingEndsAt:        &votingEndsAt,
			MessageID:           456789,
			ChannelID:           789012,
		}

		options := []*models.GroupWagerOption{
			{OptionText: "Winning", OptionOrder: 0},
			{OptionText: "Losing", OptionOrder: 1},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		detail, err := groupWagerRepo.GetDetailByID(ctx, groupWager.ID)
		require.NoError(t, err)

		winningOptionID := detail.Options[0].ID
		losingOptionID := detail.Options[1].ID

		// Place bets - only one on winning option
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, winner.DiscordID, winningOptionID, 10000)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, loser1.DiscordID, losingOptionID, 30000)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, loser2.DiscordID, losingOptionID, 20000)
		require.NoError(t, err)

		// Resolve
		result, err := groupWagerService.ResolveGroupWager(ctx, groupWager.ID, resolver.DiscordID, winningOptionID)
		require.NoError(t, err)

		// Single winner gets entire pot
		assert.Len(t, result.Winners, 1)
		assert.Len(t, result.Losers, 2)
		assert.Equal(t, int64(60000), result.PayoutDetails[winner.DiscordID]) // Gets entire pot

		// Verify balance
		updatedWinner, err := userRepo.GetByDiscordID(ctx, winner.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000+60000-10000), updatedWinner.Balance) // Initial + pot - bet = 150000
	})

	t.Run("resolution with very small bets and rounding", func(t *testing.T) {
		// Create resolver user if not exists
		resolver2, err := userRepo.Create(ctx, 999991, "resolver2", 100000)
		require.NoError(t, err)
		
		// Create users
		user1, err := userRepo.Create(ctx, 404040, "user1", 100000)
		require.NoError(t, err)
		user2, err := userRepo.Create(ctx, 505050, "user2", 100000)
		require.NoError(t, err)
		user3, err := userRepo.Create(ctx, 606060, "user3", 100000)
		require.NoError(t, err)

		// Create group wager
		votingEndsAt := time.Now().Add(24 * time.Hour)
		groupWager := &models.GroupWager{
			CreatorDiscordID:    resolver2.DiscordID,
			Condition:           "Rounding test",
			State:               models.GroupWagerStateActive,
			TotalPot:            0,
			MinParticipants:     2,
			VotingPeriodMinutes: 1440,
			VotingStartsAt:      &time.Time{},
			VotingEndsAt:        &votingEndsAt,
			MessageID:           567890,
			ChannelID:           789012,
		}

		options := []*models.GroupWagerOption{
			{OptionText: "A", OptionOrder: 0},
			{OptionText: "B", OptionOrder: 1},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		detail, err := groupWagerRepo.GetDetailByID(ctx, groupWager.ID)
		require.NoError(t, err)

		optionAID := detail.Options[0].ID
		optionBID := detail.Options[1].ID

		// Place bets with amounts that will cause rounding
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user1.DiscordID, optionAID, 333)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user2.DiscordID, optionAID, 667)
		require.NoError(t, err)
		_, err = groupWagerService.PlaceBet(ctx, groupWager.ID, user3.DiscordID, optionBID, 1000)
		require.NoError(t, err)

		// Resolve with option A as winner
		result, err := groupWagerService.ResolveGroupWager(ctx, groupWager.ID, resolver2.DiscordID, optionAID)
		require.NoError(t, err)

		// Total pot is 2000, winning option has 1000
		// User1 should get 333/1000 * 2000 = 666
		// User2 should get 667/1000 * 2000 = 1334
		assert.Equal(t, int64(666), result.PayoutDetails[user1.DiscordID])
		assert.Equal(t, int64(1334), result.PayoutDetails[user2.DiscordID])
		assert.Equal(t, int64(666+1334), result.PayoutDetails[user1.DiscordID]+result.PayoutDetails[user2.DiscordID]) // Should equal total pot
	})
}