package services

import (
	"gambler/discord-client/domain/testhelpers"
	"context"
	"errors"
	"testing"

	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_GetOrCreateUser_ExistingUser(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewUserService(mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)

	existingUser := &entities.User{
		DiscordID: 123456,
		Username:  "testuser",
		Balance:   50000,
	}

	// Mock expectations
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)

	user, err := service.GetOrCreateUser(ctx, 123456, "testuser")

	assert.NoError(t, err)
	assert.Equal(t, existingUser, user)

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
}

func TestUserService_GetOrCreateUser_NewUser(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewUserService(mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)

	newUser := &entities.User{
		DiscordID: 123456,
		Username:  "newuser",
		Balance:   InitialBalance,
	}

	// Mock expectations
	// User doesn't exist on first check
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)
	// Create call returns new user
	mockUserRepo.On("Create", ctx, int64(123456), "newuser", InitialBalance).Return(newUser, nil)

	// Expect balance history to be recorded
	mockBalanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == 123456 &&
			h.BalanceBefore == 0 &&
			h.BalanceAfter == InitialBalance &&
			h.ChangeAmount == InitialBalance &&
			h.TransactionType == entities.TransactionTypeInitial
	})).Return(nil)

	// Expect event publishing from RecordBalanceChange (both BalanceChangeEvent and UserCreatedEvent)
	mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return(nil)
	mockEventPublisher.On("Publish", mock.AnythingOfType("events.UserCreatedEvent")).Return(nil)

	user, err := service.GetOrCreateUser(ctx, 123456, "newuser")

	assert.NoError(t, err)
	assert.Equal(t, newUser, user)

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockEventPublisher.AssertExpectations(t)
}

func TestUserService_GetOrCreateUser_CreateError(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewUserService(mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)

	// Mock expectations
	// User doesn't exist
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)
	// Create fails
	mockUserRepo.On("Create", ctx, int64(123456), "failuser", InitialBalance).Return(nil, errors.New("database error"))

	user, err := service.GetOrCreateUser(ctx, 123456, "failuser")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to create user")

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertNotCalled(t, "Record")
}

func TestUserService_GetOrCreateUser_BalanceHistoryError(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewUserService(mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)

	newUser := &entities.User{
		DiscordID: 123456,
		Username:  "newuser",
		Balance:   InitialBalance,
	}

	// Mock expectations
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(nil, nil)
	mockUserRepo.On("Create", ctx, int64(123456), "newuser", InitialBalance).Return(newUser, nil)

	// Balance history recording fails
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(errors.New("history error"))

	// Even though balance history fails, event publishing won't be called since RecordBalanceChange fails early
	// No event publisher mocks needed for this test case

	// Should fail due to history error
	user, err := service.GetOrCreateUser(ctx, 123456, "newuser")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to record initial balance")

	mockUserRepo.AssertExpectations(t)
	mockBalanceHistoryRepo.AssertExpectations(t)
}
