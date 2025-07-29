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
	localHandlers map[events.EventType][]func(context.Context, events.Event) error
}

// NewNATSTransactionalPublisher creates a new transactional publisher
func NewNATSTransactionalPublisher(realPublisher service.EventPublisher) service.EventPublisher {
	return &NATSTransactionalPublisher{
		realPublisher: realPublisher,
		pending:       make([]events.Event, 0),
		localHandlers: make(map[events.EventType][]func(context.Context, events.Event) error),
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

// Flush publishes all pending events to NATS and invokes local handlers
// This should be called after successful database transaction commit
func (p *NATSTransactionalPublisher) Flush(ctx context.Context) error {
	log.WithFields(log.Fields{
		"pendingEventCount": len(p.pending),
	}).Debug("Flushing pending events from NATS transactional publisher")

	// Process all pending events
	for _, event := range p.pending {
		eventType := event.Type()

		// First, invoke any local handlers for this event type
		if handlers, exists := p.localHandlers[eventType]; exists {
			for _, handler := range handlers {
				log.WithFields(log.Fields{
					"eventType": eventType,
				}).Debug("Invoking local handler for event")

				if err := handler(ctx, event); err != nil {
					log.WithFields(log.Fields{
						"eventType": eventType,
						"error":     err,
					}).Error("Local event handler failed")
					// Continue processing - local handler errors shouldn't stop other handlers or NATS publishing
				}
			}
		}

		// Then publish to NATS
		log.WithFields(log.Fields{
			"eventType": eventType,
		}).Debug("Publishing event to NATS")

		if err := p.realPublisher.Publish(event); err != nil {
			// Log error but continue with other events
			// This ensures partial failure doesn't block all events
			log.WithFields(log.Fields{
				"eventType": eventType,
				"error":     err,
			}).Error("Failed to publish event to NATS during flush")
		}
	}

	// Clear the pending queue
	p.pending = p.pending[:0]
	log.Debug("All pending events flushed (local handlers + NATS), transactional publisher cleared")

	return nil
}

// RegisterLocalHandler registers a handler that will be invoked locally during flush
// This allows handling events in the same process that publishes them
func (p *NATSTransactionalPublisher) RegisterLocalHandler(eventType events.EventType, handler func(context.Context, events.Event) error) {
	p.localHandlers[eventType] = append(p.localHandlers[eventType], handler)
	log.WithFields(log.Fields{
		"eventType":    eventType,
		"handlerCount": len(p.localHandlers[eventType]),
	}).Debug("Registered local event handler")
}

// Discard clears all pending events without publishing them
// This should be called on database transaction rollback
func (p *NATSTransactionalPublisher) Discard() {
	log.WithFields(log.Fields{
		"discardedEventCount": len(p.pending),
	}).Debug("Discarding pending events from NATS transactional publisher")

	p.pending = p.pending[:0]
}
