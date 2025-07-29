package infrastructure

import (
	"context"

	"gambler/discord-client/events"
	"gambler/discord-client/service"

	log "github.com/sirupsen/logrus"
)

// NATSTransactionalPublisher holds events until flush, then publishes to NATS
// This replaces the TransactionalBus pattern for maintaining consistency with database transactions
type NATSTransactionalPublisher struct {
	realPublisher service.EventPublisher
	pending       []events.Event
}

// NewNATSTransactionalPublisher creates a new transactional publisher
func NewNATSTransactionalPublisher(realPublisher service.EventPublisher) service.EventPublisher {
	return &NATSTransactionalPublisher{
		realPublisher: realPublisher,
		pending:       make([]events.Event, 0),
	}
}

// Publish stores an event in the pending queue without immediately publishing
func (p *NATSTransactionalPublisher) Publish(event events.Event) error {
	log.WithFields(log.Fields{
		"eventType":    event.Type(),
		"pendingCount": len(p.pending),
	}).Debug("Adding event to NATS transactional publisher pending queue")

	p.pending = append(p.pending, event)
	return nil
}

// Flush publishes all pending events to the real publisher
// This should be called after successful database transaction commit
func (p *NATSTransactionalPublisher) Flush(ctx context.Context) error {
	log.WithFields(log.Fields{
		"pendingEventCount": len(p.pending),
	}).Debug("Flushing pending events from NATS transactional publisher")

	// Process all pending events
	for _, event := range p.pending {
		eventType := event.Type()

		log.WithFields(log.Fields{
			"eventType": eventType,
		}).Debug("Publishing event via real publisher")

		if err := p.realPublisher.Publish(event); err != nil {
			// Log error but continue with other events
			// This ensures partial failure doesn't block all events
			log.WithFields(log.Fields{
				"eventType": eventType,
				"error":     err,
			}).Error("Failed to publish event during flush")
		}
	}

	// Clear the pending queue
	p.pending = p.pending[:0]
	log.Debug("All pending events flushed to real publisher, transactional publisher cleared")

	return nil
}

// Discard clears all pending events without publishing them
// This should be called on database transaction rollback
func (p *NATSTransactionalPublisher) Discard() {
	log.WithFields(log.Fields{
		"discardedEventCount": len(p.pending),
	}).Debug("Discarding pending events from NATS transactional publisher")

	p.pending = p.pending[:0]
}
