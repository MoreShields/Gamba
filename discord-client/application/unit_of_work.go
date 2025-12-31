package application

import (
	"context"

	"gambler/discord-client/domain/interfaces"
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
	UserRepository() interfaces.UserRepository
	BalanceHistoryRepository() interfaces.BalanceHistoryRepository
	BetRepository() interfaces.BetRepository
	WagerRepository() interfaces.WagerRepository
	WagerVoteRepository() interfaces.WagerVoteRepository
	GroupWagerRepository() interfaces.GroupWagerRepository
	GuildSettingsRepository() interfaces.GuildSettingsRepository
	SummonerWatchRepository() interfaces.SummonerWatchRepository
	WordleCompletionRepo() interfaces.WordleCompletionRepository
	HighRollerPurchaseRepository() interfaces.HighRollerPurchaseRepository
	LotteryDrawRepository() interfaces.LotteryDrawRepository
	LotteryTicketRepository() interfaces.LotteryTicketRepository
	LotteryWinnerRepository() interfaces.LotteryWinnerRepository
	EventBus() interfaces.EventPublisher
}

// UnitOfWorkFactory defines the interface for creating UnitOfWork instances
type UnitOfWorkFactory interface {
	// CreateForGuild creates a new UnitOfWork instance scoped to a specific guild
	CreateForGuild(guildID int64) UnitOfWork
}
