package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper to create a test draw with common defaults
func createTestDraw(id, guildID int64, opts ...func(*entities.LotteryDraw)) *entities.LotteryDraw {
	draw := &entities.LotteryDraw{
		ID:          id,
		GuildID:     guildID,
		Difficulty:  8, // 256 possible numbers
		TicketCost:  1000,
		DrawTime:    time.Now().Add(1 * time.Hour), // Future draw time
		TotalPot:    0,
		CreatedAt:   time.Now(),
		CompletedAt: nil,
	}
	for _, opt := range opts {
		opt(draw)
	}
	return draw
}

// Helper to create test user
func createTestUser(discordID, balance int64) *entities.User {
	return &entities.User{
		DiscordID:        discordID,
		Username:         "testuser",
		Balance:          balance,
		AvailableBalance: balance,
		CreatedAt:        time.Now(),
	}
}

// Helper to create test guild settings
func createTestGuildSettings(guildID int64) *entities.GuildSettings {
	difficulty := int64(8)
	ticketCost := int64(1000)
	return &entities.GuildSettings{
		GuildID:         guildID,
		LottoDifficulty: &difficulty,
		LottoTicketCost: &ticketCost,
	}
}

// setupLotteryServiceMocks creates all the mock repositories needed for lottery service tests
func setupLotteryServiceMocks() (
	*testhelpers.MockLotteryDrawRepository,
	*testhelpers.MockLotteryTicketRepository,
	*testhelpers.MockLotteryWinnerRepository,
	*testhelpers.MockUserRepository,
	*testhelpers.MockWagerRepository,
	*testhelpers.MockGroupWagerRepository,
	*testhelpers.MockBalanceHistoryRepository,
	*testhelpers.MockGuildSettingsRepository,
	*testhelpers.MockEventPublisher,
) {
	return new(testhelpers.MockLotteryDrawRepository),
		new(testhelpers.MockLotteryTicketRepository),
		new(testhelpers.MockLotteryWinnerRepository),
		new(testhelpers.MockUserRepository),
		new(testhelpers.MockWagerRepository),
		new(testhelpers.MockGroupWagerRepository),
		new(testhelpers.MockBalanceHistoryRepository),
		new(testhelpers.MockGuildSettingsRepository),
		new(testhelpers.MockEventPublisher)
}

func TestLotteryService_CalculateNextDrawTime(t *testing.T) {
	t.Parallel()

	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, guildSettingsRepo, eventPublisher := setupLotteryServiceMocks()

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, guildSettingsRepo, eventPublisher,
	)

	nextDraw := service.CalculateNextDrawTime()

	// Verify it's a Friday
	assert.Equal(t, time.Friday, nextDraw.Weekday())

	// Verify it's at 2pm (14:00) UTC
	assert.Equal(t, 14, nextDraw.Hour())
	assert.Equal(t, 0, nextDraw.Minute())
	assert.Equal(t, 0, nextDraw.Second())

	// Verify it's in the future
	assert.True(t, nextDraw.After(time.Now().UTC()))
}

func TestLotteryService_GetOrCreateCurrentDraw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		guildID     int64
		setupMocks  func(*testhelpers.MockLotteryDrawRepository, *testhelpers.MockGuildSettingsRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful draw retrieval",
			guildID: 123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, settingsRepo *testhelpers.MockGuildSettingsRepository) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)
			},
			wantErr: false,
		},
		{
			name:    "guild settings error",
			guildID: 123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, settingsRepo *testhelpers.MockGuildSettingsRepository) {
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return((*entities.GuildSettings)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get guild settings",
		},
		{
			name:    "draw repository error",
			guildID: 123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, settingsRepo *testhelpers.MockGuildSettingsRepository) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return((*entities.LotteryDraw)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get or create lottery draw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

			tt.setupMocks(drawRepo, settingsRepo)

			service := NewLotteryService(
				drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
				balanceHistoryRepo, settingsRepo, eventPublisher,
			)

			draw, err := service.GetOrCreateCurrentDraw(ctx, tt.guildID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, draw)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, draw)
				assert.Equal(t, tt.guildID, draw.GuildID)
			}

			drawRepo.AssertExpectations(t)
			settingsRepo.AssertExpectations(t)
		})
	}
}

