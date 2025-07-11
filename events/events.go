package events

import (
	"context"
	"sync"

	"gambler/models"
)

// EventType represents different types of events in the system
type EventType string

const (
	EventTypeBalanceChange       EventType = "balance_change"
	EventTypeUserCreated         EventType = "user_created"
	EventTypeBetPlaced           EventType = "bet_placed"
	EventTypeWagerResolved       EventType = "wager_resolved"
	EventTypeGroupWagerStateChange EventType = "group_wager_state_change"
)

// Event is the base interface for all events
type Event interface {
	Type() EventType
}

// BalanceChangeEvent represents a balance change that occurred
type BalanceChangeEvent struct {
	UserID          int64
	OldBalance      int64
	NewBalance      int64
	TransactionType models.TransactionType
	ChangeAmount    int64
}

func (e BalanceChangeEvent) Type() EventType {
	return EventTypeBalanceChange
}

// UserCreatedEvent represents a new user creation
type UserCreatedEvent struct {
	UserID         int64
	DiscordID      int64
	Username       string
	InitialBalance int64
}

func (e UserCreatedEvent) Type() EventType {
	return EventTypeUserCreated
}

// BetPlacedEvent represents a bet that was placed
type BetPlacedEvent struct {
	UserID int64
	BetID  int64
	Amount int64
	Won    bool
	Payout int64
}

func (e BetPlacedEvent) Type() EventType {
	return EventTypeBetPlaced
}

// WagerResolvedEvent represents a wager that was resolved
type WagerResolvedEvent struct {
	WagerID  int64
	WinnerID int64
	LoserID  int64
	Amount   int64
}

func (e WagerResolvedEvent) Type() EventType {
	return EventTypeWagerResolved
}

// GroupWagerStateChangeEvent represents a group wager state transition
type GroupWagerStateChangeEvent struct {
	GroupWagerID int64
	OldState     string
	NewState     string
	MessageID    int64
	ChannelID    int64
}

func (e GroupWagerStateChangeEvent) Type() EventType {
	return EventTypeGroupWagerStateChange
}

// Handler is a function that handles events
type Handler func(ctx context.Context, event Event)

// Bus manages event subscriptions and dispatching
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe adds a handler for a specific event type
func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlers[eventType] == nil {
		b.handlers[eventType] = make([]Handler, 0)
	}
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Emit publishes an event to all registered handlers
func (b *Bus) Emit(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[event.Type()]))
	copy(handlers, b.handlers[event.Type()])
	b.mu.RUnlock()

	// Call handlers asynchronously to avoid blocking
	for _, handler := range handlers {
		go func(h Handler) {
			h(ctx, event)
		}(handler)
	}
}

// A transactional event bus for holding pending events coupled to the Unit of Work.
// Flushes to the underlying event bus.
type TransactionalBus struct {
	real    *Bus
	pending []Event // stashed until Flush
}

func NewTransactionalBus(real *Bus) *TransactionalBus {
	return &TransactionalBus{real: real}
}

func (b *TransactionalBus) Publish(e Event) {
	b.pending = append(b.pending, e)
}

// called after successful DB commit
func (b *TransactionalBus) Flush(ctx context.Context) error {
	for _, ev := range b.pending {
		b.real.Emit(ctx, ev)
	}
	b.pending = nil
	return nil
}

// called after db rollback or to clear state.
func (b *TransactionalBus) Discard() {
	b.pending = nil
}
