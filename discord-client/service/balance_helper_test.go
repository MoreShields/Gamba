package service

import (
	"context"
	"testing"

	"gambler/discord-client/events"
	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestRecordBalanceChange tests that balance changes are recorded and events are published
func TestRecordBalanceChange(t *testing.T) {
	ctx := context.Background()
	
	// Create mock repository and event publisher
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockEventPublisher := new(MockEventPublisher)
	
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil)
	mockEventPublisher.On("Publish", mock.MatchedBy(func(event interface{}) bool {
		_, ok := event.(events.BalanceChangeEvent)
		return ok
	})).Return(nil)
	
	// Create test balance history
	history := &models.BalanceHistory{
		DiscordID:       123456,
		GuildID:         789,
		BalanceBefore:   1000,
		BalanceAfter:    1500,
		ChangeAmount:    500,
		TransactionType: models.TransactionTypeBetWin,
	}
	
	// Call RecordBalanceChange
	err := RecordBalanceChange(ctx, mockBalanceHistoryRepo, mockEventPublisher, history)
	assert.NoError(t, err)
	
	// Verify mock expectations
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockEventPublisher.AssertExpectations(t)
}

// TestRecordBalanceChangeUserCreatedEvent tests that user created events are published for initial balances
func TestRecordBalanceChangeUserCreatedEvent(t *testing.T) {
	ctx := context.Background()
	
	// Create mock repository and event publisher
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockEventPublisher := new(MockEventPublisher)
	
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil)
	// Expect both balance change and user created events
	mockEventPublisher.On("Publish", mock.MatchedBy(func(event interface{}) bool {
		_, ok := event.(events.BalanceChangeEvent)
		return ok
	})).Return(nil)
	mockEventPublisher.On("Publish", mock.MatchedBy(func(event interface{}) bool {
		_, ok := event.(events.UserCreatedEvent)
		return ok
	})).Return(nil)
	
	// Create test balance history for initial balance (balance before = 0)
	history := &models.BalanceHistory{
		DiscordID:       123456,
		GuildID:         789,
		BalanceBefore:   0, // Initial balance triggers user created event
		BalanceAfter:    100000,
		ChangeAmount:    100000,
		TransactionType: models.TransactionTypeInitial,
		TransactionMetadata: map[string]any{
			"username": "TestUser",
		},
	}
	
	// Call RecordBalanceChange
	err := RecordBalanceChange(ctx, mockBalanceHistoryRepo, mockEventPublisher, history)
	assert.NoError(t, err)
	
	// Verify mock expectations
	mockBalanceHistoryRepo.AssertExpectations(t)
	mockEventPublisher.AssertExpectations(t)
}

// TestRecordBalanceChangeRepositoryError tests error handling when repository fails
func TestRecordBalanceChangeRepositoryError(t *testing.T) {
	ctx := context.Background()
	
	// Create mock repository and event publisher
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockEventPublisher := new(MockEventPublisher)
	
	// Setup repository to return error
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(assert.AnError)
	
	// Create test balance history
	history := &models.BalanceHistory{
		DiscordID:       123456,
		GuildID:         789,
		BalanceBefore:   1000,
		BalanceAfter:    1500,
		ChangeAmount:    500,
		TransactionType: models.TransactionTypeBetWin,
	}
	
	// Call RecordBalanceChange - should return error
	err := RecordBalanceChange(ctx, mockBalanceHistoryRepo, mockEventPublisher, history)
	assert.Error(t, err)
	
	// Event should not be published when repository fails
	mockEventPublisher.AssertNotCalled(t, "Publish")
	
	// Verify mock expectations
	mockBalanceHistoryRepo.AssertExpectations(t)
}