func TestLotteryService_PurchaseTickets_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		discordID   int64
		guildID     int64
		quantity    int
		setupMocks  func(*testhelpers.MockLotteryDrawRepository, *testhelpers.MockLotteryTicketRepository, *testhelpers.MockUserRepository, *testhelpers.MockWagerRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockGuildSettingsRepository, *testhelpers.MockEventPublisher)
		wantErr     bool
		errContains string
	}{
		{
			name:      "zero quantity",
			discordID: 123456,
			guildID:   123456789,
			quantity:  0,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				// No mocks needed - validation fails immediately
			},
			wantErr:     true,
			errContains: "quantity must be positive",
		},
		{
			name:      "negative quantity",
			discordID: 123456,
			guildID:   123456789,
			quantity:  -5,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				// No mocks needed - validation fails immediately
			},
			wantErr:     true,
			errContains: "quantity must be positive",
		},
		{
			name:      "draw already completed",
			discordID: 123456,
			guildID:   123456789,
			quantity:  1,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				now := time.Now()
				draw := createTestDraw(1, 123456789, func(d *entities.LotteryDraw) {
					d.CompletedAt = &now // Already completed
				})
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)
			},
			wantErr:     true,
			errContains: "tickets can no longer be purchased",
		},
		{
			name:      "draw time passed",
			discordID: 123456,
			guildID:   123456789,
			quantity:  1,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789, func(d *entities.LotteryDraw) {
					d.DrawTime = time.Now().Add(-1 * time.Hour) // Past draw time
				})
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)
			},
			wantErr:     true,
			errContains: "tickets can no longer be purchased",
		},
		{
			name:      "user not found",
			discordID: 123456,
			guildID:   123456789,
			quantity:  1,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)

				userRepo.On("GetByDiscordID", mock.Anything, int64(123456)).Return(nil, nil) // User not found
			},
			wantErr:     true,
			errContains: "user not found",
		},
		{
			name:      "insufficient balance",
			discordID: 123456,
			guildID:   123456789,
			quantity:  5,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)

				user := createTestUser(123456, 2000) // Only 2000, needs 5000
				userRepo.On("GetByDiscordID", mock.Anything, int64(123456)).Return(user, nil)

				// No active wagers or group wager participations
				wagerRepo.On("GetActiveByUser", mock.Anything, int64(123456)).Return([]*entities.Wager{}, nil)
				groupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123456)).Return([]*entities.GroupWagerParticipant{}, nil)
			},
			wantErr:     true,
			errContains: "insufficient balance",
		},
		{
			name:      "not enough tickets available",
			discordID: 123456,
			guildID:   123456789,
			quantity:  100,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, userRepo *testhelpers.MockUserRepository, wagerRepo *testhelpers.MockWagerRepository, groupWagerRepo *testhelpers.MockGroupWagerRepository, balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository, settingsRepo *testhelpers.MockGuildSettingsRepository, eventPublisher *testhelpers.MockEventPublisher) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789, func(d *entities.LotteryDraw) {
					d.Difficulty = 4 // Only 16 possible numbers
				})
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)

				user := createTestUser(123456, 1000000) // Plenty of balance
				userRepo.On("GetByDiscordID", mock.Anything, int64(123456)).Return(user, nil)

				wagerRepo.On("GetActiveByUser", mock.Anything, int64(123456)).Return([]*entities.Wager{}, nil)
				groupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, int64(123456)).Return([]*entities.GroupWagerParticipant{}, nil)

				// User already has 10 numbers out of 16 possible
				usedNumbers := make([]int64, 10)
				for i := 0; i < 10; i++ {
					usedNumbers[i] = int64(i)
				}
				ticketRepo.On("GetUsedNumbersByUser", mock.Anything, int64(1), int64(123456)).Return(usedNumbers, nil)
			},
			wantErr:     true,
			errContains: "you already have 10 tickets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

			tt.setupMocks(drawRepo, ticketRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher)

			service := NewLotteryService(
				drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
				balanceHistoryRepo, settingsRepo, eventPublisher,
			)

			result, err := service.PurchaseTickets(ctx, tt.discordID, tt.guildID, tt.quantity)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestLotteryService_PurchaseTickets_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

	guildID := int64(123456789)
	discordID := int64(123456)
	quantity := 3
	ticketCost := int64(1000)

	// Setup guild settings
	settings := createTestGuildSettings(guildID)
	settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, guildID).Return(settings, nil)

	// Setup draw
	draw := createTestDraw(1, guildID)
	drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, guildID, mock.AnythingOfType("time.Time"), int64(8), ticketCost).Return(draw, nil)

	// Setup user with sufficient balance
	user := createTestUser(discordID, 10000)
	userRepo.On("GetByDiscordID", mock.Anything, discordID).Return(user, nil)

	// No active wagers
	wagerRepo.On("GetActiveByUser", mock.Anything, discordID).Return([]*entities.Wager{}, nil)
	groupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, discordID).Return([]*entities.GroupWagerParticipant{}, nil)

	// No used numbers for this user
	ticketRepo.On("GetUsedNumbersByUser", mock.Anything, int64(1), discordID).Return([]int64{}, nil)

	// Update balance
	expectedNewBalance := user.Balance - (ticketCost * int64(quantity))
	userRepo.On("UpdateBalance", mock.Anything, discordID, expectedNewBalance).Return(nil)

	// Record balance history
	balanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == discordID &&
			h.GuildID == guildID &&
			h.BalanceBefore == user.Balance &&
			h.BalanceAfter == expectedNewBalance &&
			h.ChangeAmount == -(ticketCost*int64(quantity)) &&
			h.TransactionType == entities.TransactionTypeLottoTicket
	})).Return(nil).Run(func(args mock.Arguments) {
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = 100
	})

	// Event publishing
	eventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)

	// Create tickets (batch insert)
	ticketRepo.On("CreateBatch", mock.Anything, mock.MatchedBy(func(tickets []*entities.LotteryTicket) bool {
		if len(tickets) != quantity {
			return false
		}
		for _, t := range tickets {
			if t.DrawID != draw.ID || t.GuildID != guildID || t.DiscordID != discordID ||
				t.PurchasePrice != ticketCost || t.BalanceHistoryID != 100 {
				return false
			}
		}
		return true
	})).Return(nil)

	// Increment pot
	drawRepo.On("IncrementPot", mock.Anything, draw.ID, ticketCost*int64(quantity)).Return(nil)

	// Refresh draw
	updatedDraw := *draw
	updatedDraw.TotalPot = ticketCost * int64(quantity)
	drawRepo.On("GetByID", mock.Anything, draw.ID).Return(&updatedDraw, nil)

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, settingsRepo, eventPublisher,
	)

	result, err := service.PurchaseTickets(ctx, discordID, guildID, quantity)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Tickets, quantity)
	assert.Equal(t, ticketCost*int64(quantity), result.TotalCost)
	assert.Equal(t, expectedNewBalance, result.NewBalance)
	assert.Equal(t, draw.ID, result.Draw.ID)

	drawRepo.AssertExpectations(t)
	ticketRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	wagerRepo.AssertExpectations(t)
	groupWagerRepo.AssertExpectations(t)
	balanceHistoryRepo.AssertExpectations(t)
	settingsRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestLotteryService_PurchaseTickets_WithLockedBalance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

	guildID := int64(123456789)
	discordID := int64(123456)
	quantity := 5
	ticketCost := int64(1000)

	// Setup guild settings
	settings := createTestGuildSettings(guildID)
	settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, guildID).Return(settings, nil)

	// Setup draw
	draw := createTestDraw(1, guildID)
	drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, guildID, mock.AnythingOfType("time.Time"), int64(8), ticketCost).Return(draw, nil)

	// User has 10000 balance
	user := createTestUser(discordID, 10000)
	userRepo.On("GetByDiscordID", mock.Anything, discordID).Return(user, nil)

	// User has 6000 locked in active wager (proposer for 3000, target for 3000)
	activeWagers := []*entities.Wager{
		{
			ID:                 1,
			ProposerDiscordID:  discordID,
			TargetDiscordID:    999999,
			Amount:             3000,
			State:              entities.WagerStateProposed,
		},
		{
			ID:                 2,
			ProposerDiscordID:  888888,
			TargetDiscordID:    discordID,
			Amount:             3000,
			State:              entities.WagerStateVoting,
		},
	}
	wagerRepo.On("GetActiveByUser", mock.Anything, discordID).Return(activeWagers, nil)
	groupWagerRepo.On("GetActiveParticipationsByUser", mock.Anything, discordID).Return([]*entities.GroupWagerParticipant{}, nil)

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, settingsRepo, eventPublisher,
	)

	// Available balance: 10000 - 6000 = 4000
	// Needed: 5 * 1000 = 5000
	result, err := service.PurchaseTickets(ctx, discordID, guildID, quantity)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient balance")
	assert.Contains(t, err.Error(), "have 4000 available")
	assert.Contains(t, err.Error(), "need 5000")
}

