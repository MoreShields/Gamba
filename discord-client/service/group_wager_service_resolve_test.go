package service

import (
	"context"
	"fmt"
	"testing"

	"gambler/discord-client/config"
	"gambler/discord-client/events"
	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerService_ResolveGroupWager_BothTypes(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	// Test resolution for both wager types
	wagerTypeTests := []struct {
		name            string
		wagerType       models.GroupWagerType
		setupScenario   func() *GroupWagerScenario
		winningOption   int
		expectedPayouts map[int64]int64
	}{
		{
			name:      "pool wager - proportional payouts",
			wagerType: models.GroupWagerTypePool,
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Pool wager test").
					WithOptions("Yes", "No").
					WithUser(TestUser1ID, "user1", 10000).
					WithUser(TestUser2ID, "user2", 10000).
					WithUser(TestUser3ID, "user3", 10000).
					WithUser(TestUser4ID, "user4", 10000).
					WithParticipant(TestUser1ID, 0, 2000). // Yes - 2000
					WithParticipant(TestUser2ID, 0, 1000). // Yes - 1000
					WithParticipant(TestUser3ID, 1, 1500). // No - 1500
					WithParticipant(TestUser4ID, 1, 500).  // No - 500
					Build()
			},
			winningOption: 0, // Yes wins
			expectedPayouts: map[int64]int64{
				TestUser1ID: 3333, // 2000/3000 * 5000
				TestUser2ID: 1666, // 1000/3000 * 5000 (rounded down)
				TestUser3ID: 0,
				TestUser4ID: 0,
			},
		},
		{
			name:      "house wager - fixed odds payouts",
			wagerType: models.GroupWagerTypeHouse,
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithHouseWager(TestResolverID, "House wager test").
					WithOptions("Team A", "Team B").
					WithOdds(2.5, 1.8).
					WithUser(TestUser1ID, "user1", 10000).
					WithUser(TestUser2ID, "user2", 10000).
					WithUser(TestUser3ID, "user3", 10000).
					WithUser(TestUser4ID, "user4", 10000).
					WithParticipant(TestUser1ID, 0, 1000). // Team A - 1000
					WithParticipant(TestUser2ID, 0, 2000). // Team A - 2000
					WithParticipant(TestUser3ID, 1, 1500). // Team B - 1500
					WithParticipant(TestUser4ID, 1, 500).  // Team B - 500
					Build()
			},
			winningOption: 0, // Team A wins
			expectedPayouts: map[int64]int64{
				TestUser1ID: 2500, // 1000 * 2.5
				TestUser2ID: 5000, // 2000 * 2.5
				TestUser3ID: 0,
				TestUser4ID: 0,
			},
		},
		{
			name:      "house wager - underdog wins",
			wagerType: models.GroupWagerTypeHouse,
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithHouseWager(TestResolverID, "Underdog test").
					WithOptions("Favorite", "Underdog").
					WithOdds(1.5, 3.0). // Underdog has higher odds
					WithUser(TestUser1ID, "user1", 10000).
					WithUser(TestUser2ID, "user2", 10000).
					WithUser(TestUser3ID, "user3", 10000).
					WithParticipant(TestUser1ID, 0, 3000). // Favorite - 3000
					WithParticipant(TestUser2ID, 1, 1000). // Underdog - 1000
					WithParticipant(TestUser3ID, 1, 500).  // Underdog - 500
					Build()
			},
			winningOption: 1, // Underdog wins
			expectedPayouts: map[int64]int64{
				TestUser1ID: 0,
				TestUser2ID: 3000, // 1000 * 3.0
				TestUser3ID: 1500, // 500 * 3.0
			},
		},
	}

	for _, tt := range wagerTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mocks := NewTestMocks()
			helper := NewMockHelper(mocks)
			assertions := NewAssertionHelper(t)

			// Configure resolver
			service := NewGroupWagerService(
				mocks.GroupWagerRepo,
				mocks.UserRepo,
				mocks.BalanceHistoryRepo,
				mocks.EventPublisher,
			)
			service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

			// Build scenario
			scenario := tt.setupScenario()
			winningOptionID := scenario.Options[tt.winningOption].ID

			// Setup resolution mocks
			setupResolutionMocks(t, helper, mocks, scenario, scenario.Options[tt.winningOption].ID, tt.wagerType)

			// Execute
			resolverID := int64(TestResolverID)
			result, err := service.ResolveGroupWager(ctx, TestWagerID, &resolverID, winningOptionID)

			// Assert
			assertions.AssertNoError(err)
			assertions.AssertWagerResolved(result,
				len(getWinners(scenario.Participants, winningOptionID)),
				len(getLosers(scenario.Participants, winningOptionID)))
			assertions.AssertPayouts(result, tt.expectedPayouts)

			if tt.wagerType == models.GroupWagerTypePool {
				assertions.AssertPoolWagerPayouts(result)
			} else {
				assertions.AssertHouseWagerPayouts(result, scenario.Options[tt.winningOption])
			}

			mocks.AssertAllExpectations(t)
		})
	}
}

