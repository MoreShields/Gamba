package infrastructure

import (
	"gambler/discord-client/events"
)

// NoopEventPublisher is an event publisher that does nothing
// Useful for testing and admin commands where events should not be processed
type NoopEventPublisher struct{}

// NewNoopEventPublisher creates a new no-op event publisher
func NewNoopEventPublisher() *NoopEventPublisher {
	return &NoopEventPublisher{}
}

// Publish does nothing with the event
func (n *NoopEventPublisher) Publish(event events.Event) error {
	// No-op
	return nil
}
