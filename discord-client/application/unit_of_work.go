package application

import (
	"context"

	"gambler/discord-client/service"
)

// UnitOfWork defines the interface for transactional repository operations
type UnitOfWork interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) error

	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// Repository getters
	UserRepository() service.UserRepository
	BalanceHistoryRepository() service.BalanceHistoryRepository
	BetRepository() service.BetRepository
	WagerRepository() service.WagerRepository
	WagerVoteRepository() service.WagerVoteRepository
	GroupWagerRepository() service.GroupWagerRepository
	GuildSettingsRepository() service.GuildSettingsRepository
	SummonerWatchRepository() service.SummonerWatchRepository
	WordleCompletionRepo() service.WordleCompletionRepository
	EventBus() service.EventPublisher
}

// UnitOfWorkFactory defines the interface for creating UnitOfWork instances
type UnitOfWorkFactory interface {
	// CreateForGuild creates a new UnitOfWork instance scoped to a specific guild
	CreateForGuild(guildID int64) UnitOfWork
}
