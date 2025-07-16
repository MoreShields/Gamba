package service

import (
	"context"
	"testing"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerService_CreateGroupWager_TableDriven(t *testing.T) {
	ctx := context.Background()

	// Validation test cases
	validationTests := []struct {
		name            string
		creatorID       int64
		condition       string
		options         []string
		votingMinutes   int
		wagerType       models.GroupWagerType
		oddsMultipliers []float64
		expectedError   string
	}{
		{
			name:          "empty condition",
			creatorID:     TestResolverID,
			condition:     "",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			wagerType:     models.GroupWagerTypePool,
			expectedError: "condition cannot be empty",
		},
		{
			name:          "insufficient options",
			creatorID:     TestResolverID,
			condition:     "Test",
			options:       []string{"Only One"},
			votingMinutes: 60,
			wagerType:     models.GroupWagerTypePool,
			expectedError: "must provide at least 2 options",
		},
		{
			name:          "voting period too short",
			creatorID:     TestResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 3, // < 5 minutes
			wagerType:     models.GroupWagerTypePool,
			expectedError: "voting period must be between 5 minutes",
		},
		{
			name:          "voting period too long",
			creatorID:     TestResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 10081, // > 10080 minutes (7 days)
			wagerType:     models.GroupWagerTypePool,
			expectedError: "voting period must be between 5 minutes",
		},
		{
			name:          "invalid wager type",
			creatorID:     TestResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			wagerType:     models.GroupWagerType("invalid"),
			expectedError: "invalid wager type",
		},
		{
			name:            "pool wager with odds",
			creatorID:       TestResolverID,
			condition:       "Test",
			options:         []string{"Yes", "No"},
			votingMinutes:   60,
			wagerType:       models.GroupWagerTypePool,
			oddsMultipliers: []float64{2.0, 1.5},
			expectedError:   "pool wagers calculate their own odds",
		},
		{
			name:          "house wager without odds",
			creatorID:     TestResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			wagerType:     models.GroupWagerTypeHouse,
			expectedError: "must provide odds multiplier for each option",
		},
		{
			name:            "house wager odds count mismatch",
			creatorID:       TestResolverID,
			condition:       "Test",
			options:         []string{"A", "B", "C"},
			votingMinutes:   60,
			wagerType:       models.GroupWagerTypeHouse,
			oddsMultipliers: []float64{2.0, 1.5}, // Only 2 odds for 3 options
			expectedError:   "must provide odds multiplier for each option",
		},
		{
			name:            "house wager negative odds",
			creatorID:       TestResolverID,
			condition:       "Test",
			options:         []string{"Yes", "No"},
			votingMinutes:   60,
			wagerType:       models.GroupWagerTypeHouse,
			oddsMultipliers: []float64{2.0, -1.5},
			expectedError:   "odds multiplier for option 2 must be positive",
		},
		{
			name:            "house wager zero odds",
			creatorID:       TestResolverID,
			condition:       "Test",
			options:         []string{"Yes", "No"},
			votingMinutes:   60,
			wagerType:       models.GroupWagerTypeHouse,
			oddsMultipliers: []float64{2.0, 0},
			expectedError:   "odds multiplier for option 2 must be positive",
		},
		{
			name:          "duplicate options case insensitive",
			creatorID:     TestResolverID,
			condition:     "Test",
			options:       []string{"Yes", "No", "yes"}, // duplicate
			votingMinutes: 60,
			wagerType:     models.GroupWagerTypePool,
			expectedError: "duplicate option found",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mocks := NewTestMocks()
			service := NewGroupWagerService(
				mocks.GroupWagerRepo,
				mocks.UserRepo,
				mocks.BalanceHistoryRepo,
				mocks.EventPublisher,
			)

			// Execute - validation should fail before any repository calls
			result, err := service.CreateGroupWager(
				ctx,
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
			require.Error(t, err)
			require.Nil(t, result)
			require.Contains(t, err.Error(), tt.expectedError)

			// No repository calls should have been made
			mocks.AssertAllExpectations(t)
		})
	}
}

func TestGroupWagerService_CreateGroupWager_Success(t *testing.T) {
	ctx := context.Background()

	successTests := []struct {
		name            string
		wagerType       models.GroupWagerType
		condition       string
		options         []string
		oddsMultipliers []float64
		votingMinutes   int
		validateOptions func(*testing.T, []*models.GroupWagerOption)
	}{
		{
			name:          "pool wager with 2 options",
			wagerType:     models.GroupWagerTypePool,
			condition:     "Will it rain tomorrow?",
			options:       []string{"Yes", "No"},
			votingMinutes: 60,
			validateOptions: func(t *testing.T, options []*models.GroupWagerOption) {
				require.Len(t, options, 2)
				for _, opt := range options {
					assert.Equal(t, float64(0), opt.OddsMultiplier)
					assert.Equal(t, int64(0), opt.TotalAmount)
				}
			},
		},
		{
			name:            "house wager with 2 options",
			wagerType:       models.GroupWagerTypeHouse,
			condition:       "Who will win the match?",
			options:         []string{"Team A", "Team B"},
			oddsMultipliers: []float64{1.8, 2.2},
			votingMinutes:   120,
			validateOptions: func(t *testing.T, options []*models.GroupWagerOption) {
				require.Len(t, options, 2)
				assert.Equal(t, 1.8, options[0].OddsMultiplier)
				assert.Equal(t, 2.2, options[1].OddsMultiplier)
			},
		},
		{
			name:          "pool wager with many options",
			wagerType:     models.GroupWagerTypePool,
			condition:     "What will be the high temperature?",
			options:       []string{"< 60F", "60-70F", "70-80F", "80-90F", "> 90F"},
			votingMinutes: 1440, // 24 hours
			validateOptions: func(t *testing.T, options []*models.GroupWagerOption) {
				require.Len(t, options, 5)
				for i, opt := range options {
					assert.Equal(t, int16(i), opt.OptionOrder)
				}
			},
		},
		{
			name:            "house wager with uneven odds",
			wagerType:       models.GroupWagerTypeHouse,
			condition:       "Tournament winner?",
			options:         []string{"Favorite", "Dark Horse", "Underdog"},
			oddsMultipliers: []float64{1.5, 3.0, 10.0},
			votingMinutes:   10080, // Maximum 7 days
			validateOptions: func(t *testing.T, options []*models.GroupWagerOption) {
				require.Len(t, options, 3)
				assert.Equal(t, 1.5, options[0].OddsMultiplier)
				assert.Equal(t, 3.0, options[1].OddsMultiplier)
				assert.Equal(t, 10.0, options[2].OddsMultiplier)
			},
		},
	}

	for _, tt := range successTests {
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

			// Setup creator user
			creator := &models.User{
				DiscordID: TestResolverID,
				Username:  "creator",
				Balance:   TestInitialBalance,
			}
			helper.ExpectUserLookup(TestResolverID, creator)

			// Setup create expectations
			mocks.GroupWagerRepo.On("CreateWithOptions", ctx, 
				mock.MatchedBy(func(gw *models.GroupWager) bool {
					return gw.CreatorDiscordID == TestResolverID &&
						gw.Condition == tt.condition &&
						gw.State == models.GroupWagerStateActive &&
						gw.WagerType == tt.wagerType &&
						gw.TotalPot == 0 &&
						gw.MinParticipants == 3 &&
						gw.VotingPeriodMinutes == tt.votingMinutes &&
						gw.MessageID == TestMessageID &&
						gw.ChannelID == TestChannelID
				}),
				mock.MatchedBy(func(opts []*models.GroupWagerOption) bool {
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
						if tt.wagerType == models.GroupWagerTypeHouse {
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
			result, err := service.CreateGroupWager(
				ctx,
				TestResolverID,
				tt.condition,
				tt.options,
				tt.votingMinutes,
				TestMessageID,
				TestChannelID,
				tt.wagerType,
				tt.oddsMultipliers,
			)

			// Assert
			assertions.AssertNoError(err)
			assertions.AssertGroupWagerCreated(result, tt.wagerType, len(tt.options))
			tt.validateOptions(t, result.Options)

			mocks.AssertAllExpectations(t)
		})
	}
}

func TestGroupWagerService_CreateGroupWager_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mocks := NewTestMocks()
	helper := NewMockHelper(mocks)
	
	service := NewGroupWagerService(
		mocks.GroupWagerRepo,
		mocks.UserRepo,
		mocks.BalanceHistoryRepo,
		mocks.EventPublisher,
	)

	// Setup - user not found
	helper.ExpectUserNotFound(TestResolverID)

	// Execute
	result, err := service.CreateGroupWager(
		ctx,
		TestResolverID,
		"Test condition",
		[]string{"Yes", "No"},
		60,
		TestMessageID,
		TestChannelID,
		models.GroupWagerTypePool,
		nil,
	)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "creator not found")

	mocks.AssertAllExpectations(t)
}