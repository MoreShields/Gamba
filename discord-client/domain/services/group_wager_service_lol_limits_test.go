package services

import (
	"context"
	"gambler/discord-client/domain/testhelpers"
	"testing"
	"time"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerService_PlaceBet_WithLolLimits(t *testing.T) {
	// Set up test config
	cfg := config.NewTestConfig()
	cfg.MaxLolWagerPerGame = 10000 // 10k max per game
	config.SetTestConfig(cfg)

	ctx := context.Background()

	testCases := []struct {
		name          string
		setupWager    func() *entities.GroupWagerDetail
		betAmount     int64
		expectError   bool
		errorContains string
	}{
		{
			name: "non-LoL wager - no limit applied",
			setupWager: func() *entities.GroupWagerDetail {
				return &entities.GroupWagerDetail{
					Wager: &entities.GroupWager{
						ID:      1,
						GuildID: TestGuildID,
						State:   entities.GroupWagerStateActive,
						VotingEndsAt: func() *time.Time {
							t := time.Now().Add(time.Hour)
							return &t
						}(),
						// No external reference
					},
					Options: []*entities.GroupWagerOption{
						{ID: 1, GroupWagerID: 1, OptionText: "Option 1"},
						{ID: 2, GroupWagerID: 1, OptionText: "Option 2"},
					},
				}
			},
			betAmount:   60000, // Over LoL limit, but should be allowed for non-LoL
			expectError: false,
		},
		{
			name: "LoL wager - under limit",
			setupWager: func() *entities.GroupWagerDetail {
				return &entities.GroupWagerDetail{
					Wager: &entities.GroupWager{
						ID:      1,
						GuildID: TestGuildID,
						State:   entities.GroupWagerStateActive,
						VotingEndsAt: func() *time.Time {
							t := time.Now().Add(time.Hour)
							return &t
						}(),
						ExternalRef: &entities.ExternalReference{
							System: entities.SystemLeagueOfLegends,
							ID:     "game123",
						},
					},
					Options: []*entities.GroupWagerOption{
						{ID: 1, GroupWagerID: 1, OptionText: "Win"},
						{ID: 2, GroupWagerID: 1, OptionText: "Loss"},
					},
				}
			},
			betAmount:   5000,
			expectError: false,
		},
		{
			name: "LoL wager - exactly at limit",
			setupWager: func() *entities.GroupWagerDetail {
				return &entities.GroupWagerDetail{
					Wager: &entities.GroupWager{
						ID:      1,
						GuildID: TestGuildID,
						State:   entities.GroupWagerStateActive,
						VotingEndsAt: func() *time.Time {
							t := time.Now().Add(time.Hour)
							return &t
						}(),
						ExternalRef: &entities.ExternalReference{
							System: entities.SystemLeagueOfLegends,
							ID:     "game123",
						},
					},
					Options: []*entities.GroupWagerOption{
						{ID: 1, GroupWagerID: 1, OptionText: "Win"},
						{ID: 2, GroupWagerID: 1, OptionText: "Loss"},
					},
				}
			},
			betAmount:   10000, // Exactly at the limit
			expectError: false,
		},
		{
			name: "LoL wager - over limit",
			setupWager: func() *entities.GroupWagerDetail {
				return &entities.GroupWagerDetail{
					Wager: &entities.GroupWager{
						ID:      1,
						GuildID: TestGuildID,
						State:   entities.GroupWagerStateActive,
						VotingEndsAt: func() *time.Time {
							t := time.Now().Add(time.Hour)
							return &t
						}(),
						ExternalRef: &entities.ExternalReference{
							System: entities.SystemLeagueOfLegends,
							ID:     "game123",
						},
					},
					Options: []*entities.GroupWagerOption{
						{ID: 1, GroupWagerID: 1, OptionText: "Win"},
						{ID: 2, GroupWagerID: 1, OptionText: "Loss"},
					},
				}
			},
			betAmount:     15000,
			expectError:   true,
			errorContains: "bet amount exceeds maximum of 10k bits per LoL game",
		},
		{
			name: "LoL wager - way over limit",
			setupWager: func() *entities.GroupWagerDetail {
				return &entities.GroupWagerDetail{
					Wager: &entities.GroupWager{
						ID:      1,
						GuildID: TestGuildID,
						State:   entities.GroupWagerStateActive,
						VotingEndsAt: func() *time.Time {
							t := time.Now().Add(time.Hour)
							return &t
						}(),
						ExternalRef: &entities.ExternalReference{
							System: entities.SystemLeagueOfLegends,
							ID:     "game456",
						},
					},
					Options: []*entities.GroupWagerOption{
						{ID: 1, GroupWagerID: 1, OptionText: "Win"},
						{ID: 2, GroupWagerID: 1, OptionText: "Loss"},
					},
				}
			},
			betAmount:     50000,
			expectError:   true,
			errorContains: "bet amount exceeds maximum of 10k bits per LoL game",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mocks
			mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
			mockUserRepo := new(testhelpers.MockUserRepository)
			mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
			mockEventPublisher := new(testhelpers.MockEventPublisher)

			service := NewGroupWagerService(
				mockGroupWagerRepo,
				mockUserRepo,
				mockBalanceHistoryRepo,
				mockEventPublisher,
			)

			// Set up wager detail
			wagerDetail := tc.setupWager()

			// Mock GetDetailByID
			mockGroupWagerRepo.On("GetDetailByID", ctx, int64(1)).
				Return(wagerDetail, nil)

			// Mock user lookup (if we don't expect an error from LoL limits)
			if !tc.expectError {
				mockUserRepo.On("GetByDiscordID", ctx, TestUser1ID).
					Return(&entities.User{
						DiscordID:        TestUser1ID,
						Balance:          100000,
						AvailableBalance: 100000,
					}, nil)

				// Mock participant check
				mockGroupWagerRepo.On("GetParticipant", ctx, int64(1), TestUser1ID).
					Return(nil, nil) // No existing participant

				// Mock the save
				mockGroupWagerRepo.On("SaveParticipant", ctx, mock.AnythingOfType("*entities.GroupWagerParticipant")).
					Return(nil)
				mockGroupWagerRepo.On("UpdateOptionTotal", ctx, int64(1), tc.betAmount).
					Return(nil)
				mockGroupWagerRepo.On("Update", ctx, mock.AnythingOfType("*entities.GroupWager")).
					Return(nil)
			}

			// Call PlaceBet
			participant, err := service.PlaceBet(ctx, 1, TestUser1ID, 1, tc.betAmount)

			// Validate results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, participant)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, participant)
				assert.Equal(t, tc.betAmount, participant.Amount)
			}

			// Verify mocks
			mockGroupWagerRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestGroupWagerService_PlaceBet_DifferentMaxLimits(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		maxLimit    int64
		betAmount   int64
		expectError bool
	}{
		{
			name:        "5k limit - under",
			maxLimit:    5000,
			betAmount:   4000,
			expectError: false,
		},
		{
			name:        "5k limit - over",
			maxLimit:    5000,
			betAmount:   6000,
			expectError: true,
		},
		{
			name:        "25k limit - under",
			maxLimit:    25000,
			betAmount:   20000,
			expectError: false,
		},
		{
			name:        "25k limit - over",
			maxLimit:    25000,
			betAmount:   30000,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test config with specific limit
			cfg := config.NewTestConfig()
			cfg.MaxLolWagerPerGame = tc.maxLimit
			config.SetTestConfig(cfg)

			// Set up mocks
			mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
			mockUserRepo := new(testhelpers.MockUserRepository)
			mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
			mockEventPublisher := new(testhelpers.MockEventPublisher)

			service := NewGroupWagerService(
				mockGroupWagerRepo,
				mockUserRepo,
				mockBalanceHistoryRepo,
				mockEventPublisher,
			)

			// Set up LoL wager
			wagerDetail := &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:      1,
					GuildID: TestGuildID,
					State:   entities.GroupWagerStateActive,
					VotingEndsAt: func() *time.Time {
						t := time.Now().Add(time.Hour)
						return &t
					}(),
					ExternalRef: &entities.ExternalReference{
						System: entities.SystemLeagueOfLegends,
						ID:     "game123",
					},
				},
				Options: []*entities.GroupWagerOption{
					{ID: 1, GroupWagerID: 1, OptionText: "Win"},
				},
			}

			// Mock GetDetailByID
			mockGroupWagerRepo.On("GetDetailByID", ctx, int64(1)).
				Return(wagerDetail, nil)

			// If we don't expect an error, mock the rest
			if !tc.expectError {
				mockUserRepo.On("GetByDiscordID", ctx, TestUser1ID).
					Return(&entities.User{
						DiscordID:        TestUser1ID,
						Balance:          100000,
						AvailableBalance: 100000,
					}, nil)

				mockGroupWagerRepo.On("GetParticipant", ctx, int64(1), TestUser1ID).
					Return(nil, nil)

				mockGroupWagerRepo.On("SaveParticipant", ctx, mock.AnythingOfType("*entities.GroupWagerParticipant")).
					Return(nil)
				mockGroupWagerRepo.On("UpdateOptionTotal", ctx, int64(1), tc.betAmount).
					Return(nil)
				mockGroupWagerRepo.On("Update", ctx, mock.AnythingOfType("*entities.GroupWager")).
					Return(nil)
			}

			// Call PlaceBet
			_, err := service.PlaceBet(ctx, 1, TestUser1ID, 1, tc.betAmount)

			// Validate results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "bet amount exceeds maximum")
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			mockGroupWagerRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}
