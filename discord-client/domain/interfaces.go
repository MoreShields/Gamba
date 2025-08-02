package domain

import (
	"context"

	"gambler/discord-client/domain/events"
)

// EventSubscriber is an interface for subscribing to domain events
// This allows the application layer to react to domain events without
// depending on the infrastructure implementation
type EventSubscriber interface {
	Subscribe(eventType events.EventType, handler func(context.Context, events.Event) error) error
}

// MessageHandler defines the interface for handling raw messages from infrastructure
// This allows infrastructure to process messages without knowing about application specifics
type MessageHandler interface {
	HandleMessage(ctx context.Context, subject string, data []byte) error
}
