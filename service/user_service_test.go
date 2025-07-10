package service

import (
	"context"
	"errors"
	"testing"

	"gambler/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_GetOrCreateUser_ExistingUser(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	existingUser := &models.User{
		DiscordID: 123456,
		Username:  "testuser",
		Balance:   50000,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)
	// No Commit() expected since user exists and no changes are made

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)

	user, err := service.GetOrCreateUser(ctx, 123456, "testuser")

	assert.NoError(t, err)
	assert.Equal(t, existingUser, user)

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
}

func TestUserService_GetOrCreateUser_NewUser(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	newUser := &models.User{
		DiscordID: 123456,
		Username:  "newuser",
		Balance:   InitialBalance,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Commit").Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// User doesn't exist on first check
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)
	// Create call returns new user
	mockUserRepo.On("Create", ctx, int64(123456), "newuser", InitialBalance).Return(newUser, nil)

	// Expect balance history to be recorded
	mockBalanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(h *models.BalanceHistory) bool {
		return h.DiscordID == 123456 &&
			h.BalanceBefore == 0 &&
			h.BalanceAfter == InitialBalance &&
			h.ChangeAmount == InitialBalance &&
			h.TransactionType == models.TransactionTypeInitial
	})).Return(nil)

	user, err := service.GetOrCreateUser(ctx, 123456, "newuser")

	assert.NoError(t, err)
	assert.Equal(t, newUser, user)

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
}

func TestUserService_GetOrCreateUser_CreateError(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// User doesn't exist
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)
	// Create fails
	mockUserRepo.On("Create", ctx, int64(123456), "failuser", InitialBalance).Return(nil, errors.New("database error"))

	user, err := service.GetOrCreateUser(ctx, 123456, "failuser")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to create user")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
}

func TestUserService_GetOrCreateUser_BalanceHistoryError(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	newUser := &models.User{
		DiscordID: 123456,
		Username:  "newuser",
		Balance:   InitialBalance,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)
	// No Commit expected since balance history error causes rollback

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)
	mockUserRepo.On("Create", ctx, int64(123456), "newuser", InitialBalance).Return(newUser, nil)

	// Balance history recording fails
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(errors.New("history error"))

	// Should fail due to history error
	user, err := service.GetOrCreateUser(ctx, 123456, "newuser")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to record initial balance history")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
}

func TestUserService_GetUser_Success(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	existingUser := &models.User{
		DiscordID: 123456,
		Username:  "testuser",
		Balance:   75000,
	}

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)
	// No Commit() expected since no changes are made

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)

	user, err := service.GetUser(ctx, 123456)

	assert.NoError(t, err)
	assert.Equal(t, existingUser, user)

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestUserService_GetUser_NotFound(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)

	user, err := service.GetUser(ctx, 123456)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user with discord ID 123456 not found")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestUserService_GetUser_Error(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, mockBalanceHistoryRepo, nil)

	service := NewUserService(mockFactory)

	// Mock expectations
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Rollback").Return(nil)

	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, errors.New("database error"))

	user, err := service.GetUser(ctx, 123456)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to get user")

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}
