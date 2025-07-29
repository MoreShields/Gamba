package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/application"
	"gambler/discord-client/database"
	"gambler/discord-client/service"
	"github.com/jackc/pgx/v5"
)

// unitOfWork implements the UnitOfWork interface
type unitOfWork struct {
	db                          *database.DB
	tx                          pgx.Tx
	ctx                         context.Context
	guildID                     int64
	transactionalPublisher      service.TransactionalEventPublisher
	userRepo                    service.UserRepository
	balanceHistoryRepo          service.BalanceHistoryRepository
	betRepo                     service.BetRepository
	wagerRepo                   service.WagerRepository
	wagerVoteRepo               service.WagerVoteRepository
	groupWagerRepo              service.GroupWagerRepository
	guildSettingsRepo           service.GuildSettingsRepository
	summonerWatchRepo           service.SummonerWatchRepository
}

// NewUnitOfWorkFactory creates a new UnitOfWork factory
func NewUnitOfWorkFactory(db *database.DB) *unitOfWorkFactory {
	return &unitOfWorkFactory{
		db: db,
	}
}

type unitOfWorkFactory struct {
	db *database.DB
}

// CreateForGuildWithPublisher creates a new UnitOfWork with a specific transactional publisher
func (f *unitOfWorkFactory) CreateForGuildWithPublisher(guildID int64, transactionalPublisher service.TransactionalEventPublisher) application.UnitOfWork {
	return &unitOfWork{
		db:                     f.db,
		guildID:                guildID,
		transactionalPublisher: transactionalPublisher,
	}
}

// Begin starts a new transaction
func (u *unitOfWork) Begin(ctx context.Context) error {
	if u.tx != nil {
		return fmt.Errorf("transaction already started")
	}

	tx, err := u.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	u.tx = tx
	u.ctx = ctx

	// Create guild-scoped repositories with the transaction
	u.userRepo = newUserRepository(tx, u.guildID)
	u.balanceHistoryRepo = newBalanceHistoryRepository(tx, u.guildID)
	u.betRepo = newBetRepository(tx, u.guildID)
	u.wagerRepo = newWagerRepository(tx, u.guildID)
	u.wagerVoteRepo = newWagerVoteRepository(tx, u.guildID)
	u.groupWagerRepo = newGroupWagerRepository(tx, u.guildID)
	u.guildSettingsRepo = newGuildSettingsRepositoryWithTx(tx) // Guild settings don't need scoping
	u.summonerWatchRepo = newSummonerWatchRepository(tx, u.guildID)

	return nil
}

// Commit commits the transaction
func (u *unitOfWork) Commit() error {
	if u.tx == nil {
		return fmt.Errorf("no transaction to commit")
	}

	err := u.tx.Commit(u.ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	u.tx = nil

	// Flush pending events after successful commit
	if u.transactionalPublisher != nil {
		u.transactionalPublisher.Flush(u.ctx)
	}

	return nil
}

// Rollback rolls back the transaction
func (u *unitOfWork) Rollback() error {
	if u.tx == nil {
		return nil // Nothing to rollback
	}

	err := u.tx.Rollback(u.ctx)
	if err != nil && err != pgx.ErrTxClosed {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	u.tx = nil

	// Discard pending events on rollback
	if u.transactionalPublisher != nil {
		u.transactionalPublisher.Discard()
	}

	return nil
}

// UserRepository returns the user repository for this unit of work
func (u *unitOfWork) UserRepository() service.UserRepository {
	if u.userRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.userRepo
}

// BalanceHistoryRepository returns the balance history repository for this unit of work
func (u *unitOfWork) BalanceHistoryRepository() service.BalanceHistoryRepository {
	if u.balanceHistoryRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.balanceHistoryRepo
}

// BetRepository returns the bet repository for this unit of work
func (u *unitOfWork) BetRepository() service.BetRepository {
	if u.betRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.betRepo
}

// WagerRepository returns the wager repository for this unit of work
func (u *unitOfWork) WagerRepository() service.WagerRepository {
	if u.wagerRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.wagerRepo
}

// WagerVoteRepository returns the wager vote repository for this unit of work
func (u *unitOfWork) WagerVoteRepository() service.WagerVoteRepository {
	if u.wagerVoteRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.wagerVoteRepo
}

// EventBus returns the transactional event publisher for this unit of work
func (u *unitOfWork) EventBus() service.EventPublisher {
	if u.transactionalPublisher == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.transactionalPublisher
}

// GroupWagerRepository returns the group wager repository for this unit of work
func (u *unitOfWork) GroupWagerRepository() service.GroupWagerRepository {
	if u.groupWagerRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.groupWagerRepo
}

// GuildSettingsRepository returns the guild settings repository for this unit of work
func (u *unitOfWork) GuildSettingsRepository() service.GuildSettingsRepository {
	if u.guildSettingsRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.guildSettingsRepo
}

// SummonerWatchRepository returns the summoner watch repository for this unit of work
func (u *unitOfWork) SummonerWatchRepository() service.SummonerWatchRepository {
	if u.summonerWatchRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.summonerWatchRepo
}