func TestGroupWagerService_ResolveGroupWager_ValidationErrors(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	validationTests := []struct {
		name          string
		setupFunc     func(*TestMocks, *MockHelper) int64 // returns winning option ID
		resolverID    int64
		expectedError string
	}{
		{
			name: "unauthorized resolver",
			setupFunc: func(mocks *TestMocks, helper *MockHelper) int64 {
				return TestOption1ID
			},
			resolverID:    TestUser1ID, // Not in resolver list
			expectedError: "not authorized to resolve",
		},
		{
			name: "wager not found",
			setupFunc: func(mocks *TestMocks, helper *MockHelper) int64 {
				mocks.GroupWagerRepo.On("GetDetailByID", ctx, int64(TestWagerID)).Return(nil, nil)
				return TestOption1ID
			},
			resolverID:    TestResolverID,
			expectedError: "group wager not found",
		},
		{
			name: "already resolved",
			setupFunc: func(mocks *TestMocks, helper *MockHelper) int64 {
				wager := &models.GroupWager{
					ID:    TestWagerID,
					State: models.GroupWagerStateResolved,
				}
				helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
					Wager:        wager,
					Options:      []*models.GroupWagerOption{},
					Participants: []*models.GroupWagerParticipant{},
				})
				return TestOption1ID
			},
			resolverID:    TestResolverID,
			expectedError: "cannot be resolved",
		},
		{
			name: "insufficient participants",
			setupFunc: func(mocks *TestMocks, helper *MockHelper) int64 {
				scenario := NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Test").
					WithOptions("Yes", "No").
					WithParticipant(TestUser1ID, 0, 1000). // Only 1 participant
					Build()

				helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
					Wager:        scenario.Wager,
					Options:      scenario.Options,
					Participants: scenario.Participants,
				})

				return scenario.Options[0].ID
			},
			resolverID:    TestResolverID,
			expectedError: "insufficient participants",
		},
		{
			name: "single option with all participants",
			setupFunc: func(mocks *TestMocks, helper *MockHelper) int64 {
				scenario := NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Test").
					WithOptions("Yes", "No").
					WithParticipant(TestUser1ID, 0, 1000).
					WithParticipant(TestUser2ID, 0, 2000).
					WithParticipant(TestUser3ID, 0, 1500). // All on option 0
					Build()

				helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
					Wager:        scenario.Wager,
					Options:      scenario.Options,
					Participants: scenario.Participants,
				})

				return scenario.Options[0].ID
			},
			resolverID:    TestResolverID,
			expectedError: "need participants on at least 2 different options",
		},
		{
			name: "invalid winning option",
			setupFunc: func(mocks *TestMocks, helper *MockHelper) int64 {
				scenario := NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Test").
					WithOptions("Yes", "No").
					WithParticipant(TestUser1ID, 0, 1000).
					WithParticipant(TestUser2ID, 1, 2000).
					WithParticipant(TestUser3ID, 0, 1500).
					Build()

				helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
					Wager:        scenario.Wager,
					Options:      scenario.Options,
					Participants: scenario.Participants,
				})

				return 99999 // Invalid option ID
			},
			resolverID:    TestResolverID,
			expectedError: "no option found with ID",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mocks := NewTestMocks()
			helper := NewMockHelper(mocks)
			assertions := NewAssertionHelper(t)

			service := NewGroupWagerService(
				mocks.GroupWagerRepo,
				mocks.UserRepo,
				mocks.BalanceHistoryRepo,
				mocks.EventPublisher,
			)
			service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

			// Setup test-specific mocks
			winningOptionID := tt.setupFunc(mocks, helper)

			// Execute
			resolverIDPtr := &tt.resolverID
			result, err := service.ResolveGroupWager(ctx, TestWagerID, resolverIDPtr, winningOptionID)

			// Assert
			require.Nil(t, result)
			assertions.AssertValidationError(err, tt.expectedError)

			mocks.AssertAllExpectations(t)
		})
	}
}

