package infrastructure

import (
	"context"
)

// MessagePublisher defines the interface for publishing messages to a message bus
type MessagePublisher interface {
	// Publish publishes a message to the specified subject
	Publish(ctx context.Context, subject string, data []byte) error
}
