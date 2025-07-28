package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"gambler/discord-client/events"
	"gambler/discord-client/proto/models"

	log "github.com/sirupsen/logrus"
)


// NATSEventSubscriber subscribes to NATS subjects and deserializes events for application handlers
type NATSEventSubscriber struct {
	natsClient    *NATSClient
	subjectMapper *EventSubjectMapper
	handlers      map[string]func(context.Context, events.Event) error
}

// NewNATSEventSubscriber creates a new NATS event subscriber
func NewNATSEventSubscriber(natsClient *NATSClient, subjectMapper *EventSubjectMapper) *NATSEventSubscriber {
	return &NATSEventSubscriber{
		natsClient:    natsClient,
		subjectMapper: subjectMapper,
		handlers:      make(map[string]func(context.Context, events.Event) error),
	}
}

// Subscribe registers a handler for a specific event type
func (s *NATSEventSubscriber) Subscribe(eventType events.EventType, handler func(context.Context, events.Event) error) error {
	subject := s.mapEventTypeToSubject(eventType)
	s.handlers[subject] = handler
	
	log.WithFields(log.Fields{
		"eventType": eventType,
		"subject":   subject,
		"handler":   fmt.Sprintf("%T", handler),
	}).Info("Registering event handler for subject")
	
	// Subscribe to NATS subject with message handling wrapper
	return s.natsClient.Subscribe(subject, func(data []byte) error {
		return s.handleMessage(subject, data)
	})
}

// handleMessage deserializes a NATS message and routes it to the appropriate handler
func (s *NATSEventSubscriber) handleMessage(subject string, data []byte) error {
	// Deserialize event envelope
	var envelope models.EventEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		log.WithFields(log.Fields{
			"subject": subject,
			"error":   err,
		}).Error("Failed to unmarshal event envelope")
		return fmt.Errorf("failed to unmarshal event envelope: %w", err)
	}
	
	// Get event type from envelope
	eventType := events.EventType(envelope.EventType)
	
	// Deserialize event payload based on type
	event, err := s.deserializeEvent(eventType, envelope.Payload)
	if err != nil {
		log.WithFields(log.Fields{
			"subject":           subject,
			"eventType":         eventType,
			"envelopeEventType": envelope.EventType,
			"eventId":           envelope.EventId,
			"error":             err,
			"payloadSize":       len(envelope.Payload),
		}).Error("Failed to deserialize event payload")
		return fmt.Errorf("failed to deserialize event payload: %w", err)
	}
	
	// Get handler for this subject
	handler, exists := s.handlers[subject]
	if !exists {
		log.WithFields(log.Fields{
			"subject":   subject,
			"eventType": eventType,
		}).Warn("No handler registered for subject")
		return fmt.Errorf("no handler registered for subject %s", subject)
	}
	
	// Call the handler
	ctx := context.Background()
	log.WithFields(log.Fields{
		"subject":   subject,
		"eventType": eventType,
		"eventId":   envelope.EventId,
	}).Debug("Calling event handler for NATS message")
	
	if err := handler(ctx, event); err != nil {
		log.WithFields(log.Fields{
			"subject":   subject,
			"eventType": eventType,
			"eventId":   envelope.EventId,
			"error":     err,
			"handler":   fmt.Sprintf("%T", handler),
			"eventData": fmt.Sprintf("%+v", event),
		}).Error("Event handler failed")
		return err
	}
	
	log.WithFields(log.Fields{
		"subject":   subject,
		"eventType": eventType,
		"eventId":   envelope.EventId,
	}).Debug("Successfully processed NATS event")
	
	return nil
}

// deserializeEvent deserializes the event payload based on event type
func (s *NATSEventSubscriber) deserializeEvent(eventType events.EventType, payload []byte) (events.Event, error) {
	var event events.Event
	
	switch eventType {
	case events.EventTypeGroupWagerStateChange:
		event = &events.GroupWagerStateChangeEvent{}
	case events.EventTypeBalanceChange:
		event = &events.BalanceChangeEvent{}
	case events.EventTypeUserCreated:
		event = &events.UserCreatedEvent{}
	case events.EventTypeBetPlaced:
		event = &events.BetPlacedEvent{}
	case events.EventTypeWagerResolved:
		event = &events.WagerResolvedEvent{}
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
	
	if err := json.Unmarshal(payload, event); err != nil {
		return nil, err
	}
	
	return event, nil
}

// mapEventTypeToSubject maps an event type to its NATS subject
func (s *NATSEventSubscriber) mapEventTypeToSubject(eventType events.EventType) string {
	// Create a dummy event to get its subject mapping
	var event events.Event
	switch eventType {
	case events.EventTypeGroupWagerStateChange:
		event = events.GroupWagerStateChangeEvent{}
	case events.EventTypeBalanceChange:
		event = events.BalanceChangeEvent{}
	case events.EventTypeUserCreated:
		event = events.UserCreatedEvent{}
	case events.EventTypeBetPlaced:
		event = events.BetPlacedEvent{}
	case events.EventTypeWagerResolved:
		event = events.WagerResolvedEvent{}
	default:
		return fmt.Sprintf("unknown.%s", eventType)
	}
	
	return s.subjectMapper.MapEventToSubject(event)
}