func TestLotteryService_GetUserTickets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		discordID   int64
		guildID     int64
		setupMocks  func(*testhelpers.MockLotteryDrawRepository, *testhelpers.MockLotteryTicketRepository)
		wantTickets int
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful ticket retrieval",
			discordID: 123456,
			guildID:   123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository) {
				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetCurrentOpenDraw", mock.Anything, int64(123456789)).Return(draw, nil)

				tickets := []*entities.LotteryTicket{
					{ID: 1, DrawID: 1, DiscordID: 123456, TicketNumber: 42},
					{ID: 2, DrawID: 1, DiscordID: 123456, TicketNumber: 100},
				}
				ticketRepo.On("GetByUserForDraw", mock.Anything, int64(1), int64(123456)).Return(tickets, nil)
			},
			wantTickets: 2,
			wantErr:     false,
		},
		{
			name:      "no current draw",
			discordID: 123456,
			guildID:   123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository) {
				drawRepo.On("GetCurrentOpenDraw", mock.Anything, int64(123456789)).Return(nil, nil)
			},
			wantTickets: 0,
			wantErr:     false,
		},
		{
			name:      "draw repository error",
			discordID: 123456,
			guildID:   123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository) {
				drawRepo.On("GetCurrentOpenDraw", mock.Anything, int64(123456789)).Return((*entities.LotteryDraw)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get current draw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

			tt.setupMocks(drawRepo, ticketRepo)

			service := NewLotteryService(
				drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
				balanceHistoryRepo, settingsRepo, eventPublisher,
			)

			tickets, err := service.GetUserTickets(ctx, tt.discordID, tt.guildID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.wantTickets == 0 {
					assert.Nil(t, tickets)
				} else {
					assert.Len(t, tickets, tt.wantTickets)
				}
			}

			drawRepo.AssertExpectations(t)
			ticketRepo.AssertExpectations(t)
		})
	}
}

