package infrastructure

import (
	"context"
	"testing"

	"gambler/discord-client/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEventPublisher is a mock implementation of EventPublisher
type MockEventPublisher struct {
	PublishedEvents []events.Event
	PublishError    error
}

func (m *MockEventPublisher) Publish(event events.Event) error {
	if m.PublishError != nil {
		return m.PublishError
	}
	m.PublishedEvents = append(m.PublishedEvents, event)
	return nil
}

func TestNATSTransactionalPublisher_LocalHandlers(t *testing.T) {
	// Create mock publisher
	mockPublisher := &MockEventPublisher{
		PublishedEvents: make([]events.Event, 0),
	}

	// Create transactional publisher
	transPublisher := NewNATSTransactionalPublisher(mockPublisher).(*NATSTransactionalPublisher)

	// Track local handler invocations
	handlerCalled := false
	var receivedEvent events.Event

	// Register local handler for group wager state changes
	transPublisher.RegisterLocalHandler(events.EventTypeGroupWagerStateChange, func(ctx context.Context, event events.Event) error {
		handlerCalled = true
		receivedEvent = event
		return nil
	})

	// Create test event
	testEvent := events.GroupWagerStateChangeEvent{
		GroupWagerID: 123,
		GuildID:      456,
		OldState:     "active",
		NewState:     "pending_resolution",
		MessageID:    789,
		ChannelID:    101112,
	}

	// Publish event (it gets queued)
	err := transPublisher.Publish(testEvent)
	require.NoError(t, err)

	// Handler should not be called yet
	assert.False(t, handlerCalled)
	assert.Len(t, mockPublisher.PublishedEvents, 0)

	// Flush to trigger handlers and NATS publishing
	err = transPublisher.Flush(context.Background())
	require.NoError(t, err)

	// Verify local handler was called
	assert.True(t, handlerCalled)
	assert.Equal(t, testEvent, receivedEvent)

	// Verify event was also published to NATS
	assert.Len(t, mockPublisher.PublishedEvents, 1)
	assert.Equal(t, testEvent, mockPublisher.PublishedEvents[0])
}

func TestNATSTransactionalPublisher_MultipleLocalHandlers(t *testing.T) {
	// Create mock publisher
	mockPublisher := &MockEventPublisher{
		PublishedEvents: make([]events.Event, 0),
	}

	// Create transactional publisher
	transPublisher := NewNATSTransactionalPublisher(mockPublisher).(*NATSTransactionalPublisher)

	// Track handler invocations
	handler1Called := false
	handler2Called := false

	// Register multiple handlers for the same event type
	transPublisher.RegisterLocalHandler(events.EventTypeGroupWagerStateChange, func(ctx context.Context, event events.Event) error {
		handler1Called = true
		return nil
	})

	transPublisher.RegisterLocalHandler(events.EventTypeGroupWagerStateChange, func(ctx context.Context, event events.Event) error {
		handler2Called = true
		return nil
	})

	// Create and publish test event
	testEvent := events.GroupWagerStateChangeEvent{
		GroupWagerID: 123,
		GuildID:      456,
		OldState:     "active",
		NewState:     "cancelled",
		MessageID:    789,
		ChannelID:    101112,
	}

	err := transPublisher.Publish(testEvent)
	require.NoError(t, err)

	// Flush
	err = transPublisher.Flush(context.Background())
	require.NoError(t, err)

	// Verify both handlers were called
	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
}

func TestNATSTransactionalPublisher_Discard(t *testing.T) {
	// Create mock publisher
	mockPublisher := &MockEventPublisher{
		PublishedEvents: make([]events.Event, 0),
	}

	// Create transactional publisher
	transPublisher := NewNATSTransactionalPublisher(mockPublisher).(*NATSTransactionalPublisher)

	// Track handler invocations
	handlerCalled := false

	// Register local handler
	transPublisher.RegisterLocalHandler(events.EventTypeGroupWagerStateChange, func(ctx context.Context, event events.Event) error {
		handlerCalled = true
		return nil
	})

	// Publish event
	testEvent := events.GroupWagerStateChangeEvent{
		GroupWagerID: 123,
		GuildID:      456,
		OldState:     "active",
		NewState:     "resolved",
		MessageID:    789,
		ChannelID:    101112,
	}

	err := transPublisher.Publish(testEvent)
	require.NoError(t, err)

	// Discard instead of flush
	transPublisher.Discard()

	// Verify handler was NOT called and event was NOT published
	assert.False(t, handlerCalled)
	assert.Len(t, mockPublisher.PublishedEvents, 0)
}