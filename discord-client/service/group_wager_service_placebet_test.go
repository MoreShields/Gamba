package service

import (
	"testing"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// findParticipantInScenario finds an existing participant for a user in the scenario
func findParticipantInScenario(participants []*models.GroupWagerParticipant, userID int64) *models.GroupWagerParticipant {
	for _, p := range participants {
		if p.DiscordID == userID {
			return p
		}
	}
	return nil
}

func TestGroupWagerService_PlaceBet_TableDriven(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	// Test cases that apply to both pool and house wagers
	testCases := []struct {
		name      string
		wagerType models.GroupWagerType
		odds      []float64
		setupFunc func(*GroupWagerScenarioBuilder) *GroupWagerScenario
		betAmount int64
		betOption int // index of option to bet on
		validate  func(*testing.T, *models.GroupWagerParticipant, error)
	}{
		{
			name:      "successful first bet - pool wager",
			wagerType: models.GroupWagerTypePool,
			odds:      nil,
			setupFunc: func(b *GroupWagerScenarioBuilder) *GroupWagerScenario {
				return b.
					WithPoolWager(TestResolverID, "Test condition").
					WithOptions("Option 1", "Option 2").
					WithUser(TestUser1ID, "user1", TestInitialBalance).
					Build()
			},
			betAmount: 1000,
			betOption: 0,
			validate: func(t *testing.T, participant *models.GroupWagerParticipant, err error) {
				require.NoError(t, err)
				require.NotNil(t, participant)
				assert.Equal(t, int64(1000), participant.Amount)
				assert.Equal(t, int64(TestOption1ID), participant.OptionID)
			},
		},
		{
			name:      "successful first bet - house wager",
			wagerType: models.GroupWagerTypeHouse,
			odds:      []float64{2.5, 1.8},
			setupFunc: func(b *GroupWagerScenarioBuilder) *GroupWagerScenario {
				return b.
					WithHouseWager(TestResolverID, "Test condition").
					WithOptions("Option 1", "Option 2").
					WithOdds(2.5, 1.8).
					WithUser(TestUser1ID, "user1", TestInitialBalance).
					Build()
			},
			betAmount: 1000,
			betOption: 0,
			validate: func(t *testing.T, participant *models.GroupWagerParticipant, err error) {
				require.NoError(t, err)
				require.NotNil(t, participant)
				assert.Equal(t, int64(1000), participant.Amount)
				assert.Equal(t, int64(TestOption1ID), participant.OptionID)
			},
		},
		{
			name:      "increase existing bet same option",
			wagerType: models.GroupWagerTypePool,
			odds:      nil,
			setupFunc: func(b *GroupWagerScenarioBuilder) *GroupWagerScenario {
				return b.
					WithPoolWager(TestResolverID, "Test condition").
					WithOptions("Option 1", "Option 2").
					WithUser(TestUser1ID, "user1", TestInitialBalance).
					WithParticipant(TestUser1ID, 0, 1000). // Existing bet
					Build()
			},
			betAmount: 3000, // Increase to 3000
			betOption: 0,
			validate: func(t *testing.T, participant *models.GroupWagerParticipant, err error) {
				require.NoError(t, err)
				require.NotNil(t, participant)
				assert.Equal(t, int64(3000), participant.Amount)
				assert.Equal(t, int64(TestOption1ID), participant.OptionID)
			},
		},
		{
			name:      "change to different option",
			wagerType: models.GroupWagerTypeHouse,
			odds:      []float64{2.5, 1.8},
			setupFunc: func(b *GroupWagerScenarioBuilder) *GroupWagerScenario {
				return b.
					WithHouseWager(TestResolverID, "Test condition").
					WithOptions("Option 1", "Option 2").
					WithOdds(2.5, 1.8).
					WithUser(TestUser1ID, "user1", TestInitialBalance).
					WithParticipant(TestUser1ID, 0, 1000). // Existing bet on option 0
					Build()
			},
			betAmount: 2000,
			betOption: 1, // Change to option 1
			validate: func(t *testing.T, participant *models.GroupWagerParticipant, err error) {
				require.NoError(t, err)
				require.NotNil(t, participant)
				assert.Equal(t, int64(2000), participant.Amount)
				assert.Equal(t, int64(TestOption2ID), participant.OptionID)
			},
		},
		{
			name:      "insufficient balance",
			wagerType: models.GroupWagerTypePool,
			odds:      nil,
			setupFunc: func(b *GroupWagerScenarioBuilder) *GroupWagerScenario {
				return b.
					WithPoolWager(TestResolverID, "Test condition").
					WithOptions("Option 1", "Option 2").
					WithUser(TestUser1ID, "user1", 1000). // Only 1000 balance
					Build()
			},
			betAmount: 10000, // Trying to bet 10000
			betOption: 0,
			validate: func(t *testing.T, participant *models.GroupWagerParticipant, err error) {
				require.Error(t, err)
				assert.Nil(t, participant)
				assert.Contains(t, err.Error(), "insufficient balance")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset fixture for each test
			fixture.Reset()

			// Build scenario
			scenario := NewGroupWagerScenario()
			fullScenario := tc.setupFunc(scenario)

			// Setup common mocks
			// Create detail from scenario
			detail := &models.GroupWagerDetail{
				Wager:        fullScenario.Wager,
				Options:      fullScenario.Options,
				Participants: fullScenario.Participants,
			}
			fixture.Helper.ExpectWagerDetailLookup(TestWagerID, detail)
			fixture.Helper.ExpectWagerDetailLookup(TestWagerID, detail)

			// Setup user mock if user exists in scenario
			if user, exists := fullScenario.GetUser(TestUser1ID); exists {
				fixture.Helper.ExpectUserLookup(TestUser1ID, user)
			}

			// Setup participant lookup mock - needed for all test cases
			existingParticipant := findParticipantInScenario(fullScenario.Participants, TestUser1ID)
			fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, existingParticipant)

			// For successful cases, setup additional mocks
			if tc.name != "insufficient balance" {
				
				if existingParticipant != nil {
					// For existing participants, expect participant update
					fixture.Mocks.GroupWagerRepo.On("SaveParticipant", fixture.Ctx, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
						return p.GroupWagerID == TestWagerID &&
							p.DiscordID == TestUser1ID &&
							p.OptionID == fullScenario.Options[tc.betOption].ID &&
							p.Amount == tc.betAmount
					})).Return(nil)
					
					// Expect option total updates for both old and new options
					if existingParticipant.OptionID != fullScenario.Options[tc.betOption].ID {
						// Different option - update old option to 0, new option to bet amount
						fixture.Helper.ExpectOptionTotalUpdate(existingParticipant.OptionID, 0)
						fixture.Helper.ExpectOptionTotalUpdate(fullScenario.Options[tc.betOption].ID, tc.betAmount)
					} else {
						// Same option - update to new total
						fixture.Helper.ExpectOptionTotalUpdate(fullScenario.Options[tc.betOption].ID, tc.betAmount)
					}
				} else {
					// For new participants
					fixture.Helper.ExpectNewParticipant(TestWagerID, TestUser1ID, fullScenario.Options[tc.betOption].ID, tc.betAmount)
					fixture.Helper.ExpectOptionTotalUpdate(fullScenario.Options[tc.betOption].ID, tc.betAmount)
				}
				
				// Expect wager update
				fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
					return gw.ID == TestWagerID
				})).Return(nil)
				
				// For pool wagers, expect odds recalculation
				if tc.wagerType == models.GroupWagerTypePool {
					fixture.Mocks.GroupWagerRepo.On("UpdateAllOptionOdds", fixture.Ctx, int64(TestWagerID), mock.AnythingOfType("map[int64]float64")).Return(nil)
				}
			}

			// Execute
			participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, fullScenario.Options[tc.betOption].ID, tc.betAmount)

			// Validate
			tc.validate(t, participant, err)
			
			// Only assert expectations if we don't expect errors
			// (error cases may not call all mocked methods)
			if err == nil {
				fixture.AssertAllMocks()
			}
		})
	}
}