func TestLotteryService_GetDrawInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		guildID            int64
		setupMocks         func(*testhelpers.MockLotteryDrawRepository, *testhelpers.MockLotteryTicketRepository, *testhelpers.MockGuildSettingsRepository)
		wantTicketCount    int64
		wantParticipants   int
		wantErr            bool
		errContains        string
	}{
		{
			name:    "successful draw info retrieval",
			guildID: 123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, settingsRepo *testhelpers.MockGuildSettingsRepository) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789)
				draw.TotalPot = 5000
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)

				ticketRepo.On("CountTicketsForDraw", mock.Anything, int64(1)).Return(int64(5), nil)

				participants := []*entities.LotteryParticipantInfo{
					{DiscordID: 111, TicketCount: 3},
					{DiscordID: 222, TicketCount: 2},
				}
				ticketRepo.On("GetParticipantSummary", mock.Anything, int64(1)).Return(participants, nil)
			},
			wantTicketCount:  5,
			wantParticipants: 2,
			wantErr:          false,
		},
		{
			name:    "draw count error",
			guildID: 123456789,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository, ticketRepo *testhelpers.MockLotteryTicketRepository, settingsRepo *testhelpers.MockGuildSettingsRepository) {
				settings := createTestGuildSettings(123456789)
				settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, int64(123456789)).Return(settings, nil)

				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, int64(123456789), mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(draw, nil)

				ticketRepo.On("CountTicketsForDraw", mock.Anything, int64(1)).Return(int64(0), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to count tickets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

			tt.setupMocks(drawRepo, ticketRepo, settingsRepo)

			service := NewLotteryService(
				drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
				balanceHistoryRepo, settingsRepo, eventPublisher,
			)

			info, err := service.GetDrawInfo(ctx, tt.guildID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, info)
				assert.Equal(t, tt.wantTicketCount, info.TicketCount)
				assert.Len(t, info.Participants, tt.wantParticipants)
			}

			drawRepo.AssertExpectations(t)
			ticketRepo.AssertExpectations(t)
			settingsRepo.AssertExpectations(t)
		})
	}
}

func TestLotteryService_ConductDraw_AlreadyCompleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

	now := time.Now()
	draw := createTestDraw(1, 123456789, func(d *entities.LotteryDraw) {
		d.CompletedAt = &now
	})

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, settingsRepo, eventPublisher,
	)

	result, err := service.ConductDraw(ctx, draw)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "draw already completed")
}

