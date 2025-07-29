package repository

import (
	"gambler/discord-client/application"
	"gambler/discord-client/database"
	"gambler/discord-client/service"
)

// NewTestUnitOfWorkFactory creates a unit of work factory for tests
// Tests should provide their own transactional publisher mock
func NewTestUnitOfWorkFactory(db *database.DB) *unitOfWorkFactory {
	return NewUnitOfWorkFactory(db)
}

// CreateTestUnitOfWork creates a unit of work for testing with the provided transactional publisher
func CreateTestUnitOfWork(db *database.DB, guildID int64, transactionalPublisher service.TransactionalEventPublisher) application.UnitOfWork {
	factory := NewTestUnitOfWorkFactory(db)
	return factory.CreateForGuildWithPublisher(guildID, transactionalPublisher)
}