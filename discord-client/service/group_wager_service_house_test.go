package service

import (
	"context"
	"testing"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGroupWagerService_HouseWager_SpecificScenarios tests scenarios specific to house wagers
func TestGroupWagerService_HouseWager_SpecificScenarios(t *testing.T) {
	ctx := context.Background()

	t.Run("house wager odds remain fixed after bets", func(t *testing.T) {
		// Setup
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)
		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)

		// Build scenario with fixed odds
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "Fixed odds test").
			WithOptions("Team A", "Team B").
			WithOdds(2.5, 1.8).
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		// Setup mocks for bet
		helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])
		helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil)
		helper.ExpectNewParticipant(TestWagerID, TestUser1ID, TestOption1ID, 1000)
		helper.ExpectOptionTotalUpdate(TestOption1ID, 1000)
		
		mocks.GroupWagerRepo.On("Update", ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == TestWagerID && gw.TotalPot == 1000
		})).Return(nil)
		
		// House wagers should NOT update odds
		// No call to UpdateAllOptionOdds expected

		// Execute
		participant, err := service.PlaceBet(ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)
		
		// Odds should remain unchanged
		assert.Equal(t, 2.5, scenario.Options[0].OddsMultiplier)
		assert.Equal(t, 1.8, scenario.Options[1].OddsMultiplier)

		mocks.AssertAllExpectations(t)
	})

	t.Run("house wager payout calculation with various odds", func(t *testing.T) {
		oddsTests := []struct {
			name         string
			odds         []float64
			bets         []struct{ userID int64; optionIndex int; amount int64 }
			winningIndex int
			expected     map[int64]int64 // userID -> expected payout
		}{
			{
				name: "even odds",
				odds: []float64{2.0, 2.0},
				bets: []struct{ userID int64; optionIndex int; amount int64 }{
					{TestUser1ID, 0, 1000},
					{TestUser2ID, 1, 1500},
					{TestUser3ID, 0, 500}, // Third participant to meet minimum
				},
				winningIndex: 0,
				expected: map[int64]int64{
					TestUser1ID: 2000, // 1000 * 2.0
					TestUser2ID: 0,
					TestUser3ID: 1000, // 500 * 2.0
				},
			},
			{
				name: "favorite vs underdog",
				odds: []float64{1.2, 5.0}, // Heavy favorite vs big underdog
				bets: []struct{ userID int64; optionIndex int; amount int64 }{
					{TestUser1ID, 0, 5000}, // Bet on favorite
					{TestUser2ID, 1, 1000}, // Bet on underdog
					{TestUser3ID, 1, 500},  // Also on underdog
				},
				winningIndex: 1, // Underdog wins
				expected: map[int64]int64{
					TestUser1ID: 0,
					TestUser2ID: 5000, // 1000 * 5.0
					TestUser3ID: 2500, // 500 * 5.0
				},
			},
			{
				name: "fractional odds",
				odds: []float64{1.75, 2.33, 4.5},
				bets: []struct{ userID int64; optionIndex int; amount int64 }{
					{TestUser1ID, 0, 1000},
					{TestUser2ID, 1, 1000},
					{TestUser3ID, 2, 1000},
				},
				winningIndex: 2,
				expected: map[int64]int64{
					TestUser1ID: 0,
					TestUser2ID: 0,
					TestUser3ID: 4500, // 1000 * 4.5
				},
			},
		}

		for _, tt := range oddsTests {
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

				// Build scenario
				builder := NewGroupWagerScenario().
					WithHouseWager(TestResolverID, "Odds test")
				
				// Add options based on odds count
				optionTexts := make([]string, len(tt.odds))
				for i := range optionTexts {
					optionTexts[i] = string(rune('A' + i))
				}
				builder.WithOptions(optionTexts...).WithOdds(tt.odds...)
				
				// Add users and bets
				for _, bet := range tt.bets {
					builder.WithUser(bet.userID, "user", TestInitialBalance)
					builder.WithParticipant(bet.userID, bet.optionIndex, bet.amount)
				}
				
				scenario := builder.Build()
				winningOptionID := scenario.Options[tt.winningIndex].ID
				winningOptionText := scenario.Options[tt.winningIndex].OptionText

				// Setup mocks for resolution
				setupResolutionMocks(t, helper, mocks, scenario, winningOptionID, models.GroupWagerTypeHouse)

				// Execute
				result, err := service.ResolveGroupWager(ctx, TestWagerID, TestResolverID, winningOptionText)

				// Assert
				assertions.AssertNoError(err)
				assertions.AssertPayouts(result, tt.expected)
				assertions.AssertHouseWagerPayouts(result, scenario.Options[tt.winningIndex])

				mocks.AssertAllExpectations(t)
			})
		}
	})

	t.Run("house wager balance deduction timing", func(t *testing.T) {
		// For house wagers, balance is only changed at resolution (not when bet is placed)
		// This test verifies that winners get their net win and losers have their bets deducted at resolution
		
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)
		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

		// Build scenario
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "Balance timing test").
			WithOptions("Win", "Lose").
			WithOdds(2.0, 2.0).
			WithUser(TestUser1ID, "winner", TestInitialBalance).
			WithUser(TestUser2ID, "loser", TestInitialBalance).
			WithUser(TestUser3ID, "loser2", TestInitialBalance).
			WithParticipant(TestUser1ID, 0, 2000).
			WithParticipant(TestUser2ID, 1, 2000).
			WithParticipant(TestUser3ID, 1, 1000). // Third participant
			Build()

		// Setup mocks
		helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])
		helper.ExpectUserLookup(TestUser2ID, scenario.Users[TestUser2ID])
		helper.ExpectUserLookup(TestUser3ID, scenario.Users[TestUser3ID])
		
		// Winner gets net win (payout - bet)
		helper.ExpectBalanceUpdate(TestUser1ID, TestInitialBalance+2000) // 10000 + (4000-2000) = 12000
		helper.ExpectBalanceHistoryRecord(TestUser1ID, 2000, models.TransactionTypeGroupWagerWin)
		helper.ExpectEventPublish("events.BalanceChangeEvent")
		
		// Losers have bet amounts deducted
		helper.ExpectBalanceUpdate(TestUser2ID, TestInitialBalance-2000) // 10000 - 2000 = 8000
		helper.ExpectBalanceHistoryRecord(TestUser2ID, -2000, models.TransactionTypeGroupWagerLoss)
		helper.ExpectEventPublish("events.BalanceChangeEvent")
		
		helper.ExpectBalanceUpdate(TestUser3ID, TestInitialBalance-1000) // 10000 - 1000 = 9000
		helper.ExpectBalanceHistoryRecord(TestUser3ID, -1000, models.TransactionTypeGroupWagerLoss)
		helper.ExpectEventPublish("events.BalanceChangeEvent")
		
		// Other resolution mocks
		mocks.GroupWagerRepo.On("UpdateParticipantPayouts", ctx, mock.Anything).Return(nil)
		mocks.GroupWagerRepo.On("Update", ctx, mock.Anything).Return(nil)
		mocks.EventPublisher.On("Publish", mock.AnythingOfType("events.GroupWagerStateChangeEvent")).Return()

		// Execute
		result, err := service.ResolveGroupWager(ctx, TestWagerID, TestResolverID, scenario.Options[0].OptionText)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, result)
		
		// Winner should have payout
		assert.Len(t, result.Winners, 1)
		assert.Equal(t, int64(4000), *result.Winners[0].PayoutAmount)
		
		// Losers should have 0 payout
		assert.Len(t, result.Losers, 2)
		assert.Equal(t, int64(0), *result.Losers[0].PayoutAmount)
		assert.Equal(t, int64(0), *result.Losers[1].PayoutAmount)

		mocks.AssertAllExpectations(t)
	})
}

