package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHighRollerService_GetCurrentHighRoller(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setup        func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository)
		guildID      int64
		want         *interfaces.HighRollerInfo
		wantErr      bool
		errContains  string
	}{
		{
			name: "no purchases found",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(123)).Return(nil, nil)
				
				return mockRepo, mockUserRepo
			},
			guildID: 123,
			want: &interfaces.HighRollerInfo{
				CurrentPrice: 0,
			},
			wantErr: false,
		},
		{
			name: "latest purchase found with user",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				
				purchasedAt := time.Now()
				purchase := &entities.HighRollerPurchase{
					ID:            1,
					GuildID:       123,
					DiscordID:     456,
					PurchasePrice: 50000,
					PurchasedAt:   purchasedAt,
				}
				user := &entities.User{
					DiscordID: 456,
					Username:  "testuser",
					Balance:   100000,
				}
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(123)).Return(purchase, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(456)).Return(user, nil)
				
				return mockRepo, mockUserRepo
			},
			guildID: 123,
			want: &interfaces.HighRollerInfo{
				CurrentHolder:   &entities.User{DiscordID: 456, Username: "testuser", Balance: 100000},
				CurrentPrice:    50000,
				LastPurchasedAt: func() *time.Time { t := time.Now(); return &t }(),
			},
			wantErr: false,
		},
		{
			name: "repository error",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(123)).Return(nil, errors.New("database error"))
				
				return mockRepo, mockUserRepo
			},
			guildID:     123,
			wantErr:     true,
			errContains: "failed to get latest purchase",
		},
		{
			name: "user not found error",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				
				purchase := &entities.HighRollerPurchase{
					ID:            1,
					GuildID:       123,
					DiscordID:     456,
					PurchasePrice: 50000,
					PurchasedAt:   time.Now(),
				}
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(123)).Return(purchase, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(456)).Return(nil, errors.New("user not found"))
				
				return mockRepo, mockUserRepo
			},
			guildID:     123,
			wantErr:     true,
			errContains: "failed to get high roller user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			mockRepo, mockUserRepo := tt.setup()
			mockWagerRepo := new(testhelpers.MockWagerRepository)
			mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
			mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
			mockEventPublisher := new(testhelpers.MockEventPublisher)

			service := NewHighRollerService(
				mockRepo,
				mockUserRepo,
				mockWagerRepo,
				mockGroupWagerRepo,
				mockBalanceHistoryRepo,
				mockEventPublisher,
			)

			ctx := context.Background()
			result, err := service.GetCurrentHighRoller(ctx, tt.guildID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.want.CurrentPrice, result.CurrentPrice)
				
				if tt.want.CurrentHolder != nil {
					assert.NotNil(t, result.CurrentHolder)
					assert.Equal(t, tt.want.CurrentHolder.DiscordID, result.CurrentHolder.DiscordID)
					assert.Equal(t, tt.want.CurrentHolder.Username, result.CurrentHolder.Username)
					assert.NotNil(t, result.LastPurchasedAt)
				} else {
					assert.Nil(t, result.CurrentHolder)
					assert.Nil(t, result.LastPurchasedAt)
				}
			}

			mockRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestHighRollerService_PurchaseHighRollerRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher)
		discordID   int64
		guildID     int64
		offerAmount int64
		wantErr     bool
		errContains string
	}{
		{
			name: "invalid offer amount - zero",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				return new(testhelpers.MockHighRollerPurchaseRepository), new(testhelpers.MockUserRepository), new(testhelpers.MockWagerRepository), new(testhelpers.MockGroupWagerRepository), new(testhelpers.MockBalanceHistoryRepository), new(testhelpers.MockEventPublisher)
			},
			discordID:   123,
			guildID:     456,
			offerAmount: 0,
			wantErr:     true,
			errContains: "offer amount must be positive",
		},
		{
			name: "invalid offer amount - negative",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				return new(testhelpers.MockHighRollerPurchaseRepository), new(testhelpers.MockUserRepository), new(testhelpers.MockWagerRepository), new(testhelpers.MockGroupWagerRepository), new(testhelpers.MockBalanceHistoryRepository), new(testhelpers.MockEventPublisher)
			},
			discordID:   123,
			guildID:     456,
			offerAmount: -1000,
			wantErr:     true,
			errContains: "offer amount must be positive",
		},
		{
			name: "user already holds role",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				
				purchase := &entities.HighRollerPurchase{
					DiscordID:     123,
					PurchasePrice: 30000,
				}
				user := &entities.User{
					DiscordID: 123,
					Username:  "currentHolder",
				}
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(456)).Return(purchase, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(123)).Return(user, nil)
				
				return mockRepo, mockUserRepo, new(testhelpers.MockWagerRepository), new(testhelpers.MockGroupWagerRepository), new(testhelpers.MockBalanceHistoryRepository), new(testhelpers.MockEventPublisher)
			},
			discordID:   123,
			guildID:     456,
			offerAmount: 50000,
			wantErr:     true,
			errContains: "you already hold the high roller role",
		},
		{
			name: "offer too low",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				
				purchase := &entities.HighRollerPurchase{
					DiscordID:     789,  // Different user
					PurchasePrice: 50000,
				}
				user := &entities.User{
					DiscordID: 789,
					Username:  "otherUser",
				}
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(456)).Return(purchase, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(789)).Return(user, nil)
				
				return mockRepo, mockUserRepo, new(testhelpers.MockWagerRepository), new(testhelpers.MockGroupWagerRepository), new(testhelpers.MockBalanceHistoryRepository), new(testhelpers.MockEventPublisher)
			},
			discordID:   123,
			guildID:     456,
			offerAmount: 40000,  // Less than current price of 50000
			wantErr:     true,
			errContains: "offer must be greater than current price of 50000 bits",
		},
		{
			name: "insufficient balance",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				purchase := &entities.HighRollerPurchase{
					DiscordID:     789,
					PurchasePrice: 30000,
				}
				currentHolder := &entities.User{
					DiscordID: 789,
					Username:  "currentHolder",
				}
				buyer := &entities.User{
					DiscordID: 123,
					Username:  "buyer",
					Balance:   40000,  // Not enough for 60000 offer
				}
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(456)).Return(purchase, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(789)).Return(currentHolder, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(123)).Return(buyer, nil)
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return([]*entities.Wager{}, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return([]*entities.GroupWagerParticipant{}, nil)
				
				return mockRepo, mockUserRepo, mockWagerRepo, mockGroupWagerRepo, new(testhelpers.MockBalanceHistoryRepository), new(testhelpers.MockEventPublisher)
			},
			discordID:   123,
			guildID:     456,
			offerAmount: 60000,
			wantErr:     true,
			errContains: "insufficient balance: available 40000 bits, need 60000 bits",
		},
		{
			name: "successful purchase - no existing high roller",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
				mockEventPublisher := new(testhelpers.MockEventPublisher)
				
				buyer := &entities.User{
					DiscordID: 123,
					Username:  "buyer",
					Balance:   100000,
				}
				
				// No existing purchase
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(456)).Return(nil, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(123)).Return(buyer, nil)
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return([]*entities.Wager{}, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return([]*entities.GroupWagerParticipant{}, nil)
				mockUserRepo.On("UpdateBalance", mock.Anything, int64(123), int64(50000)).Return(nil)
				mockBalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
					return h.DiscordID == 123 &&
						h.BalanceBefore == 100000 &&
						h.BalanceAfter == 50000 &&
						h.ChangeAmount == -50000 &&
						h.TransactionType == entities.TransactionTypeHighRollerPurchase
				})).Return(nil)
				mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)
				mockRepo.On("CreatePurchase", mock.Anything, mock.MatchedBy(func(p *entities.HighRollerPurchase) bool {
					return p.GuildID == 456 &&
						p.DiscordID == 123 &&
						p.PurchasePrice == 50000
				})).Return(nil)
				
				return mockRepo, mockUserRepo, mockWagerRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher
			},
			discordID:   123,
			guildID:     456,
			offerAmount: 50000,
			wantErr:     false,
		},
		{
			name: "successful purchase - outbidding existing holder",
			setup: func() (*testhelpers.MockHighRollerPurchaseRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
				mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
				mockUserRepo := new(testhelpers.MockUserRepository)
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
				mockEventPublisher := new(testhelpers.MockEventPublisher)
				
				purchase := &entities.HighRollerPurchase{
					DiscordID:     789,
					PurchasePrice: 30000,
				}
				currentHolder := &entities.User{
					DiscordID: 789,
					Username:  "currentHolder",
				}
				buyer := &entities.User{
					DiscordID: 123,
					Username:  "buyer",
					Balance:   100000,
				}
				
				mockRepo.On("GetLatestPurchase", mock.Anything, int64(456)).Return(purchase, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(789)).Return(currentHolder, nil)
				mockUserRepo.On("GetByDiscordID", mock.Anything, int64(123)).Return(buyer, nil)
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return([]*entities.Wager{}, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return([]*entities.GroupWagerParticipant{}, nil)
				mockUserRepo.On("UpdateBalance", mock.Anything, int64(123), int64(50000)).Return(nil)
				mockBalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
					return h.DiscordID == 123 &&
						h.BalanceBefore == 100000 &&
						h.BalanceAfter == 50000 &&
						h.ChangeAmount == -50000 &&
						h.TransactionType == entities.TransactionTypeHighRollerPurchase
				})).Return(nil)
				mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)
				mockRepo.On("CreatePurchase", mock.Anything, mock.MatchedBy(func(p *entities.HighRollerPurchase) bool {
					return p.GuildID == 456 &&
						p.DiscordID == 123 &&
						p.PurchasePrice == 50000
				})).Return(nil)
				
				return mockRepo, mockUserRepo, mockWagerRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher
			},
			discordID:   123,
			guildID:     456,
			offerAmount: 50000,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			mockRepo, mockUserRepo, mockWagerRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := tt.setup()

			service := NewHighRollerService(
				mockRepo,
				mockUserRepo,
				mockWagerRepo,
				mockGroupWagerRepo,
				mockBalanceHistoryRepo,
				mockEventPublisher,
			)

			ctx := context.Background()
			err := service.PurchaseHighRollerRole(ctx, tt.discordID, tt.guildID, tt.offerAmount)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
			mockWagerRepo.AssertExpectations(t)
			mockGroupWagerRepo.AssertExpectations(t)
			mockBalanceHistoryRepo.AssertExpectations(t)
			mockEventPublisher.AssertExpectations(t)
		})
	}
}

