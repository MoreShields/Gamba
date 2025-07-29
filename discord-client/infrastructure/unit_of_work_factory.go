package infrastructure

import (
	"context"

	"gambler/discord-client/application"
	"gambler/discord-client/database"
	"gambler/discord-client/events"
	"gambler/discord-client/service"
)

// UnitOfWorkFactory implements the application.UnitOfWorkFactory interface
// It creates UnitOfWork instances that handle both database transactions and event publishing
type UnitOfWorkFactory struct {
	db             *database.DB
	eventPublisher service.EventPublisher
}

// NewUnitOfWorkFactory creates a new UnitOfWorkFactory
func NewUnitOfWorkFactory(db *database.DB, eventPublisher service.EventPublisher) *UnitOfWorkFactory {
	return &UnitOfWorkFactory{
		db:             db,
		eventPublisher: eventPublisher,
	}
}

// RegisterLocalHandler registers a handler that will be invoked locally for events
// This ensures events published within the same process are handled immediately
func (f *UnitOfWorkFactory) RegisterLocalHandler(eventType events.EventType, handler func(context.Context, events.Event) error) {
	// Register directly with the NATSEventPublisher
	if natsPublisher, ok := f.eventPublisher.(*NATSEventPublisher); ok {
		natsPublisher.RegisterLocalHandler(eventType, handler)
	}
}

// CreateForGuild creates a new UnitOfWork with integrated event publishing
func (f *UnitOfWorkFactory) CreateForGuild(guildID int64) application.UnitOfWork {
	return &unitOfWork{
		db:             f.db,
		guildID:        guildID,
		eventPublisher: f.eventPublisher,
	}
}
