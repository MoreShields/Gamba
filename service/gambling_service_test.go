package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gambler/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGamblingService_PlaceBet_Win(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBetRepo := new(MockBetRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, mockBetRepo)

	service := NewGamblingService(mockFactory)

	existingUser := &models.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Commit").Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*models.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	mockUserRepo.On("AddBalance", ctx, int64(123456), int64(10)).Return(nil) // 0.99 odds, 1000 bet = ~10 win

	mockBalanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(h *models.BalanceHistory) bool {
		return h.DiscordID == 123456 &&
			h.BalanceBefore == 10000 &&
			h.BalanceAfter == 10010 &&
			h.ChangeAmount == 10 &&
			h.TransactionType == models.TransactionTypeBetWin
	})).Return(nil).Run(func(args mock.Arguments) {
		// Set the ID on the history object for the bet creation
		history := args.Get(1).(*models.BalanceHistory)
		history.ID = 42
	})

	mockBetRepo.On("Create", ctx, mock.MatchedBy(func(b *models.Bet) bool {
		return b.DiscordID == 123456 &&
			b.Amount == 1000 &&
			b.WinProbability == 0.99 &&
			b.Won == true &&
			b.WinAmount == 10 &&
			*b.BalanceHistoryID == 42
	})).Return(nil)

	// Force a win by setting a high probability
	result, err := service.PlaceBet(ctx, 123456, 0.99, 1000)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Won)
	assert.Equal(t, int64(1000), result.BetAmount)
	assert.Equal(t, int64(10), result.WinAmount)
	assert.Equal(t, int64(10010), result.NewBalance)

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockBetRepo.AssertExpectations(t)
}

func TestGamblingService_PlaceBet_Loss(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBetRepo := new(MockBetRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, mockBetRepo)

	service := NewGamblingService(mockFactory)

	existingUser := &models.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Commit").Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*models.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	mockUserRepo.On("DeductBalance", ctx, int64(123456), int64(1000)).Return(nil)

	mockBalanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(h *models.BalanceHistory) bool {
		return h.DiscordID == 123456 &&
			h.BalanceBefore == 10000 &&
			h.BalanceAfter == 9000 &&
			h.ChangeAmount == -1000 &&
			h.TransactionType == models.TransactionTypeBetLoss
	})).Return(nil).Run(func(args mock.Arguments) {
		history := args.Get(1).(*models.BalanceHistory)
		history.ID = 43
	})

	mockBetRepo.On("Create", ctx, mock.MatchedBy(func(b *models.Bet) bool {
		return b.DiscordID == 123456 &&
			b.Amount == 1000 &&
			b.WinProbability == 0.01 &&
			b.Won == false &&
			*b.BalanceHistoryID == 43
	})).Return(nil)

	// Force a loss by setting a very low probability
	result, err := service.PlaceBet(ctx, 123456, 0.01, 1000)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Won)
	assert.Equal(t, int64(1000), result.BetAmount)
	assert.Equal(t, int64(9000), result.NewBalance)

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockBetRepo.AssertExpectations(t)
}

func TestGamblingService_PlaceBet_InvalidProbability(t *testing.T) {
	ctx := context.Background()
	mockFactory := new(MockUnitOfWorkFactory)
	service := NewGamblingService(mockFactory)

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

	mockFactory.AssertNotCalled(t, "Create")
}

func TestGamblingService_PlaceBet_InvalidAmount(t *testing.T) {
	ctx := context.Background()
	mockFactory := new(MockUnitOfWorkFactory)
	service := NewGamblingService(mockFactory)

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

	mockFactory.AssertNotCalled(t, "Create")
}

func TestGamblingService_PlaceBet_InsufficientBalance(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBetRepo := new(MockBetRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, mockBetRepo)

	service := NewGamblingService(mockFactory)

	existingUser := &models.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          500, // Less than bet amount
		AvailableBalance: 500,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*models.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	mockUserRepo.On("DeductBalance", ctx, int64(123456), int64(1000)).Return(
		fmt.Errorf("insufficient balance: have 500, need 1000"))

	// Force a loss to trigger deduction
	result, err := service.PlaceBet(ctx, 123456, 0.01, 1000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient balance: have 500, need 1000")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
	mockBetRepo.AssertNotCalled(t, "Create")
}

func TestGamblingService_PlaceBet_UserNotFound(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBetRepo := new(MockBetRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, mockBetRepo)

	service := NewGamblingService(mockFactory)

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*models.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil) // User not found

	result, err := service.PlaceBet(ctx, 123456, 0.5, 1000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user not found")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
	mockBetRepo.AssertNotCalled(t, "Create")
}

func TestGamblingService_PlaceBet_TransactionRollback(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBetRepo := new(MockBetRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, mockBetRepo)

	service := NewGamblingService(mockFactory)

	existingUser := &models.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Mock for daily limit check
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*models.Bet{}, nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
	mockUserRepo.On("AddBalance", ctx, int64(123456), int64(10)).Return(nil)

	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		history := args.Get(1).(*models.BalanceHistory)
		history.ID = 44
	})

	// Bet creation fails, should trigger rollback
	mockBetRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

	// Force a win
	result, err := service.PlaceBet(ctx, 123456, 0.99, 1000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create bet record")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockBetRepo.AssertExpectations(t)
}
