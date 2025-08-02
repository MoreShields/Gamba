package services

import (
	"context"
	"errors"
	"testing"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGamblingService_PlaceBet_Win(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	existingUser := &entities.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	mockUserRepo.On("UpdateBalance", ctx, int64(123456), int64(10010)).Return(nil) // Balance 10000 + 10 win = 10010

	mockBalanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == 123456 &&
			h.BalanceBefore == 10000 &&
			h.BalanceAfter == 10010 &&
			h.ChangeAmount == 10 &&
			h.TransactionType == entities.TransactionTypeBetWin
	})).Return(nil).Run(func(args mock.Arguments) {
		// Set the ID on the history object for the bet creation
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = 42
	})

	mockBetRepo.On("Create", ctx, mock.MatchedBy(func(b *entities.Bet) bool {
		return b.DiscordID == 123456 &&
			b.Amount == 1000 &&
			b.WinProbability == 0.99 &&
			b.Won == true &&
			b.WinAmount == 10 &&
			*b.BalanceHistoryID == 42
	})).Return(nil)

	// Expect event publishing from RecordBalanceChange
	mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)

	// Force a win by setting a high probability
	result, err := service.PlaceBet(ctx, 123456, 0.99, 1000)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Won)
	assert.Equal(t, int64(1000), result.BetAmount)
	assert.Equal(t, int64(10), result.WinAmount)
	assert.Equal(t, int64(10010), result.NewBalance)

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockBetRepo.AssertExpectations(t)
	mockEventPublisher.AssertExpectations(t)
}

func TestGamblingService_PlaceBet_Loss(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	existingUser := &entities.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	mockUserRepo.On("UpdateBalance", ctx, int64(123456), int64(9000)).Return(nil) // Balance 10000 - 1000 bet = 9000

	mockBalanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == 123456 &&
			h.BalanceBefore == 10000 &&
			h.BalanceAfter == 9000 &&
			h.ChangeAmount == -1000 &&
			h.TransactionType == entities.TransactionTypeBetLoss
	})).Return(nil).Run(func(args mock.Arguments) {
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = 43
	})

	mockBetRepo.On("Create", ctx, mock.MatchedBy(func(b *entities.Bet) bool {
		return b.DiscordID == 123456 &&
			b.Amount == 1000 &&
			b.WinProbability == 0.01 &&
			b.Won == false &&
			*b.BalanceHistoryID == 43
	})).Return(nil)

	// Expect event publishing from RecordBalanceChange
	mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)

	// Force a loss by setting a very low probability
	result, err := service.PlaceBet(ctx, 123456, 0.01, 1000)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Won)
	assert.Equal(t, int64(1000), result.BetAmount)
	assert.Equal(t, int64(9000), result.NewBalance)

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockBetRepo.AssertExpectations(t)
	mockEventPublisher.AssertExpectations(t)
}

func TestGamblingService_PlaceBet_InvalidProbability(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)
	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	// Test probability too low
	result, err := service.PlaceBet(ctx, 123456, 0.0, 1000)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "win probability must be between 0 and 1 (exclusive)")

	// Test probability too high
	result, err = service.PlaceBet(ctx, 123456, 1.0, 1000)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "win probability must be between 0 and 1 (exclusive)")

	// Test negative probability
	result, err = service.PlaceBet(ctx, 123456, -0.1, 1000)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "win probability must be between 0 and 1 (exclusive)")

	mockUserRepo.AssertNotCalled(t, "GetByDiscordID")
	mockBetRepo.AssertNotCalled(t, "GetByUserSince")
}

func TestGamblingService_PlaceBet_InvalidAmount(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)
	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	// Test negative amount
	result, err := service.PlaceBet(ctx, 123456, 0.5, -100)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "bet amount must be positive")

	// Test zero amount
	result, err = service.PlaceBet(ctx, 123456, 0.5, 0)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "bet amount must be positive")

	mockUserRepo.AssertNotCalled(t, "GetByDiscordID")
	mockBetRepo.AssertNotCalled(t, "GetByUserSince")
}

func TestGamblingService_PlaceBet_InsufficientBalance(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	existingUser := &entities.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          500, // Less than bet amount
		AvailableBalance: 500,
	}

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	// No UpdateBalance call expected - service layer will catch insufficient balance before calling repository

	// Force a loss to trigger deduction
	result, err := service.PlaceBet(ctx, 123456, 0.01, 1000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient balance: have 500 available, need 1.0k")

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
	mockBetRepo.AssertNotCalled(t, "Create")
}

func TestGamblingService_PlaceBet_UserNotFound(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil) // User not found

	result, err := service.PlaceBet(ctx, 123456, 0.5, 1000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user not found")

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
	mockBetRepo.AssertNotCalled(t, "Create")
}

func TestGamblingService_PlaceBet_TransactionRollback(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	existingUser := &entities.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	// Accept any balance update - we're testing rollback, not the specific win/loss outcome
	mockUserRepo.On("UpdateBalance", ctx, int64(123456), mock.AnythingOfType("int64")).Return(nil)

	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = 44
	})

	// Bet creation fails, should trigger rollback
	mockBetRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

	// Expect event publishing from RecordBalanceChange (before bet creation fails)
	mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)

	// Force a win
	result, err := service.PlaceBet(ctx, 123456, 0.99, 1000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create bet record")

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockBetRepo.AssertExpectations(t)
	mockEventPublisher.AssertExpectations(t)
}
