package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gambler/discord-client/domain/events"
	"gambler/discord-client/proto/models"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NATSEventPublisher implements the EventPublisher interface using NATS
type NATSEventPublisher struct {
	natsClient    *NATSClient
	subjectMapper *EventSubjectMapper
	localHandlers map[events.EventType][]func(context.Context, events.Event) error
}

// NewNATSEventPublisher creates a new NATS event publisher
func NewNATSEventPublisher(natsClient *NATSClient, subjectMapper *EventSubjectMapper) *NATSEventPublisher {
	return &NATSEventPublisher{
		natsClient:    natsClient,
		subjectMapper: subjectMapper,
		localHandlers: make(map[events.EventType][]func(context.Context, events.Event) error),
	}
}

// Publish publishes an event to NATS using the appropriate subject
func (p *NATSEventPublisher) Publish(event events.Event) error {
	ctx := context.Background()
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

	// Map event to subject
	subject := p.subjectMapper.MapEventToSubject(event)

	// Serialize event payload
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create event envelope
	envelope := &models.EventEnvelope{
		EventId:       uuid.New().String(),
		EventType:     string(event.Type()),
		Timestamp:     timestamppb.Now(),
		SourceService: "discord-client",
		Payload:       payload,
	}

	// Serialize envelope
	envelopeData, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event envelope: %w", err)
	}

	// Publish to NATS
	if err := p.natsClient.Publish(ctx, subject, envelopeData); err != nil {
		if strings.Contains(err.Error(), "no response from stream") {
			return nil
		}
		return fmt.Errorf("failed to publish event to NATS: %w", err)
	}

	log.WithFields(log.Fields{
		"eventType": event.Type(),
		"eventId":   envelope.EventId,
		"subject":   subject,
	}).Debug("Successfully published event to NATS")

	return nil
}

// RegisterLocalHandler registers a handler that will be invoked locally for events
// This allows handling events in the same process that publishes them
func (p *NATSEventPublisher) RegisterLocalHandler(eventType events.EventType, handler func(context.Context, events.Event) error) {
	p.localHandlers[eventType] = append(p.localHandlers[eventType], handler)
	log.WithFields(log.Fields{
		"eventType":    eventType,
		"handlerCount": len(p.localHandlers[eventType]),
	}).Info("Registered local event handler")
}

// EnsureDomainEventStream ensures the domain_events stream exists with the correct subjects
func (p *NATSEventPublisher) EnsureDomainEventStream() error {
	subjects := p.subjectMapper.GetAllSubjects()
	return p.natsClient.ensureStream("domain_events", subjects)
}
