package infrastructure

import (
	"gambler/discord-client/database"
	"gambler/discord-client/repository"
	"gambler/discord-client/service"
)

// UnitOfWorkFactoryWrapper wraps the repository UnitOfWorkFactory to provide transactional publishers
type UnitOfWorkFactoryWrapper struct {
	repoFactory    interface {
		CreateForGuildWithPublisher(guildID int64, transactionalPublisher service.TransactionalEventPublisher) service.UnitOfWork
	}
	eventPublisher service.EventPublisher
}

// NewUnitOfWorkFactoryWrapper creates a new wrapper that implements service.UnitOfWorkFactory
func NewUnitOfWorkFactoryWrapper(db *database.DB, eventPublisher service.EventPublisher) service.UnitOfWorkFactory {
	repoFactory := repository.NewUnitOfWorkFactory(db)
	return &UnitOfWorkFactoryWrapper{
		repoFactory:    repoFactory,
		eventPublisher: eventPublisher,
	}
}

// CreateForGuild creates a new UnitOfWork with a transactional event publisher
func (w *UnitOfWorkFactoryWrapper) CreateForGuild(guildID int64) service.UnitOfWork {
	// Create a transactional publisher for this unit of work
	transactionalPublisher := NewNATSTransactionalPublisher(w.eventPublisher).(service.TransactionalEventPublisher)
	
	// Create the unit of work with the transactional publisher
	return w.repoFactory.CreateForGuildWithPublisher(guildID, transactionalPublisher)
}