package repository

import (
	"context"
	"fmt"

	"gambler/database"
	"gambler/events"
	"gambler/service"
	"github.com/jackc/pgx/v5"
)

// unitOfWork implements the UnitOfWork interface
type unitOfWork struct {
	db                          *database.DB
	tx                          pgx.Tx
	ctx                         context.Context
	transactionalBus            *events.TransactionalBus
	userRepo                    service.UserRepository
	balanceHistoryRepo          service.BalanceHistoryRepository
	betRepo                     service.BetRepository
	wagerRepo                   service.WagerRepository
	wagerVoteRepo               service.WagerVoteRepository
	groupWagerRepo              service.GroupWagerRepository
	guildSettingsRepo           service.GuildSettingsRepository
}

// NewUnitOfWorkFactory creates a new UnitOfWork factory
func NewUnitOfWorkFactory(db *database.DB, eventBus *events.Bus) service.UnitOfWorkFactory {
	return &unitOfWorkFactory{
		db:       db,
		eventBus: eventBus,
	}
}

type unitOfWorkFactory struct {
	db       *database.DB
	eventBus *events.Bus
}

func (f *unitOfWorkFactory) Create() service.UnitOfWork {
	return &unitOfWork{
		db:               f.db,
		transactionalBus: events.NewTransactionalBus(f.eventBus),
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

	// Create repositories with the transaction
	u.userRepo = newUserRepositoryWithTx(tx)
	u.balanceHistoryRepo = newBalanceHistoryRepositoryWithTx(tx)
	u.betRepo = newBetRepositoryWithTx(tx)
	u.wagerRepo = newWagerRepositoryWithTx(tx)
	u.wagerVoteRepo = newWagerVoteRepositoryWithTx(tx)
	u.groupWagerRepo = newGroupWagerRepositoryWithTx(tx)
	u.guildSettingsRepo = newGuildSettingsRepositoryWithTx(tx)

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
	if u.transactionalBus != nil {
		u.transactionalBus.Flush(u.ctx)
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
	if u.transactionalBus != nil {
		u.transactionalBus.Discard()
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

// EventBus returns the transactional event bus for this unit of work
func (u *unitOfWork) EventBus() service.EventPublisher {
	if u.transactionalBus == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.transactionalBus
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