func TestGroupWagerService_HouseWager_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("house wager with extreme odds", func(t *testing.T) {
		// Test with very high and very low odds
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)
		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)

		creator := &models.User{
			DiscordID: TestResolverID,
			Username:  "creator",
			Balance:   TestInitialBalance,
		}
		helper.ExpectUserLookup(TestResolverID, creator)

		// Very high odds (100x) and very low odds (1.01x)
		mocks.GroupWagerRepo.On("CreateWithOptions", ctx, mock.Anything, mock.MatchedBy(func(opts []*models.GroupWagerOption) bool {
			return len(opts) == 2 &&
				opts[0].OddsMultiplier == 100.0 &&
				opts[1].OddsMultiplier == 1.01
		})).Return(nil)

		result, err := service.CreateGroupWager(
			ctx,
			TestResolverID,
			"Extreme odds test",
			[]string{"Longshot", "Sure Thing"},
			60,
			TestMessageID,
			TestChannelID,
			models.GroupWagerTypeHouse,
			[]float64{100.0, 1.01},
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 100.0, result.Options[0].OddsMultiplier)
		assert.Equal(t, 1.01, result.Options[1].OddsMultiplier)

		mocks.AssertAllExpectations(t)
	})

	t.Run("house wager all bets on losing option", func(t *testing.T) {
		// When all participants bet on the losing option, house keeps everything
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
		winningOptionText := scenario.Options[2].OptionText

		// Setup mocks - no balance changes for losers
		helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		
		// User lookups
		for _, p := range scenario.Participants {
			helper.ExpectUserLookup(p.DiscordID, scenario.Users[p.DiscordID])
		}
		
		// All participants lose their bets
		helper.ExpectBalanceUpdate(TestUser1ID, TestInitialBalance-1000) // 10000 - 1000 = 9000
		helper.ExpectBalanceHistoryRecord(TestUser1ID, -1000, models.TransactionTypeGroupWagerLoss)
		helper.ExpectEventPublish("events.BalanceChangeEvent")
		
		helper.ExpectBalanceUpdate(TestUser2ID, TestInitialBalance-2000) // 10000 - 2000 = 8000
		helper.ExpectBalanceHistoryRecord(TestUser2ID, -2000, models.TransactionTypeGroupWagerLoss)
		helper.ExpectEventPublish("events.BalanceChangeEvent")
		
		helper.ExpectBalanceUpdate(TestUser3ID, TestInitialBalance-3000) // 10000 - 3000 = 7000
		helper.ExpectBalanceHistoryRecord(TestUser3ID, -3000, models.TransactionTypeGroupWagerLoss)
		helper.ExpectEventPublish("events.BalanceChangeEvent")
		
		// Other resolution mocks
		mocks.GroupWagerRepo.On("UpdateParticipantPayouts", ctx, mock.Anything).Return(nil)
		mocks.GroupWagerRepo.On("Update", ctx, mock.Anything).Return(nil)
		mocks.EventPublisher.On("Publish", mock.AnythingOfType("events.GroupWagerStateChangeEvent")).Return()

		// Execute
		result, err := service.ResolveGroupWager(ctx, TestWagerID, TestResolverID, winningOptionText)

		// Verify
		assertions.AssertNoError(err)
		assertions.AssertWagerResolved(result, 0, 3) // 0 winners, 3 losers
		
		// All payouts should be 0
		for _, loser := range result.Losers {
			assert.Equal(t, int64(0), *loser.PayoutAmount)
		}

		mocks.AssertAllExpectations(t)
	})

	t.Run("house wager rounding behavior", func(t *testing.T) {
		// Test how fractional payouts are handled
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)
		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{TestResolverID}

		// Odds that will create fractional payouts
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "Rounding test").
			WithOptions("A", "B").
			WithOdds(1.33, 2.67). // Will create fractional results
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			WithUser(TestUser2ID, "user2", TestInitialBalance).
			WithUser(TestUser3ID, "user3", TestInitialBalance).
			WithParticipant(TestUser1ID, 0, 100). // 100 * 1.33 = 133
			WithParticipant(TestUser2ID, 1, 100). // 100 * 2.67 = 267
			WithParticipant(TestUser3ID, 0, 50). // Third participant
			Build()

		// Test option A winning
		setupResolutionMocks(t, helper, mocks, scenario, scenario.Options[0].ID, models.GroupWagerTypeHouse)

		result, err := service.ResolveGroupWager(ctx, TestWagerID, TestResolverID, scenario.Options[0].OptionText)

		require.NoError(t, err)
		
		// Verify rounding behavior (should truncate to integer)
		winner := result.Winners[0]
		assert.Equal(t, int64(133), *winner.PayoutAmount) // 100 * 1.33 = 133

		mocks.AssertAllExpectations(t)
	})
}