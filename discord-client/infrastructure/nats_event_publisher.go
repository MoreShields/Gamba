package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"gambler/discord-client/events"
	"gambler/discord-client/proto/models"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NATSEventPublisher implements the EventPublisher interface using NATS
type NATSEventPublisher struct {
	natsClient    *NATSClient
	subjectMapper *EventSubjectMapper
}

// NewNATSEventPublisher creates a new NATS event publisher
func NewNATSEventPublisher(natsClient *NATSClient, subjectMapper *EventSubjectMapper) *NATSEventPublisher {
	return &NATSEventPublisher{
		natsClient:    natsClient,
		subjectMapper: subjectMapper,
	}
}

// Publish publishes an event to NATS using the appropriate subject
func (p *NATSEventPublisher) Publish(event events.Event) error {
	ctx := context.Background()

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
		return fmt.Errorf("failed to publish event to NATS: %w", err)
	}

	log.WithFields(log.Fields{
		"eventType": event.Type(),
		"eventId":   envelope.EventId,
		"subject":   subject,
	}).Debug("Successfully published event to NATS")

	return nil
}

// EnsureDomainEventStream ensures the domain_events stream exists with the correct subjects
func (p *NATSEventPublisher) EnsureDomainEventStream() error {
	subjects := p.subjectMapper.GetAllSubjects()
	return p.natsClient.ensureStream("domain_events", subjects)
}
