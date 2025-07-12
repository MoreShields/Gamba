package events

import (
	"context"
	"gambler/models"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestEventDeliveryIntegration tests the complete event flow from TransactionalBus to main Bus
func TestEventDeliveryIntegration(t *testing.T) {
	// Create main event bus
	mainBus := NewBus()
	
	// Create transactional bus that wraps the main bus
	transactionalBus := NewTransactionalBus(mainBus)
	
	// Set up a channel to capture received events
	eventReceived := make(chan BalanceChangeEvent, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	
	// Subscribe to balance change events on the main bus
	mainBus.Subscribe(EventTypeBalanceChange, func(ctx context.Context, event Event) {
		defer wg.Done()
		if balanceEvent, ok := event.(BalanceChangeEvent); ok {
			select {
			case eventReceived <- balanceEvent:
			case <-time.After(1 * time.Second):
				t.Error("Timeout sending event to channel")
			}
		} else {
			t.Errorf("Expected BalanceChangeEvent, got %T", event)
		}
	})
	
	// Create a test event
	testEvent := BalanceChangeEvent{
		UserID:          123456,
		GuildID:         789,
		OldBalance:      1000,
		NewBalance:      1500,
		TransactionType: models.TransactionTypeBetWin,
		ChangeAmount:    500,
	}
	
	// Publish event to transactional bus (simulating service layer)
	transactionalBus.Publish(testEvent)
	
	// Flush events (simulating successful transaction commit)
	ctx := context.Background()
	err := transactionalBus.Flush(ctx)
	assert.NoError(t, err)
	
	// Wait for event to be processed
	wg.Wait()
	
	// Verify the event was received
	select {
	case receivedEvent := <-eventReceived:
		assert.Equal(t, testEvent.UserID, receivedEvent.UserID)
		assert.Equal(t, testEvent.GuildID, receivedEvent.GuildID)
		assert.Equal(t, testEvent.OldBalance, receivedEvent.OldBalance)
		assert.Equal(t, testEvent.NewBalance, receivedEvent.NewBalance)
		assert.Equal(t, testEvent.TransactionType, receivedEvent.TransactionType)
		assert.Equal(t, testEvent.ChangeAmount, receivedEvent.ChangeAmount)
	case <-time.After(2 * time.Second):
		t.Fatal("Event was not received within timeout")
	}
}

// TestMultipleEventsDelivery tests delivering multiple events in sequence
func TestMultipleEventsDelivery(t *testing.T) {
	mainBus := NewBus()
	transactionalBus := NewTransactionalBus(mainBus)
	
	eventsReceived := make(chan BalanceChangeEvent, 3)
	var wg sync.WaitGroup
	wg.Add(3)
	
	mainBus.Subscribe(EventTypeBalanceChange, func(ctx context.Context, event Event) {
		defer wg.Done()
		if balanceEvent, ok := event.(BalanceChangeEvent); ok {
			eventsReceived <- balanceEvent
		}
	})
	
	// Create and publish multiple test events
	events := []BalanceChangeEvent{
		{UserID: 1, GuildID: 100, OldBalance: 1000, NewBalance: 1100, TransactionType: models.TransactionTypeBetWin, ChangeAmount: 100},
		{UserID: 2, GuildID: 100, OldBalance: 2000, NewBalance: 2200, TransactionType: models.TransactionTypeBetWin, ChangeAmount: 200},
		{UserID: 3, GuildID: 100, OldBalance: 3000, NewBalance: 3300, TransactionType: models.TransactionTypeBetWin, ChangeAmount: 300},
	}
	
	for _, event := range events {
		transactionalBus.Publish(event)
	}
	
	// Flush all events
	ctx := context.Background()
	err := transactionalBus.Flush(ctx)
	assert.NoError(t, err)
	
	// Wait for all events to be processed
	wg.Wait()
	
	// Verify all events were received
	receivedEvents := make([]BalanceChangeEvent, 0, 3)
	for i := 0; i < 3; i++ {
		select {
		case event := <-eventsReceived:
			receivedEvents = append(receivedEvents, event)
		case <-time.After(2 * time.Second):
			t.Fatalf("Only received %d out of 3 events", len(receivedEvents))
		}
	}
	
	assert.Len(t, receivedEvents, 3)
	
	// Check that all original events were received (order may vary due to goroutines)
	userIDs := make(map[int64]bool)
	for _, received := range receivedEvents {
		userIDs[received.UserID] = true
	}
	
	assert.True(t, userIDs[1])
	assert.True(t, userIDs[2])
	assert.True(t, userIDs[3])
}

// TestTransactionalBusDiscard tests that discarded events are not delivered
func TestTransactionalBusDiscard(t *testing.T) {
	mainBus := NewBus()
	transactionalBus := NewTransactionalBus(mainBus)
	
	eventReceived := make(chan bool, 1)
	
	mainBus.Subscribe(EventTypeBalanceChange, func(ctx context.Context, event Event) {
		eventReceived <- true
	})
	
	// Publish event
	testEvent := BalanceChangeEvent{
		UserID:          123456,
		GuildID:         789,
		OldBalance:      1000,
		NewBalance:      1500,
		TransactionType: models.TransactionTypeBetWin,
		ChangeAmount:    500,
	}
	transactionalBus.Publish(testEvent)
	
	// Discard instead of flush (simulating transaction rollback)
	transactionalBus.Discard()
	
	// Verify no event was received
	select {
	case <-eventReceived:
		t.Fatal("Event was received despite being discarded")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}
}