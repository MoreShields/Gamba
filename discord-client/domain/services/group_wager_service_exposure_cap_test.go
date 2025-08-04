package services

import (
	"context"
	"testing"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function to get participant by user ID
func getParticipantByUserID(participants []*entities.GroupWagerParticipant, userID int64) *entities.GroupWagerParticipant {
	for _, p := range participants {
		if p.DiscordID == userID {
			return p
		}
	}
	return nil
}

func TestGroupWagerService_ResolveGroupWager_ExposureCap(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	ctx := context.Background()

	exposureCapTests := []struct {
		name            string
		setupScenario   func() *GroupWagerScenario
		winningOption   int
		expectedPayouts map[int64]int64
		expectedLosses  map[int64]int64
		description     string
	}{
		{
			name: "small winner caps large losers",
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Small winner test").
					WithOptions("Small", "Large").
					WithUser(TestUser1ID, "user1", 100000).
					WithUser(TestUser2ID, "user2", 100000).
					WithUser(TestUser3ID, "user3", 100000).
					WithParticipant(TestUser1ID, 0, 10).    // Small bet wins
					WithParticipant(TestUser2ID, 1, 1000).  // Large bet loses
					WithParticipant(TestUser3ID, 1, 2000).  // Large bet loses
					Build()
			},
			winningOption: 0, // Small wins
			expectedPayouts: map[int64]int64{
				TestUser1ID: 30,  // 10 + 10 + 10 = 30 total pool
				TestUser2ID: 0,
				TestUser3ID: 0,
			},
			expectedLosses: map[int64]int64{
				TestUser2ID: -10,  // Capped at 10
				TestUser3ID: -10,  // Capped at 10
			},
			description: "Large losers should only lose up to the highest winner bet (10)",
		},
		{
			name: "multiple winners with different amounts",
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Multiple winners test").
					WithOptions("Win", "Lose").
					WithUser(TestUser1ID, "user1", 100000).
					WithUser(TestUser2ID, "user2", 100000).
					WithUser(TestUser3ID, "user3", 100000).
					WithUser(TestUser4ID, "user4", 100000).
					WithParticipant(TestUser1ID, 0, 100).   // Winner
					WithParticipant(TestUser2ID, 0, 200).   // Winner (highest)
					WithParticipant(TestUser3ID, 1, 300).   // Loser
					WithParticipant(TestUser4ID, 1, 500).   // Loser
					Build()
			},
			winningOption: 0,
			expectedPayouts: map[int64]int64{
				TestUser1ID: 233,  // 100/300 * 700 = 233
				TestUser2ID: 466,  // 200/300 * 700 = 466 (rounded down)
				TestUser3ID: 0,
				TestUser4ID: 0,
			},
			expectedLosses: map[int64]int64{
				TestUser3ID: -200,  // Capped at 200
				TestUser4ID: -200,  // Capped at 200
			},
			description: "Losers capped at highest winner bet (200), prize pool = 100+200+200+200 = 700",
		},
		{
			name: "all losers bet less than highest winner",
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "No cap needed test").
					WithOptions("Win", "Lose").
					WithUser(TestUser1ID, "user1", 100000).
					WithUser(TestUser2ID, "user2", 100000).
					WithUser(TestUser3ID, "user3", 100000).
					WithParticipant(TestUser1ID, 0, 1000).  // Winner (high bet)
					WithParticipant(TestUser2ID, 1, 100).   // Loser (low bet)
					WithParticipant(TestUser3ID, 1, 200).   // Loser (low bet)
					Build()
			},
			winningOption: 0,
			expectedPayouts: map[int64]int64{
				TestUser1ID: 1300,  // Gets entire pool
				TestUser2ID: 0,
				TestUser3ID: 0,
			},
			expectedLosses: map[int64]int64{
				TestUser2ID: -100,  // Full loss
				TestUser3ID: -200,  // Full loss
			},
			description: "When losers bet less than highest winner, they lose full amounts",
		},
		{
			name: "equal bets on both sides",
			setupScenario: func() *GroupWagerScenario {
				return NewGroupWagerScenario().
					WithPoolWager(TestResolverID, "Equal bets test").
					WithOptions("A", "B").
					WithUser(TestUser1ID, "user1", 100000).
					WithUser(TestUser2ID, "user2", 100000).
					WithUser(TestUser3ID, "user3", 100000).
					WithUser(TestUser4ID, "user4", 100000).
					WithParticipant(TestUser1ID, 0, 500).   // A
					WithParticipant(TestUser2ID, 0, 500).   // A
					WithParticipant(TestUser3ID, 1, 500).   // B
					WithParticipant(TestUser4ID, 1, 500).   // B
					Build()
			},
			winningOption: 0,
			expectedPayouts: map[int64]int64{
				TestUser1ID: 1000,  // 500/1000 * 2000 = 1000
				TestUser2ID: 1000,  // 500/1000 * 2000 = 1000
				TestUser3ID: 0,
				TestUser4ID: 0,
			},
			expectedLosses: map[int64]int64{
				TestUser3ID: -500,  // Full loss (equal to highest winner)
				TestUser4ID: -500,  // Full loss (equal to highest winner)
			},
			description: "Equal bets work as before with exposure cap",
		},
	}

	for _, tt := range exposureCapTests {
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

			// Setup resolution mocks manually to account for exposure cap logic
			helper.ExpectWagerDetailLookup(TestWagerID, &entities.GroupWagerDetail{
				Wager:        scenario.Wager,
				Options:      scenario.Options,
				Participants: scenario.Participants,
			})

			// User lookups for all participants
			for _, participant := range scenario.Participants {
				if user, exists := scenario.GetUser(participant.DiscordID); exists {
					helper.ExpectUserLookup(participant.DiscordID, user)
				}
			}

			// Calculate expected balance changes based on exposure cap logic
			winners := getWinners(scenario.Participants, winningOptionID)

			// Find max winner bet for exposure cap
			maxWinnerBet := int64(0)
			for _, winner := range winners {
				if winner.Amount > maxWinnerBet {
					maxWinnerBet = winner.Amount
				}
			}

			// Set up winner balance updates
			for userID, expectedPayout := range tt.expectedPayouts {
				if expectedPayout > 0 {
					user, _ := scenario.GetUser(userID)
					participant := getParticipantByUserID(scenario.Participants, userID)
					balanceChange := expectedPayout - participant.Amount
					newBalance := user.Balance + balanceChange
					helper.ExpectBalanceUpdate(userID, newBalance)
					helper.ExpectBalanceHistoryRecordSimple(userID, newBalance, entities.TransactionTypeGroupWagerWin)
					helper.ExpectEventPublish("balance_change")
				}
			}

			// Set up loser balance updates with exposure cap
			for userID, expectedLoss := range tt.expectedLosses {
				user, _ := scenario.GetUser(userID)
				newBalance := user.Balance + expectedLoss
				helper.ExpectBalanceUpdate(userID, newBalance)
				helper.ExpectBalanceHistoryRecordSimple(userID, newBalance, entities.TransactionTypeGroupWagerLoss)
				helper.ExpectEventPublish("balance_change")
			}

			// Other resolution mocks
			mocks.GroupWagerRepo.On("UpdateParticipantPayouts", ctx, mock.Anything).Return(nil)
			mocks.GroupWagerRepo.On("Update", ctx, mock.Anything).Return(nil)
			mocks.EventPublisher.On("Publish", mock.AnythingOfType("events.GroupWagerStateChangeEvent")).Return(nil)

			// Execute
			resolverID := int64(TestResolverID)
			result, err := service.ResolveGroupWager(ctx, TestWagerID, &resolverID, winningOptionID)

			// Assert
			assertions.AssertNoError(err)
			require.NotNil(t, result, "Result should not be nil")

			// Verify payouts
			for userID, expectedPayout := range tt.expectedPayouts {
				assert.Equal(t, expectedPayout, result.PayoutDetails[userID], 
					"User %d payout mismatch. %s", userID, tt.description)
			}

			// Verify actual balance changes by checking the mocks
			// The balance changes should reflect the capped losses
			mocks.AssertAllExpectations(t)
		})
	}
}