// Helper functions

func setupResolutionMocks(t *testing.T, helper *MockHelper, mocks *TestMocks, scenario *GroupWagerScenario, winningOptionID int64, wagerType models.GroupWagerType) {
	ctx := context.Background()

	// Basic lookups
	helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
		Wager:        scenario.Wager,
		Options:      scenario.Options,
		Participants: scenario.Participants,
	})

	// User lookups for all participants
	for _, participant := range scenario.Participants {
		if user, exists := scenario.Users[participant.DiscordID]; exists {
			helper.ExpectUserLookup(participant.DiscordID, user)
		}
	}

	// Balance updates based on wager type
	winners := getWinners(scenario.Participants, winningOptionID)
	losers := getLosers(scenario.Participants, winningOptionID)

	// Find winning option
	var winningOption *models.GroupWagerOption
	for _, opt := range scenario.Options {
		if opt.ID == winningOptionID {
			winningOption = opt
			break
		}
	}
	require.NotNil(t, winningOption)

	// Setup balance update expectations
	for _, winner := range winners {
		user := scenario.Users[winner.DiscordID]
		var payout int64
		var balanceChange int64

		if wagerType == models.GroupWagerTypePool {
			payout = winner.CalculatePayout(winningOption.TotalAmount, scenario.Wager.TotalPot)
		} else {
			payout = int64(float64(winner.Amount) * winningOption.OddsMultiplier)
		}
		// For both pool and house wagers: net win (payout - original bet)
		balanceChange = payout - winner.Amount

		// Always update balance and record history for all winners
		if true {
			newBalance := user.Balance + balanceChange
			helper.ExpectBalanceUpdate(winner.DiscordID, newBalance)
			helper.ExpectBalanceHistoryRecord(winner.DiscordID, balanceChange, models.TransactionTypeGroupWagerWin)
			helper.ExpectEventPublish("events.BalanceChangeEvent")
		}
	}

	for _, loser := range losers {
		user := scenario.Users[loser.DiscordID]
		var balanceChange int64

		// For both pool and house wagers: deduct bet amount from loser
		balanceChange = -loser.Amount

		// Always update balance and record history for all losers
		if true {
			newBalance := user.Balance + balanceChange
			helper.ExpectBalanceUpdate(loser.DiscordID, newBalance)
			helper.ExpectBalanceHistoryRecord(loser.DiscordID, balanceChange, models.TransactionTypeGroupWagerLoss)
			helper.ExpectEventPublish("events.BalanceChangeEvent")
		}
	}

	// Participant payout updates
	mocks.GroupWagerRepo.On("UpdateParticipantPayouts", ctx, mock.MatchedBy(func(participants []*models.GroupWagerParticipant) bool {
		// All participants should have payout amounts set
		for _, p := range participants {
			if p.PayoutAmount == nil {
				return false
			}
		}
		return len(participants) == len(scenario.Participants)
	})).Return(nil)

	// Wager state update
	mocks.GroupWagerRepo.On("Update", ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
		// For house wagers, allow nil resolver (system resolution)
		// For pool wagers, require non-nil resolver
		resolverValid := wagerType == models.GroupWagerTypeHouse || gw.ResolverDiscordID != nil
		return gw.ID == TestWagerID &&
			gw.State == models.GroupWagerStateResolved &&
			resolverValid &&
			gw.WinningOptionID != nil &&
			gw.ResolvedAt != nil
	})).Return(nil)

	// State change event
	mocks.EventPublisher.On("Publish", mock.MatchedBy(func(e events.GroupWagerStateChangeEvent) bool {
		return e.GroupWagerID == TestWagerID &&
			e.OldState == string(models.GroupWagerStateActive) &&
			e.NewState == string(models.GroupWagerStateResolved)
	})).Return()
}

