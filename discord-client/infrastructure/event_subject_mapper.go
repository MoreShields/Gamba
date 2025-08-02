package infrastructure

import (
	"fmt"

	"gambler/discord-client/domain/events"
)

// EventSubjectMapper handles mapping between domain events and NATS subjects
type EventSubjectMapper struct{}

// NewEventSubjectMapper creates a new event subject mapper
func NewEventSubjectMapper() *EventSubjectMapper {
	return &EventSubjectMapper{}
}

// MapEventToSubject converts a domain event to its corresponding NATS subject
func (m *EventSubjectMapper) MapEventToSubject(event events.Event) string {
	switch event.Type() {
	case events.EventTypeGroupWagerStateChange:
		return "wagers.group.state_changed"
	case events.EventTypeBalanceChange:
		return "users.balance_changed"
	case events.EventTypeUserCreated:
		return "users.created"
	case events.EventTypeBetPlaced:
		return "betting.placed"
	case events.EventTypeWagerResolved:
		return "wagers.individual.resolved"
	default:
		// Fallback for unknown event types
		return fmt.Sprintf("unknown.%s", event.Type())
	}
}

// MapSubjectToEventType converts a NATS subject back to an event type
func (m *EventSubjectMapper) MapSubjectToEventType(subject string) events.EventType {
	switch subject {
	case "wagers.group.state_changed":
		return events.EventTypeGroupWagerStateChange
	case "users.balance_changed":
		return events.EventTypeBalanceChange
	case "users.created":
		return events.EventTypeUserCreated
	case "betting.placed":
		return events.EventTypeBetPlaced
	case "wagers.individual.resolved":
		return events.EventTypeWagerResolved
	default:
		return events.EventType(subject)
	}
}

// GetAllSubjects returns all subjects that this service publishes to
func (m *EventSubjectMapper) GetAllSubjects() []string {
	return []string{
		"wagers.group.state_changed",
		"users.balance_changed",
		"users.created",
		"betting.placed",
		"wagers.individual.resolved",
	}
}
