package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"gambler/discord-client/events"
	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestRecordBalanceChangeEventFlow tests the complete event flow from RecordBalanceChange to event subscription
func TestRecordBalanceChangeEventFlow(t *testing.T) {
	ctx := context.Background()
	
	// Create real event bus and transactional bus (not mocked)
	mainBus := events.NewBus()
	transactionalBus := events.NewTransactionalBus(mainBus)
	
	// Set up event capture
	eventReceived := make(chan events.BalanceChangeEvent, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	
	// Subscribe to balance change events
	mainBus.Subscribe(events.EventTypeBalanceChange, func(ctx context.Context, event events.Event) {
		defer wg.Done()
		if balanceEvent, ok := event.(events.BalanceChangeEvent); ok {
			eventReceived <- balanceEvent
		}
	})
	
	// Create mock repository
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil)
	
	// Create test balance history
	history := &models.BalanceHistory{
		DiscordID:       123456,
		GuildID:         789,
		BalanceBefore:   1000,
		BalanceAfter:    1500,
		ChangeAmount:    500,
		TransactionType: models.TransactionTypeBetWin,
		TransactionMetadata: map[string]any{
			"username": "testuser",
		},
	}
	
	// Call RecordBalanceChange with transactional bus
	err := RecordBalanceChange(ctx, mockBalanceHistoryRepo, transactionalBus, history)
	assert.NoError(t, err)
	
	// Flush the transactional bus (simulating successful transaction commit)
	err = transactionalBus.Flush(ctx)
	assert.NoError(t, err)
	
	// Wait for event to be processed
	wg.Wait()
	
	// Verify the event was received
	select {
	case receivedEvent := <-eventReceived:
		assert.Equal(t, history.DiscordID, receivedEvent.UserID)
		assert.Equal(t, history.GuildID, receivedEvent.GuildID)
		assert.Equal(t, history.BalanceBefore, receivedEvent.OldBalance)
		assert.Equal(t, history.BalanceAfter, receivedEvent.NewBalance)
		assert.Equal(t, history.TransactionType, receivedEvent.TransactionType)
		assert.Equal(t, history.ChangeAmount, receivedEvent.ChangeAmount)
	case <-time.After(1 * time.Second):
		t.Fatal("Event was not received within timeout")
	}
	
	// Verify mock expectations
	mockBalanceHistoryRepo.AssertExpectations(t)
}

// TestRecordBalanceChangeUserCreatedEvent tests that user created events are also published
func TestRecordBalanceChangeUserCreatedEvent(t *testing.T) {
	ctx := context.Background()
	
	// Create real event bus and transactional bus
	mainBus := events.NewBus()
	transactionalBus := events.NewTransactionalBus(mainBus)
	
	// Set up event capture for both balance change and user created events
	balanceEventReceived := make(chan events.BalanceChangeEvent, 1)
	userCreatedEventReceived := make(chan events.UserCreatedEvent, 1)
	var wg sync.WaitGroup
	wg.Add(2) // Expecting both events
	
	// Subscribe to balance change events
	mainBus.Subscribe(events.EventTypeBalanceChange, func(ctx context.Context, event events.Event) {
		defer wg.Done()
		if balanceEvent, ok := event.(events.BalanceChangeEvent); ok {
			balanceEventReceived <- balanceEvent
		}
	})
	
	// Subscribe to user created events
	mainBus.Subscribe(events.EventTypeUserCreated, func(ctx context.Context, event events.Event) {
		defer wg.Done()
		if userEvent, ok := event.(events.UserCreatedEvent); ok {
			userCreatedEventReceived <- userEvent
		}
	})
	
	// Create mock repository
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil)
	
	// Create test balance history for initial balance (should trigger user created event)
	history := &models.BalanceHistory{
		DiscordID:       123456,
		GuildID:         789,
		BalanceBefore:   0,
		BalanceAfter:    100000, // Initial balance
		ChangeAmount:    100000,
		TransactionType: models.TransactionTypeInitial,
		TransactionMetadata: map[string]any{
			"username": "newuser",
		},
	}
	
	// Call RecordBalanceChange
	err := RecordBalanceChange(ctx, mockBalanceHistoryRepo, transactionalBus, history)
	assert.NoError(t, err)
	
	// Flush the transactional bus
	err = transactionalBus.Flush(ctx)
	assert.NoError(t, err)
	
	// Wait for both events to be processed
	wg.Wait()
	
	// Verify balance change event was received
	select {
	case balanceEvent := <-balanceEventReceived:
		assert.Equal(t, history.DiscordID, balanceEvent.UserID)
		assert.Equal(t, models.TransactionTypeInitial, balanceEvent.TransactionType)
	case <-time.After(1 * time.Second):
		t.Fatal("Balance change event was not received")
	}
	
	// Verify user created event was received
	select {
	case userEvent := <-userCreatedEventReceived:
		assert.Equal(t, history.DiscordID, userEvent.UserID)
		assert.Equal(t, history.DiscordID, userEvent.DiscordID)
		assert.Equal(t, "newuser", userEvent.Username)
		assert.Equal(t, history.BalanceAfter, userEvent.InitialBalance)
	case <-time.After(1 * time.Second):
		t.Fatal("User created event was not received")
	}
	
	// Verify mock expectations
	mockBalanceHistoryRepo.AssertExpectations(t)
}

// TestRecordBalanceChangeWithRollback tests that events are not delivered if transaction is rolled back
func TestRecordBalanceChangeWithRollback(t *testing.T) {
	ctx := context.Background()
	
	// Create real event bus and transactional bus
	mainBus := events.NewBus()
	transactionalBus := events.NewTransactionalBus(mainBus)
	
	// Set up event capture
	eventReceived := make(chan bool, 1)
	
	// Subscribe to balance change events
	mainBus.Subscribe(events.EventTypeBalanceChange, func(ctx context.Context, event events.Event) {
		eventReceived <- true
	})
	
	// Create mock repository
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockBalanceHistoryRepo.On("Record", ctx, mock.Anything).Return(nil)
	
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
	err := RecordBalanceChange(ctx, mockBalanceHistoryRepo, transactionalBus, history)
	assert.NoError(t, err)
	
	// Discard instead of flush (simulating transaction rollback)
	transactionalBus.Discard()
	
	// Verify no event was received
	select {
	case <-eventReceived:
		t.Fatal("Event was received despite transaction rollback")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}
	
	// Verify mock expectations
	mockBalanceHistoryRepo.AssertExpectations(t)
}