func TestGroupWagerService_ResolveGroupWager_SystemUser(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	fixture := NewGroupWagerTestFixture(t)

	t.Run("system resolution of system wager", func(t *testing.T) {
		fixture.Reset()

		// Setup scenario with system-created wager
		scenario := NewGroupWagerScenario().
			WithHouseWager(0, "System house wager"). // 0 will be converted to nil
			WithOptions("Team A", "Team B").
			WithOdds(2.0, 1.5).
			WithUser(TestUser1ID, "user1", 10000).
			WithUser(TestUser2ID, "user2", 10000).
			WithParticipant(TestUser1ID, 0, 1000).
			WithParticipant(TestUser2ID, 1, 2000).
			Build()

		// Manually set creator to nil for system wager
		scenario.Wager.CreatorDiscordID = nil

		winningOptionID := scenario.Options[0].ID

		// Setup resolution mocks
		setupResolutionMocks(t, fixture.Helper, fixture.Mocks, scenario, winningOptionID, models.GroupWagerTypeHouse)

		// Execute with nil resolver (system resolution)
		result, err := fixture.Service.ResolveGroupWager(fixture.Ctx, TestWagerID, nil, winningOptionID)

		// Assert
		fixture.Assertions.AssertNoError(err)
		require.NotNil(t, result)
		assert.Equal(t, models.GroupWagerStateResolved, result.GroupWager.State)
		assert.Nil(t, result.GroupWager.ResolverDiscordID) // System resolution

		fixture.AssertAllMocks()
	})

	t.Run("resolver can resolve system wager", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers(TestResolverID)

		// Setup scenario with system-created wager
		scenario := NewGroupWagerScenario().
			WithHouseWager(0, "System house wager"). // 0 will be converted to nil
			WithOptions("Yes", "No").
			WithOdds(2.0, 1.5).
			WithUser(TestUser1ID, "user1", 10000).
			WithUser(TestUser2ID, "user2", 10000).
			WithParticipant(TestUser1ID, 0, 3000).
			WithParticipant(TestUser2ID, 1, 2000).
			Build()

		// Manually set creator to nil for system wager
		scenario.Wager.CreatorDiscordID = nil

		winningOptionID := scenario.Options[1].ID

		// Setup resolution mocks
		setupResolutionMocks(t, fixture.Helper, fixture.Mocks, scenario, winningOptionID, models.GroupWagerTypeHouse)

		// Execute with resolver ID
		resolverID := int64(TestResolverID)
		result, err := fixture.Service.ResolveGroupWager(fixture.Ctx, TestWagerID, &resolverID, winningOptionID)

		// Assert
		fixture.Assertions.AssertNoError(err)
		require.NotNil(t, result)
		assert.Equal(t, models.GroupWagerStateResolved, result.GroupWager.State)
		assert.Equal(t, int64(TestResolverID), *result.GroupWager.ResolverDiscordID)

		fixture.AssertAllMocks()
	})
}