func TestGroupWagerService_ResolveGroupWager_ExposureCap_EdgeCases(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
	ctx := context.Background()

	t.Run("no winners scenario with exposure cap", func(t *testing.T) {
		// Setup
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)

		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

		// Create scenario where nobody bet on winning option
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "No winners test").
			WithOptions("A", "B", "C").
			WithUser(TestUser1ID, "user1", 100000).
			WithUser(TestUser2ID, "user2", 100000).
			WithParticipant(TestUser1ID, 0, 1000).  // Bets on A
			WithParticipant(TestUser2ID, 1, 2000).  // Bets on B
			Build()

		// C wins but nobody bet on it
		winningOptionID := scenario.Options[2].ID

		// Setup mocks
		helper.ExpectWagerDetailLookup(TestWagerID, &entities.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})

		// With no winners, everyone loses their full bet
		for _, p := range scenario.Participants {
			if user, exists := scenario.GetUser(p.DiscordID); exists {
				helper.ExpectUserLookup(p.DiscordID, user)
				helper.ExpectBalanceUpdate(p.DiscordID, user.Balance - p.Amount)
				helper.ExpectBalanceHistoryRecordSimple(p.DiscordID, user.Balance - p.Amount, entities.TransactionTypeGroupWagerLoss)
				helper.ExpectEventPublish("balance_change")
			}
		}

		mocks.GroupWagerRepo.On("UpdateParticipantPayouts", ctx, mock.Anything).Return(nil)
		mocks.GroupWagerRepo.On("Update", ctx, mock.Anything).Return(nil)
		helper.ExpectEventPublish("group_wager_state_change")

		// Execute
		resolverID := int64(TestResolverID)
		result, err := service.ResolveGroupWager(ctx, TestWagerID, &resolverID, winningOptionID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Winners, 0, "Should have no winners")
		assert.Len(t, result.Losers, 2, "Should have 2 losers")

		mocks.AssertAllExpectations(t)
	})

	t.Run("house wager not affected by exposure cap", func(t *testing.T) {
		// Setup
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)

		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

		// Create house wager scenario
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "House wager test").
			WithOptions("A", "B").
			WithOdds(2.0, 3.0).
			WithUser(TestUser1ID, "user1", 100000).
			WithUser(TestUser2ID, "user2", 100000).
			WithUser(TestUser3ID, "user3", 100000).
			WithParticipant(TestUser1ID, 0, 10).    // Small bet on A
			WithParticipant(TestUser2ID, 1, 1000).  // Large bet on B
			WithParticipant(TestUser3ID, 1, 2000).  // Large bet on B
			Build()

		winningOptionID := scenario.Options[0].ID // A wins

		// Setup mocks
		setupResolutionMocks(t, helper, mocks, scenario, winningOptionID, entities.GroupWagerTypeHouse)

		// Execute
		resolverID := int64(TestResolverID)
		result, err := service.ResolveGroupWager(ctx, TestWagerID, &resolverID, winningOptionID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		// House wagers should use odds multipliers, not exposure cap
		assert.Equal(t, int64(20), result.PayoutDetails[TestUser1ID], "Winner should get 10 * 2.0 = 20")
		assert.Equal(t, int64(0), result.PayoutDetails[TestUser2ID])
		assert.Equal(t, int64(0), result.PayoutDetails[TestUser3ID])

		mocks.AssertAllExpectations(t)
	})
}