func TestHighRollerService_calculateAvailableBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository)
		user    *entities.User
		want    int64
		wantErr bool
	}{
		{
			name: "no active wagers or participations",
			setup: func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository) {
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return([]*entities.Wager{}, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return([]*entities.GroupWagerParticipant{}, nil)
				
				return mockWagerRepo, mockGroupWagerRepo
			},
			user: &entities.User{
				DiscordID: 123,
				Balance:   100000,
			},
			want:    100000,
			wantErr: false,
		},
		{
			name: "with active wagers",
			setup: func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository) {
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				activeWagers := []*entities.Wager{
					{
						ProposerDiscordID: 123,
						Amount:           10000,
						State:            entities.WagerStateProposed,
					},
					{
						TargetDiscordID: 123,
						Amount:          15000,
						State:           entities.WagerStateVoting,
					},
					{
						ProposerDiscordID: 123,
						Amount:           5000,
						State:            entities.WagerStateResolved, // Should not be counted
					},
				}
				
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return(activeWagers, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return([]*entities.GroupWagerParticipant{}, nil)
				
				return mockWagerRepo, mockGroupWagerRepo
			},
			user: &entities.User{
				DiscordID: 123,
				Balance:   100000,
			},
			want:    75000, // 100000 - 10000 - 15000
			wantErr: false,
		},
		{
			name: "with group wager participations",
			setup: func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository) {
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				participations := []*entities.GroupWagerParticipant{
					{
						DiscordID: 123,
						Amount:    20000,
					},
					{
						DiscordID: 123,
						Amount:    5000,
					},
				}
				
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return([]*entities.Wager{}, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return(participations, nil)
				
				return mockWagerRepo, mockGroupWagerRepo
			},
			user: &entities.User{
				DiscordID: 123,
				Balance:   100000,
			},
			want:    75000, // 100000 - 20000 - 5000
			wantErr: false,
		},
		{
			name: "with both wagers and participations",
			setup: func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository) {
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				activeWagers := []*entities.Wager{
					{
						ProposerDiscordID: 123,
						Amount:           10000,
						State:            entities.WagerStateProposed,
					},
				}
				participations := []*entities.GroupWagerParticipant{
					{
						DiscordID: 123,
						Amount:    15000,
					},
				}
				
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return(activeWagers, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return(participations, nil)
				
				return mockWagerRepo, mockGroupWagerRepo
			},
			user: &entities.User{
				DiscordID: 123,
				Balance:   100000,
			},
			want:    75000, // 100000 - 10000 - 15000
			wantErr: false,
		},
		{
			name: "wager repository error",
			setup: func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository) {
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return(nil, errors.New("database error"))
				
				return mockWagerRepo, mockGroupWagerRepo
			},
			user: &entities.User{
				DiscordID: 123,
				Balance:   100000,
			},
			wantErr: true,
		},
		{
			name: "group wager repository error",
			setup: func() (*testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository) {
				mockWagerRepo := new(testhelpers.MockWagerRepository)
				mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
				
				mockWagerRepo.On("GetActiveByUser", mock.Anything, int64(123)).Return([]*entities.Wager{}, nil)
				mockGroupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123)).Return(nil, errors.New("database error"))
				
				return mockWagerRepo, mockGroupWagerRepo
			},
			user: &entities.User{
				DiscordID: 123,
				Balance:   100000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			mockWagerRepo, mockGroupWagerRepo := tt.setup()
			mockRepo := new(testhelpers.MockHighRollerPurchaseRepository)
			mockUserRepo := new(testhelpers.MockUserRepository)
			mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
			mockEventPublisher := new(testhelpers.MockEventPublisher)

			service := &highRollerService{
				highRollerRepo:     mockRepo,
				userRepo:           mockUserRepo,
				wagerRepo:          mockWagerRepo,
				groupWagerRepo:     mockGroupWagerRepo,
				balanceHistoryRepo: mockBalanceHistoryRepo,
				eventPublisher:     mockEventPublisher,
			}

			ctx := context.Background()
			result, err := service.calculateAvailableBalance(ctx, tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			mockWagerRepo.AssertExpectations(t)
			mockGroupWagerRepo.AssertExpectations(t)
		})
	}
}