func TestGroupWagerService_PlaceBet_CompleteFlow(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	t.Run("pool wager - complete bet flow with odds recalculation", func(t *testing.T) {
		// Reset fixture for this test
		fixture.Reset()

		// Build scenario
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test pool wager").
			WithOptions("Yes", "No").
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		// Setup mocks for successful bet
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		user1, _ := scenario.GetUser(TestUser1ID)
		fixture.Helper.ExpectUserLookup(TestUser1ID, user1)
		fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil) // No existing participant
		fixture.Helper.ExpectNewParticipant(TestWagerID, TestUser1ID, TestOption1ID, int64(1000))
		fixture.Helper.ExpectOptionTotalUpdate(TestOption1ID, 1000)
		
		// Expect wager pot update
		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == TestWagerID && gw.TotalPot == 1000
		})).Return(nil)
		
		// Expect odds recalculation for pool wager
		fixture.Mocks.GroupWagerRepo.On("UpdateAllOptionOdds", fixture.Ctx, int64(TestWagerID), mock.MatchedBy(func(odds map[int64]float64) bool {
			// Option 1 should have odds of 1.0 (1000/1000)
			// Option 2 should have odds of 0 (no bets)
			return odds[TestOption1ID] == 1.0 && odds[TestOption2ID] == 0
		})).Return(nil)

		// Execute
		participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)
		assert.Equal(t, int64(TestOption1ID), participant.OptionID)
		assert.Equal(t, int64(1000), participant.Amount)

		fixture.AssertAllMocks()
	})

	t.Run("house wager - complete bet flow without odds recalculation", func(t *testing.T) {
		// Reset fixture for this test
		fixture.Reset()

		// Build scenario
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "Test house wager").
			WithOptions("Team A", "Team B").
			WithOdds(2.5, 1.8).
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		// Setup mocks for successful bet
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		user1, _ := scenario.GetUser(TestUser1ID)
		fixture.Helper.ExpectUserLookup(TestUser1ID, user1)
		fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil) // No existing participant
		fixture.Helper.ExpectNewParticipant(TestWagerID, TestUser1ID, TestOption1ID, int64(1000))
		fixture.Helper.ExpectOptionTotalUpdate(TestOption1ID, 1000)
		
		// Expect wager pot update
		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == TestWagerID && gw.TotalPot == 1000
		})).Return(nil)
		
		// House wagers should NOT trigger odds recalculation
		// No expectation for UpdateAllOptionOdds

		// Execute
		participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)
		assert.Equal(t, int64(TestOption1ID), participant.OptionID)
		assert.Equal(t, int64(1000), participant.Amount)

		fixture.AssertAllMocks()
	})
}

func TestGroupWagerService_PlaceBet_EdgeCases(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	t.Run("bet on non-existent option", func(t *testing.T) {
		// Reset fixture for this test
		fixture.Reset()

		// Build scenario
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test wager").
			WithOptions("Yes", "No").
			Build()

		// Setup mocks
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})

		// Execute with non-existent option ID
		participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, 99999, 1000)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "invalid option")

		fixture.AssertAllMocks()
	})

	t.Run("negative bet amount", func(t *testing.T) {
		// Reset fixture for this test
		fixture.Reset()

		// Execute - should fail immediately without any repository calls
		participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, -1000)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "bet amount must be positive")

		fixture.AssertAllMocks()
	})

	t.Run("zero bet amount", func(t *testing.T) {
		// Reset fixture for this test
		fixture.Reset()

		// Execute - should fail immediately without any repository calls
		participant, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 0)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "bet amount must be positive")

		fixture.AssertAllMocks()
	})
}