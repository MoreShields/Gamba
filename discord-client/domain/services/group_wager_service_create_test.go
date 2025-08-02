package services

import (
	"testing"

	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerService_CreateGroupWager_TableDriven(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)
	testResolverID := int64(TestResolverID)

	// Validation test cases
	validationTests := []struct {
		name            string
		creatorID       *int64
		condition       string
		options         []string
		votingMinutes   int
		wagerType       entities.GroupWagerType
		oddsMultipliers []float64
		expectedError   string
	}{
		{
			name:          "empty condition",
			creatorID:     &testResolverID,
			condition:     "",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			wagerType:     entities.GroupWagerTypePool,
			expectedError: "condition cannot be empty",
		},
		{
			name:          "insufficient options",
			creatorID:     &testResolverID,
			condition:     "Test",
			options:       []string{"Only One"},
			votingMinutes: 60,
			wagerType:     entities.GroupWagerTypePool,
			expectedError: "must provide at least 2 options",
		},
		{
			name:          "voting period too short",
			creatorID:     &testResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 3, // < 5 minutes
			wagerType:     entities.GroupWagerTypePool,
			expectedError: "voting period must be between 5 minutes",
		},
		{
			name:          "voting period too long",
			creatorID:     &testResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 10081, // > 10080 minutes (7 days)
			wagerType:     entities.GroupWagerTypePool,
			expectedError: "voting period must be between 5 minutes",
		},
		{
			name:          "invalid wager type",
			creatorID:     &testResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			wagerType:     entities.GroupWagerType("invalid"),
			expectedError: "invalid wager type",
		},
		{
			name:            "pool wager with odds",
			creatorID:       &testResolverID,
			condition:       "Test",
			options:         []string{"Yes", "No"},
			votingMinutes:   60,
			wagerType:       entities.GroupWagerTypePool,
			oddsMultipliers: []float64{2.0, 1.5},
			expectedError:   "pool wagers calculate their own odds",
		},
		{
			name:          "house wager without odds",
			creatorID:     &testResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			wagerType:     entities.GroupWagerTypeHouse,
			expectedError: "must provide odds multiplier for each option",
		},
		{
			name:            "house wager odds count mismatch",
			creatorID:       &testResolverID,
			condition:       "Test",
			options:         []string{"A", "B", "C"},
			votingMinutes:   60,
			wagerType:       entities.GroupWagerTypeHouse,
			oddsMultipliers: []float64{2.0, 1.5}, // Only 2 odds for 3 options
			expectedError:   "must provide odds multiplier for each option",
		},
		{
			name:            "house wager negative odds",
			creatorID:       &testResolverID,
			condition:       "Test",
			options:         []string{"Yes", "No"},
			votingMinutes:   60,
			wagerType:       entities.GroupWagerTypeHouse,
			oddsMultipliers: []float64{2.0, -1.5},
			expectedError:   "odds multiplier for option 2 must be positive",
		},
		{
			name:            "house wager zero odds",
			creatorID:       &testResolverID,
			condition:       "Test",
			options:         []string{"Yes", "No"},
			votingMinutes:   60,
			wagerType:       entities.GroupWagerTypeHouse,
			oddsMultipliers: []float64{2.0, 0},
			expectedError:   "odds multiplier for option 2 must be positive",
		},
		{
			name:          "duplicate options case insensitive",
			creatorID:     &testResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No", "yes"}, // duplicate
			votingMinutes: 60,
			wagerType:     entities.GroupWagerTypePool,
			expectedError: "duplicate option found",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset fixture for each test
			fixture.Reset()

			// Execute - validation should fail before any repository calls
			result, err := fixture.Service.CreateGroupWager(
				fixture.Ctx,
				tt.creatorID,
				tt.condition,
				tt.options,
				tt.votingMinutes,
				TestMessageID,
				TestChannelID,
				tt.wagerType,
				tt.oddsMultipliers,
			)

			// Assert
			fixture.Assertions.AssertValidationError(err, tt.expectedError)
			if result != nil {
				fixture.T.Errorf("expected nil result, got %v", result)
			}

			// No repository calls should have been made
			fixture.AssertAllMocks()
		})
	}
}

