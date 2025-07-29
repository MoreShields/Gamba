package infrastructure

import (
	"gambler/discord-client/application"
	"gambler/discord-client/database"
	"gambler/discord-client/repository"
	"gambler/discord-client/service"
)

// TestUnitOfWorkFactory is a test factory that creates new unit of work instances
// This is placed in infrastructure package to avoid circular dependencies between
// application and repository packages
type TestUnitOfWorkFactory struct {
	db                     *database.DB
	transactionalPublisher service.TransactionalEventPublisher
}

// NewTestUnitOfWorkFactory creates a new test unit of work factory
func NewTestUnitOfWorkFactory(db *database.DB, transactionalPublisher service.TransactionalEventPublisher) *TestUnitOfWorkFactory {
	return &TestUnitOfWorkFactory{
		db:                     db,
		transactionalPublisher: transactionalPublisher,
	}
}

// CreateForGuild creates a new UnitOfWork instance for testing
func (f *TestUnitOfWorkFactory) CreateForGuild(guildID int64) application.UnitOfWork {
	// Create a fresh UoW for each call to avoid transaction state issues
	return repository.CreateTestUnitOfWork(f.db, guildID, f.transactionalPublisher)
}