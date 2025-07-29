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

func TestNATSTransactionalPublisher_BufferingAndFlush(t *testing.T) {
	// Create mock publisher
	mockPublisher := &MockEventPublisher{
		PublishedEvents: make([]events.Event, 0),
	}

	// Create transactional publisher
	transPublisher := NewNATSTransactionalPublisher(mockPublisher).(*NATSTransactionalPublisher)

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

	// Event should not be published yet
	assert.Len(t, mockPublisher.PublishedEvents, 0)

	// Flush to trigger publishing
	err = transPublisher.Flush(context.Background())
	require.NoError(t, err)

	// Verify event was published
	assert.Len(t, mockPublisher.PublishedEvents, 1)
	assert.Equal(t, testEvent, mockPublisher.PublishedEvents[0])
}

func TestNATSTransactionalPublisher_MultipleEvents(t *testing.T) {
	// Create mock publisher
	mockPublisher := &MockEventPublisher{
		PublishedEvents: make([]events.Event, 0),
	}

	// Create transactional publisher
	transPublisher := NewNATSTransactionalPublisher(mockPublisher).(*NATSTransactionalPublisher)

	// Create and publish multiple test events
	testEvent1 := events.GroupWagerStateChangeEvent{
		GroupWagerID: 123,
		GuildID:      456,
		OldState:     "active",
		NewState:     "cancelled",
		MessageID:    789,
		ChannelID:    101112,
	}

	testEvent2 := events.BalanceChangeEvent{
		UserID:          111,
		GuildID:         456,
		OldBalance:      1000,
		NewBalance:      2000,
		TransactionType: "bet_win",
		ChangeAmount:    1000,
	}

	err := transPublisher.Publish(testEvent1)
	require.NoError(t, err)
	err = transPublisher.Publish(testEvent2)
	require.NoError(t, err)

	// Events should not be published yet
	assert.Len(t, mockPublisher.PublishedEvents, 0)

	// Flush
	err = transPublisher.Flush(context.Background())
	require.NoError(t, err)

	// Verify both events were published
	assert.Len(t, mockPublisher.PublishedEvents, 2)
	assert.Equal(t, testEvent1, mockPublisher.PublishedEvents[0])
	assert.Equal(t, testEvent2, mockPublisher.PublishedEvents[1])
}

func TestNATSTransactionalPublisher_Discard(t *testing.T) {
	// Create mock publisher
	mockPublisher := &MockEventPublisher{
		PublishedEvents: make([]events.Event, 0),
	}

	// Create transactional publisher
	transPublisher := NewNATSTransactionalPublisher(mockPublisher).(*NATSTransactionalPublisher)

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

	// Event should be buffered
	assert.Len(t, mockPublisher.PublishedEvents, 0)

	// Discard instead of flush
	transPublisher.Discard()

	// Verify event was NOT published
	assert.Len(t, mockPublisher.PublishedEvents, 0)

	// Attempting to flush after discard should publish nothing
	err = transPublisher.Flush(context.Background())
	require.NoError(t, err)
	assert.Len(t, mockPublisher.PublishedEvents, 0)
}
