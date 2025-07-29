package infrastructure

import (
	"context"

	"gambler/discord-client/application"
	"gambler/discord-client/database"
	"gambler/discord-client/events"
	"gambler/discord-client/repository"
	"gambler/discord-client/service"
)

// UnitOfWorkFactory implements the application.UnitOfWorkFactory interface
// It creates UnitOfWork instances that handle both database transactions and event publishing
type UnitOfWorkFactory struct {
	repoFactory interface {
		CreateForGuild(guildID int64) application.UnitOfWork
	}
	eventPublisher service.EventPublisher
	localHandlers  map[events.EventType][]func(context.Context, events.Event) error
}

// NewUnitOfWorkFactory creates a new UnitOfWorkFactory
func NewUnitOfWorkFactory(db *database.DB, eventPublisher service.EventPublisher) *UnitOfWorkFactory {
	repoFactory := repository.NewUnitOfWorkFactory(db)
	return &UnitOfWorkFactory{
		repoFactory:    repoFactory,
		eventPublisher: eventPublisher,
		localHandlers:  make(map[events.EventType][]func(context.Context, events.Event) error),
	}
}

// RegisterLocalHandler registers a handler that will be invoked locally for events
// This ensures events published within the same process are handled immediately
func (f *UnitOfWorkFactory) RegisterLocalHandler(eventType events.EventType, handler func(context.Context, events.Event) error) {
	f.localHandlers[eventType] = append(f.localHandlers[eventType], handler)
}

// CreateForGuild creates a new UnitOfWork with a transactional event publisher
func (f *UnitOfWorkFactory) CreateForGuild(guildID int64) application.UnitOfWork {
	// Create a transactional publisher for this unit of work
	transactionalPublisher := NewNATSTransactionalPublisher(f.eventPublisher).(*NATSTransactionalPublisher)

	// Register all local handlers on this publisher instance
	for eventType, handlers := range f.localHandlers {
		for _, handler := range handlers {
			transactionalPublisher.RegisterLocalHandler(eventType, handler)
		}
	}

	// Create the repository unit of work
	repoUow := f.repoFactory.CreateForGuild(guildID)

	// Wrap it with our infrastructure unit of work that handles event flushing
	return &unitOfWork{
		inner:                  repoUow,
		transactionalPublisher: transactionalPublisher,
	}
}
