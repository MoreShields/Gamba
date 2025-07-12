package events

import (
	"context"
	"sync"

	"gambler/models"
	log "github.com/sirupsen/logrus"
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
	GuildID         int64
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
	
	log.WithFields(log.Fields{
		"eventType":    eventType,
		"handlerCount": len(b.handlers[eventType]),
	}).Debug("Subscribed handler to event type on main event bus")
}

// Emit publishes an event to all registered handlers
func (b *Bus) Emit(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[event.Type()]))
	copy(handlers, b.handlers[event.Type()])
	b.mu.RUnlock()

	log.WithFields(log.Fields{
		"eventType":    event.Type(),
		"handlerCount": len(handlers),
	}).Debug("Emitting event to handlers on main event bus")

	// Call handlers asynchronously to avoid blocking
	for i, handler := range handlers {
		go func(h Handler, handlerIndex int) {
			log.WithFields(log.Fields{
				"eventType":    event.Type(),
				"handlerIndex": handlerIndex,
			}).Debug("Calling event handler")
			defer func() {
				if r := recover(); r != nil {
					log.WithFields(log.Fields{
						"eventType":    event.Type(),
						"handlerIndex": handlerIndex,
						"panic":        r,
					}).Error("Event handler panicked")
				}
			}()
			h(ctx, event)
		}(handler, i)
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
	log.WithFields(log.Fields{
		"eventType":    e.Type(),
		"pendingCount": len(b.pending),
	}).Debug("Adding event to transactional bus pending queue")
	b.pending = append(b.pending, e)
}

// called after successful DB commit
func (b *TransactionalBus) Flush(ctx context.Context) error {
	log.WithFields(log.Fields{
		"pendingEventCount": len(b.pending),
	}).Debug("Flushing pending events from transactional bus to main event bus")
	
	// Use background context for event emission to avoid issues with transaction context expiration
	// Events should be processed independently of the transaction lifecycle
	eventCtx := context.Background()
	
	for _, ev := range b.pending {
		log.WithFields(log.Fields{
			"eventType": ev.Type(),
		}).Debug("Emitting event to main event bus")
		b.real.Emit(eventCtx, ev)
	}
	b.pending = nil
	log.Debug("All pending events flushed, transactional bus cleared")
	return nil
}

// called after db rollback or to clear state.
func (b *TransactionalBus) Discard() {
	b.pending = nil
}