func TestLotteryService_ConductDraw_WithWinner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

	guildID := int64(123456789)
	winnerID := int64(123456)
	potAmount := int64(10000)

	draw := createTestDraw(1, guildID, func(d *entities.LotteryDraw) {
		d.TotalPot = potAmount
	})

	// Lock the draw for update
	drawRepo.On("GetByIDForUpdate", mock.Anything, draw.ID).Return(draw, nil)

	// Mock winning tickets - one winner
	winningTickets := []*entities.LotteryTicket{
		{ID: 1, DrawID: draw.ID, DiscordID: winnerID, TicketNumber: 42},
	}
	ticketRepo.On("GetWinningTickets", mock.Anything, draw.ID, mock.AnythingOfType("int64")).Return(winningTickets, nil)

	// Winner user
	winner := createTestUser(winnerID, 5000)
	userRepo.On("GetByDiscordID", mock.Anything, winnerID).Return(winner, nil)

	// Update winner balance
	newBalance := winner.Balance + potAmount
	userRepo.On("UpdateBalance", mock.Anything, winnerID, newBalance).Return(nil)

	// Record balance history for winner
	balanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == winnerID &&
			h.GuildID == guildID &&
			h.BalanceBefore == winner.Balance &&
			h.BalanceAfter == newBalance &&
			h.ChangeAmount == potAmount &&
			h.TransactionType == entities.TransactionTypeLottoWin
	})).Return(nil)

	// Event publishing
	eventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)

	// Create LotteryWinner record
	winnerRepo.On("Create", mock.Anything, mock.MatchedBy(func(w *entities.LotteryWinner) bool {
		return w.DrawID == draw.ID &&
			w.DiscordID == winnerID &&
			w.TicketID == int64(1) &&
			w.WinningAmount == potAmount
	})).Return(nil)

	// Update draw record
	drawRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *entities.LotteryDraw) bool {
		return d.ID == draw.ID &&
			d.CompletedAt != nil &&
			d.WinningNumber != nil
	})).Return(nil)

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, settingsRepo, eventPublisher,
	)

	result, err := service.ConductDraw(ctx, draw)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, potAmount, result.PotAmount)
	assert.False(t, result.RolledOver)
	assert.Len(t, result.Winners, 1)
	assert.Equal(t, winnerID, result.Winners[0].DiscordID)

	drawRepo.AssertExpectations(t)
	ticketRepo.AssertExpectations(t)
	winnerRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	balanceHistoryRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestLotteryService_ConductDraw_MultipleWinners(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

	guildID := int64(123456789)
	potAmount := int64(10000)
	winnerID1 := int64(111111)
	winnerID2 := int64(222222)

	draw := createTestDraw(1, guildID, func(d *entities.LotteryDraw) {
		d.TotalPot = potAmount
	})

	// Lock the draw for update
	drawRepo.On("GetByIDForUpdate", mock.Anything, draw.ID).Return(draw, nil)

	// Mock winning tickets - two winners with same number (edge case)
	winningTickets := []*entities.LotteryTicket{
		{ID: 1, DrawID: draw.ID, DiscordID: winnerID1, TicketNumber: 42},
		{ID: 2, DrawID: draw.ID, DiscordID: winnerID2, TicketNumber: 42},
	}
	ticketRepo.On("GetWinningTickets", mock.Anything, draw.ID, mock.AnythingOfType("int64")).Return(winningTickets, nil)

	// Winner users
	winner1 := createTestUser(winnerID1, 5000)
	winner2 := createTestUser(winnerID2, 3000)
	userRepo.On("GetByDiscordID", mock.Anything, winnerID1).Return(winner1, nil)
	userRepo.On("GetByDiscordID", mock.Anything, winnerID2).Return(winner2, nil)

	// Each winner gets half the pot
	winningsPerWinner := potAmount / 2
	userRepo.On("UpdateBalance", mock.Anything, winnerID1, winner1.Balance+winningsPerWinner).Return(nil)
	userRepo.On("UpdateBalance", mock.Anything, winnerID2, winner2.Balance+winningsPerWinner).Return(nil)

	// Record balance history for both winners
	balanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.TransactionType == entities.TransactionTypeLottoWin &&
			h.ChangeAmount == winningsPerWinner
	})).Return(nil).Times(2)

	// Event publishing for both winners
	eventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil).Times(2)

	// Create LotteryWinner records for both winners
	winnerRepo.On("Create", mock.Anything, mock.MatchedBy(func(w *entities.LotteryWinner) bool {
		return w.DrawID == draw.ID && w.WinningAmount == winningsPerWinner
	})).Return(nil).Times(2)

	// Update draw record
	drawRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *entities.LotteryDraw) bool {
		return d.ID == draw.ID && d.CompletedAt != nil
	})).Return(nil)

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, settingsRepo, eventPublisher,
	)

	result, err := service.ConductDraw(ctx, draw)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, potAmount, result.PotAmount)
	assert.False(t, result.RolledOver)
	assert.Len(t, result.Winners, 2)

	drawRepo.AssertExpectations(t)
	ticketRepo.AssertExpectations(t)
	winnerRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	balanceHistoryRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestLotteryService_ConductDraw_NoWinner_Rollover(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

	guildID := int64(123456789)
	potAmount := int64(10000)

	draw := createTestDraw(1, guildID, func(d *entities.LotteryDraw) {
		d.TotalPot = potAmount
	})

	// Lock the draw for update
	drawRepo.On("GetByIDForUpdate", mock.Anything, draw.ID).Return(draw, nil)

	// No winning tickets
	ticketRepo.On("GetWinningTickets", mock.Anything, draw.ID, mock.AnythingOfType("int64")).Return([]*entities.LotteryTicket{}, nil)

	// Update draw record (marked as complete with no winner)
	drawRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *entities.LotteryDraw) bool {
		return d.ID == draw.ID && d.CompletedAt != nil
	})).Return(nil)

	// Guild settings for next draw
	settings := createTestGuildSettings(guildID)
	settingsRepo.On("GetOrCreateGuildSettings", mock.Anything, guildID).Return(settings, nil)

	// Create next draw for rollover
	nextDraw := createTestDraw(2, guildID)
	drawRepo.On("GetOrCreateCurrentDraw", mock.Anything, guildID, mock.AnythingOfType("time.Time"), int64(8), int64(1000)).Return(nextDraw, nil)

	// Increment pot on next draw
	drawRepo.On("IncrementPot", mock.Anything, nextDraw.ID, potAmount).Return(nil)

	service := NewLotteryService(
		drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
		balanceHistoryRepo, settingsRepo, eventPublisher,
	)

	result, err := service.ConductDraw(ctx, draw)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, potAmount, result.PotAmount)
	assert.True(t, result.RolledOver)
	assert.Empty(t, result.Winners)
	assert.NotNil(t, result.NextDraw)

	drawRepo.AssertExpectations(t)
	ticketRepo.AssertExpectations(t)
	settingsRepo.AssertExpectations(t)
}