func TestGroupWagerService_HouseWager_SpecificScenarios(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	fixture := NewGroupWagerTestFixture(t)

	t.Run("house wager odds remain fixed after bets", func(t *testing.T) {
		fixture.Reset()

		// Build scenario with fixed odds
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "Fixed odds test").
			WithOptions("Team A", "Team B").
			WithOdds(2.5, 1.8).
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		// Setup mocks for bet
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		fixture.Helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])
		fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil)
		fixture.Helper.ExpectNewParticipant(TestWagerID, TestUser1ID, TestOption1ID, 1000)
		fixture.Helper.ExpectOptionTotalUpdate(TestOption1ID, 1000)

		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == TestWagerID && gw.TotalPot == 1000
		})).Return(nil)

		// House wagers should NOT update odds
		// No call to UpdateAllOptionOdds expected

		// Execute
		participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)

		// Odds should remain unchanged
		assert.Equal(t, 2.5, scenario.Options[0].OddsMultiplier)
		assert.Equal(t, 1.8, scenario.Options[1].OddsMultiplier)

		fixture.AssertAllMocks()
	})

	t.Run("house wager all bets on losing option", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers(TestResolverID)

		// When all participants bet on the losing option, house keeps everything
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "House wins all").
			WithOptions("Option A", "Option B", "Option C").
			WithOdds(2.0, 3.0, 4.0).
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			WithUser(TestUser2ID, "user2", TestInitialBalance).
			WithUser(TestUser3ID, "user3", TestInitialBalance).
			WithParticipant(TestUser1ID, 0, 1000). // Bets on A
			WithParticipant(TestUser2ID, 1, 2000). // Bets on B
			WithParticipant(TestUser3ID, 1, 3000). // Also bets on B
			Build()

		// Option C wins, but nobody bet on it
		winningOptionID := scenario.Options[2].ID

		// Setup mocks - no balance changes for losers
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})

		// User lookups
		for _, p := range scenario.Participants {
			fixture.Helper.ExpectUserLookup(p.DiscordID, scenario.Users[p.DiscordID])
		}

		// All participants lose their bets
		fixture.Helper.ExpectBalanceUpdate(TestUser1ID, TestInitialBalance-1000) // 10000 - 1000 = 9000
		fixture.Helper.ExpectBalanceHistoryRecord(TestUser1ID, -1000, models.TransactionTypeGroupWagerLoss)
		fixture.Helper.ExpectEventPublish("events.BalanceChangeEvent")

		fixture.Helper.ExpectBalanceUpdate(TestUser2ID, TestInitialBalance-2000) // 10000 - 2000 = 8000
		fixture.Helper.ExpectBalanceHistoryRecord(TestUser2ID, -2000, models.TransactionTypeGroupWagerLoss)
		fixture.Helper.ExpectEventPublish("events.BalanceChangeEvent")

		fixture.Helper.ExpectBalanceUpdate(TestUser3ID, TestInitialBalance-3000) // 10000 - 3000 = 7000
		fixture.Helper.ExpectBalanceHistoryRecord(TestUser3ID, -3000, models.TransactionTypeGroupWagerLoss)
		fixture.Helper.ExpectEventPublish("events.BalanceChangeEvent")

		// Other resolution mocks
		fixture.Mocks.GroupWagerRepo.On("UpdateParticipantPayouts", fixture.Ctx, mock.Anything).Return(nil)
		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.Anything).Return(nil)
		fixture.Mocks.EventPublisher.On("Publish", mock.AnythingOfType("events.GroupWagerStateChangeEvent")).Return()

		// Execute
		resolverID := int64(TestResolverID)
		result, err := fixture.Service.ResolveGroupWager(fixture.Ctx, TestWagerID, &resolverID, winningOptionID)

		// Verify
		fixture.Assertions.AssertNoError(err)
		fixture.Assertions.AssertWagerResolved(result, 0, 3) // 0 winners, 3 losers

		// All payouts should be 0
		for _, loser := range result.Losers {
			assert.Equal(t, int64(0), *loser.PayoutAmount)
		}

		fixture.AssertAllMocks()
	})
}

func TestGroupWagerService_ResolveGroupWager_BalanceUpdateFailure(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()
	mocks := NewTestMocks()
	helper := NewMockHelper(mocks)

	service := NewGroupWagerService(
		mocks.GroupWagerRepo,
		mocks.UserRepo,
		mocks.BalanceHistoryRepo,
		mocks.EventPublisher,
	)
	service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

	// Setup scenario
	scenario := NewGroupWagerScenario().
		WithPoolWager(TestResolverID, "Test").
		WithOptions("Yes", "No").
		WithUser(TestUser1ID, "user1", 10000).
		WithUser(TestUser2ID, "user2", 10000).
		WithUser(TestUser3ID, "user3", 10000).
		WithParticipant(TestUser1ID, 0, 2000).
		WithParticipant(TestUser2ID, 0, 1000).
		WithParticipant(TestUser3ID, 1, 1000).
		Build()

	winningOptionID := scenario.Options[0].ID

	// Setup basic mocks
	helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
		Wager:        scenario.Wager,
		Options:      scenario.Options,
		Participants: scenario.Participants,
	})
	helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])

	// Simulate balance update failure
	mocks.UserRepo.On("UpdateBalance", ctx, int64(TestUser1ID), mock.AnythingOfType("int64")).Return(fmt.Errorf("database error"))

	// Execute
	resolverID := int64(TestResolverID)
	result, err := service.ResolveGroupWager(ctx, TestWagerID, &resolverID, winningOptionID)

	// Verify rollback
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update winner balance")

	mocks.AssertAllExpectations(t)
}
