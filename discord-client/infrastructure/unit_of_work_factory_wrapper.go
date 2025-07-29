package infrastructure

import (
	"context"
	
	"gambler/discord-client/application"
	"gambler/discord-client/database"
	"gambler/discord-client/events"
	"gambler/discord-client/repository"
	"gambler/discord-client/service"
)

// UnitOfWorkFactoryWrapper wraps the repository UnitOfWorkFactory to provide transactional publishers
type UnitOfWorkFactoryWrapper struct {
	repoFactory    interface {
		CreateForGuildWithPublisher(guildID int64, transactionalPublisher service.TransactionalEventPublisher) application.UnitOfWork
	}
	eventPublisher service.EventPublisher
	localHandlers  map[events.EventType][]func(context.Context, events.Event) error
}

// NewUnitOfWorkFactoryWrapper creates a new wrapper that implements service.UnitOfWorkFactory
func NewUnitOfWorkFactoryWrapper(db *database.DB, eventPublisher service.EventPublisher) *UnitOfWorkFactoryWrapper {
	repoFactory := repository.NewUnitOfWorkFactory(db)
	return &UnitOfWorkFactoryWrapper{
		repoFactory:    repoFactory,
		eventPublisher: eventPublisher,
		localHandlers:  make(map[events.EventType][]func(context.Context, events.Event) error),
	}
}

// RegisterLocalHandler registers a handler that will be invoked locally for events
// This ensures events published within the same process are handled immediately
func (w *UnitOfWorkFactoryWrapper) RegisterLocalHandler(eventType events.EventType, handler func(context.Context, events.Event) error) {
	w.localHandlers[eventType] = append(w.localHandlers[eventType], handler)
}

// CreateForGuild creates a new UnitOfWork with a transactional event publisher
func (w *UnitOfWorkFactoryWrapper) CreateForGuild(guildID int64) application.UnitOfWork {
	// Create a transactional publisher for this unit of work
	transactionalPublisher := NewNATSTransactionalPublisher(w.eventPublisher).(*NATSTransactionalPublisher)
	
	// Register all local handlers on this publisher instance
	for eventType, handlers := range w.localHandlers {
		for _, handler := range handlers {
			transactionalPublisher.RegisterLocalHandler(eventType, handler)
		}
	}
	
	// Create the unit of work with the transactional publisher
	return w.repoFactory.CreateForGuildWithPublisher(guildID, transactionalPublisher)
}