func TestGroupWagerService_CreateGroupWager_Success(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)
	testResolverID := int64(TestResolverID)

	successTests := []struct {
		name            string
		wagerType       entities.GroupWagerType
		condition       string
		options         []string
		oddsMultipliers []float64
		votingMinutes   int
		validateOptions func(*testing.T, []*entities.GroupWagerOption)
	}{
		{
			name:          "pool wager with 2 options",
			wagerType:     entities.GroupWagerTypePool,
			condition:     "Will it rain tomorrow?",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			validateOptions: func(t *testing.T, options []*entities.GroupWagerOption) {
				require.Len(t, options, 2)
				for _, opt := range options {
					assert.Equal(t, float64(0), opt.OddsMultiplier)
					assert.Equal(t, int64(0), opt.TotalAmount)
				}
			},
		},
		{
			name:            "house wager with 2 options",
			wagerType:       entities.GroupWagerTypeHouse,
			condition:       "Who will win the match?",
			options:         []string{"Team A", "Team B"},
			oddsMultipliers: []float64{1.8, 2.2},
			votingMinutes:   120,
			validateOptions: func(t *testing.T, options []*entities.GroupWagerOption) {
				require.Len(t, options, 2)
				assert.Equal(t, 1.8, options[0].OddsMultiplier)
				assert.Equal(t, 2.2, options[1].OddsMultiplier)
			},
		},
		{
			name:          "pool wager with many options",
			wagerType:     entities.GroupWagerTypePool,
			condition:     "What will be the high temperature?",
			options:       []string{"< 60F", "60-70F", "70-80F", "80-90F", "> 90F"},
			votingMinutes: 1440, // 24 hours
			validateOptions: func(t *testing.T, options []*entities.GroupWagerOption) {
				require.Len(t, options, 5)
				for i, opt := range options {
					assert.Equal(t, int16(i), opt.OptionOrder)
				}
			},
		},
		{
			name:            "house wager with uneven odds",
			wagerType:       entities.GroupWagerTypeHouse,
			condition:       "Tournament winner?",
			options:         []string{"Favorite", "Dark Horse", "Underdog"},
			oddsMultipliers: []float64{1.5, 3.0, 10.0},
			votingMinutes:   10080, // Maximum 7 days
			validateOptions: func(t *testing.T, options []*entities.GroupWagerOption) {
				require.Len(t, options, 3)
				assert.Equal(t, 1.5, options[0].OddsMultiplier)
				assert.Equal(t, 3.0, options[1].OddsMultiplier)
				assert.Equal(t, 10.0, options[2].OddsMultiplier)
			},
		},
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset fixture for each test
			fixture.Reset()

			// Setup creator user
			creator := &entities.User{
				DiscordID: TestResolverID,
				Username:  "creator",
				Balance:   TestInitialBalance,
			}
			fixture.Helper.ExpectUserLookup(TestResolverID, creator)

			// Setup create expectations
			expectedMinParticipants := 3 // Default for pool wagers
			if tt.wagerType == entities.GroupWagerTypeHouse {
				expectedMinParticipants = 0 // House wagers don't require minimum participants
			}

			fixture.Mocks.GroupWagerRepo.On("CreateWithOptions", fixture.Ctx,
				mock.MatchedBy(func(gw *entities.GroupWager) bool {
					return gw.CreatorDiscordID != nil && *gw.CreatorDiscordID == TestResolverID &&
						gw.Condition == tt.condition &&
						gw.State == entities.GroupWagerStateActive &&
						gw.WagerType == tt.wagerType &&
						gw.TotalPot == 0 &&
						gw.MinParticipants == expectedMinParticipants &&
						gw.VotingPeriodMinutes == tt.votingMinutes &&
						gw.MessageID == TestMessageID &&
						gw.ChannelID == TestChannelID
				}),
				mock.MatchedBy(func(opts []*entities.GroupWagerOption) bool {
					if len(opts) != len(tt.options) {
						return false
					}
					for i, opt := range opts {
						if opt.OptionText != tt.options[i] ||
							opt.OptionOrder != int16(i) ||
							opt.TotalAmount != 0 {
							return false
						}
						// Check odds based on wager type
						if tt.wagerType == entities.GroupWagerTypeHouse {
							if opt.OddsMultiplier != tt.oddsMultipliers[i] {
								return false
							}
						} else {
							if opt.OddsMultiplier != 0 {
								return false
							}
						}
					}
					return true
				}),
			).Return(nil)

			// Execute
			result, err := fixture.Service.CreateGroupWager(
				fixture.Ctx,
				&testResolverID,
				tt.condition,
				tt.options,
				tt.votingMinutes,
				TestMessageID,
				TestChannelID,
				tt.wagerType,
				tt.oddsMultipliers,
			)

			// Assert
			fixture.Assertions.AssertNoError(err)
			fixture.Assertions.AssertGroupWagerCreated(result, tt.wagerType, len(tt.options))
			tt.validateOptions(fixture.T, result.Options)

			fixture.AssertAllMocks()
		})
	}
}

