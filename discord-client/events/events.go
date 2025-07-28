package events

import "gambler/discord-client/models"

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
	GuildID      int64
	OldState     string
	NewState     string
	MessageID    int64
	ChannelID    int64
}

func (e GroupWagerStateChangeEvent) Type() EventType {
	return EventTypeGroupWagerStateChange
}