func TestLotteryService_SetDrawMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		drawID      int64
		channelID   int64
		messageID   int64
		setupMocks  func(*testhelpers.MockLotteryDrawRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful message update",
			drawID:    1,
			channelID: 111222333,
			messageID: 444555666,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository) {
				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetByID", mock.Anything, int64(1)).Return(draw, nil)
				drawRepo.On("Update", mock.Anything, mock.MatchedBy(func(d *entities.LotteryDraw) bool {
					return d.ID == 1 &&
						d.ChannelID != nil && *d.ChannelID == int64(111222333) &&
						d.MessageID != nil && *d.MessageID == int64(444555666)
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "draw not found",
			drawID:    999,
			channelID: 111222333,
			messageID: 444555666,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository) {
				drawRepo.On("GetByID", mock.Anything, int64(999)).Return(nil, nil)
			},
			wantErr:     true,
			errContains: "draw not found",
		},
		{
			name:      "get draw error",
			drawID:    1,
			channelID: 111222333,
			messageID: 444555666,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository) {
				drawRepo.On("GetByID", mock.Anything, int64(1)).Return((*entities.LotteryDraw)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get draw",
		},
		{
			name:      "update error",
			drawID:    1,
			channelID: 111222333,
			messageID: 444555666,
			setupMocks: func(drawRepo *testhelpers.MockLotteryDrawRepository) {
				draw := createTestDraw(1, 123456789)
				drawRepo.On("GetByID", mock.Anything, int64(1)).Return(draw, nil)
				drawRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to update draw message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo, balanceHistoryRepo, settingsRepo, eventPublisher := setupLotteryServiceMocks()

			tt.setupMocks(drawRepo)

			service := NewLotteryService(
				drawRepo, ticketRepo, winnerRepo, userRepo, wagerRepo, groupWagerRepo,
				balanceHistoryRepo, settingsRepo, eventPublisher,
			)

			err := service.SetDrawMessage(ctx, tt.drawID, tt.channelID, tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			drawRepo.AssertExpectations(t)
		})
	}
}