func TestGroupWagerService_CreateGroupWager_UserNotFound(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)
	testResolverID := int64(TestResolverID)

	// Setup - user not found
	fixture.Helper.ExpectUserNotFound(TestResolverID)

	// Execute
	result, err := fixture.Service.CreateGroupWager(
		fixture.Ctx,
		&testResolverID,
		"Test condition",
		[]string{"Yes", "No"},
		60,
		TestMessageID,
		TestChannelID,
		entities.GroupWagerTypePool,
		nil,
	)

	// Assert
	fixture.Assertions.AssertValidationError(err, "creator 999999 not found")
	if result != nil {
		fixture.T.Errorf("expected nil result, got %v", result)
	}

	fixture.AssertAllMocks()
}

func TestGroupWagerService_CreateGroupWager_SystemUser(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	// Setup - system user (ID 0) should skip creator validation
	// Note: No user lookup expectation needed for system user
	fixture.Mocks.GroupWagerRepo.On("CreateWithOptions", fixture.Ctx,
		mock.MatchedBy(func(gw *entities.GroupWager) bool {
			return gw.CreatorDiscordID == nil && // System user
				gw.Condition == "Test condition" &&
				gw.State == entities.GroupWagerStateActive &&
				gw.WagerType == entities.GroupWagerTypeHouse &&
				gw.TotalPot == 0 &&
				gw.MinParticipants == 0 && // House wagers don't require minimum participants
				gw.VotingPeriodMinutes == 60 &&
				gw.MessageID == TestMessageID &&
				gw.ChannelID == TestChannelID
		}),
		mock.MatchedBy(func(opts []*entities.GroupWagerOption) bool {
			return len(opts) == 2 &&
				opts[0].OptionText == "Yes" &&
				opts[0].OptionOrder == 0 &&
				opts[0].OddsMultiplier == 1.5 &&
				opts[1].OptionText == "No" &&
				opts[1].OptionOrder == 1 &&
				opts[1].OddsMultiplier == 2.0
		}),
	).Return(nil)

	// Execute
	result, err := fixture.Service.CreateGroupWager(
		fixture.Ctx,
		nil, // System user - no specific creator
		"Test condition",
		[]string{"Yes", "No"},
		60,
		TestMessageID,
		TestChannelID,
		entities.GroupWagerTypeHouse,
		[]float64{1.5, 2.0},
	)

	// Assert
	fixture.Assertions.AssertNoError(err)
	require.NotNil(fixture.T, result)
	assert.Nil(fixture.T, result.Wager.CreatorDiscordID)
	assert.Equal(fixture.T, "Test condition", result.Wager.Condition)
	assert.Equal(fixture.T, entities.GroupWagerTypeHouse, result.Wager.WagerType)
	assert.Len(fixture.T, result.Options, 2)

	fixture.AssertAllMocks()
}
