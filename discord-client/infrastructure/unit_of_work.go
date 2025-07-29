package infrastructure

import (
	"context"

	"gambler/discord-client/application"
	"gambler/discord-client/service"
)

// unitOfWork wraps the repository UnitOfWork and adds event publishing on commit
type unitOfWork struct {
	inner                  application.UnitOfWork
	transactionalPublisher *NATSTransactionalPublisher
	ctx                    context.Context
}

// Begin starts a new transaction
func (u *unitOfWork) Begin(ctx context.Context) error {
	u.ctx = ctx
	return u.inner.Begin(ctx)
}

// Commit commits the transaction and flushes events on success
func (u *unitOfWork) Commit() error {
	// First commit the database transaction
	if err := u.inner.Commit(); err != nil {
		return err
	}

	// Then flush pending events after successful commit
	if u.transactionalPublisher != nil {
		// Note: We don't return errors from event publishing since the database
		// transaction has already committed. Events are best-effort after commit.
		_ = u.transactionalPublisher.Flush(u.ctx)
	}

	return nil
}

// Rollback rolls back the transaction and discards pending events
func (u *unitOfWork) Rollback() error {
	// Discard pending events
	if u.transactionalPublisher != nil {
		u.transactionalPublisher.Discard()
	}

	// Then rollback the database transaction
	return u.inner.Rollback()
}

// Repository getters - delegate to inner UnitOfWork
func (u *unitOfWork) UserRepository() service.UserRepository {
	return u.inner.UserRepository()
}

func (u *unitOfWork) BalanceHistoryRepository() service.BalanceHistoryRepository {
	return u.inner.BalanceHistoryRepository()
}

func (u *unitOfWork) BetRepository() service.BetRepository {
	return u.inner.BetRepository()
}

func (u *unitOfWork) WagerRepository() service.WagerRepository {
	return u.inner.WagerRepository()
}

func (u *unitOfWork) WagerVoteRepository() service.WagerVoteRepository {
	return u.inner.WagerVoteRepository()
}

func (u *unitOfWork) GroupWagerRepository() service.GroupWagerRepository {
	return u.inner.GroupWagerRepository()
}

func (u *unitOfWork) GuildSettingsRepository() service.GuildSettingsRepository {
	return u.inner.GuildSettingsRepository()
}

func (u *unitOfWork) SummonerWatchRepository() service.SummonerWatchRepository {
	return u.inner.SummonerWatchRepository()
}

// EventBus returns the transactional event publisher
func (u *unitOfWork) EventBus() service.EventPublisher {
	if u.transactionalPublisher == nil {
		panic("transactional publisher not configured")
	}
	return u.transactionalPublisher
}
