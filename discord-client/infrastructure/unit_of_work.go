package infrastructure

import (
	"context"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/repository"
	"gambler/discord-client/domain/interfaces"

	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
)

// unitOfWork implements the UnitOfWork interface with integrated event publishing
type unitOfWork struct {
	db                       *database.DB
	tx                       pgx.Tx
	ctx                      context.Context
	guildID                  int64
	eventPublisher           interfaces.EventPublisher
	pendingEvents            []events.Event
	userRepo                 interfaces.UserRepository
	balanceHistoryRepo       interfaces.BalanceHistoryRepository
	betRepo                  interfaces.BetRepository
	wagerRepo                interfaces.WagerRepository
	wagerVoteRepo            interfaces.WagerVoteRepository
	groupWagerRepo           interfaces.GroupWagerRepository
	guildSettingsRepo        interfaces.GuildSettingsRepository
	summonerWatchRepo        interfaces.SummonerWatchRepository
	wordleCompletionRepo     interfaces.WordleCompletionRepository
	highRollerPurchaseRepo   interfaces.HighRollerPurchaseRepository
}

// transactionalEventBus wraps the unit of work to buffer events
type transactionalEventBus struct {
	uow *unitOfWork
}

// Publish stores an event in the pending queue without immediately publishing
func (t *transactionalEventBus) Publish(event events.Event) error {
	log.WithFields(log.Fields{
		"eventType":    event.Type(),
		"pendingCount": len(t.uow.pendingEvents),
	}).Debug("Adding event to unit of work pending queue")

	t.uow.pendingEvents = append(t.uow.pendingEvents, event)
	return nil
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
	u.pendingEvents = make([]events.Event, 0)

	// Create guild-scoped repositories with the transaction
	u.userRepo = repository.NewUserRepositoryScoped(tx, u.guildID)
	u.balanceHistoryRepo = repository.NewBalanceHistoryRepositoryScoped(tx, u.guildID)
	u.betRepo = repository.NewBetRepositoryScoped(tx, u.guildID)
	u.wagerRepo = repository.NewWagerRepositoryScoped(tx, u.guildID)
	u.wagerVoteRepo = repository.NewWagerVoteRepositoryScoped(tx, u.guildID)
	u.groupWagerRepo = repository.NewGroupWagerRepositoryScoped(tx, u.guildID)
	u.guildSettingsRepo = repository.NewGuildSettingsRepositoryWithTx(tx) // Guild settings don't need scoping
	u.summonerWatchRepo = repository.NewSummonerWatchRepositoryScoped(tx, u.guildID)
	u.wordleCompletionRepo = repository.NewWordleCompletionRepositoryScoped(tx, u.guildID)
	u.highRollerPurchaseRepo = repository.NewHighRollerPurchaseRepositoryScoped(tx, u.guildID)

	return nil
}

// Commit commits the transaction and flushes events on success
func (u *unitOfWork) Commit() error {
	if u.tx == nil {
		return fmt.Errorf("no transaction to commit")
	}

	// First commit the database transaction
	err := u.tx.Commit(u.ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	u.tx = nil

	// Then flush pending events after successful commit
	if u.eventPublisher != nil && len(u.pendingEvents) > 0 {
		log.WithFields(log.Fields{
			"pendingEventCount": len(u.pendingEvents),
		}).Debug("Flushing pending events from unit of work")

		// Process all pending events
		for _, event := range u.pendingEvents {
			eventType := event.Type()

			log.WithFields(log.Fields{
				"eventType": eventType,
			}).Debug("Publishing event via real publisher")

			if err := u.eventPublisher.Publish(event); err != nil {
				// Log error but continue with other events
				// This ensures partial failure doesn't block all events
				log.WithFields(log.Fields{
					"eventType": eventType,
					"error":     err,
				}).Error("Failed to publish event during flush")
			}
		}

		// Clear the pending queue
		u.pendingEvents = u.pendingEvents[:0]
		log.Debug("All pending events flushed to real publisher")
	}

	return nil
}

// Rollback rolls back the transaction and discards pending events
func (u *unitOfWork) Rollback() error {
	// Discard pending events
	if len(u.pendingEvents) > 0 {
		log.WithFields(log.Fields{
			"discardedEventCount": len(u.pendingEvents),
		}).Debug("Discarding pending events from unit of work")
		u.pendingEvents = u.pendingEvents[:0]
	}

	// Then rollback the database transaction
	if u.tx == nil {
		return nil // Nothing to rollback
	}

	err := u.tx.Rollback(u.ctx)
	if err != nil && err != pgx.ErrTxClosed {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	u.tx = nil
	return nil
}

// Repository getters
func (u *unitOfWork) UserRepository() interfaces.UserRepository {
	if u.userRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.userRepo
}

func (u *unitOfWork) BalanceHistoryRepository() interfaces.BalanceHistoryRepository {
	if u.balanceHistoryRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.balanceHistoryRepo
}

func (u *unitOfWork) BetRepository() interfaces.BetRepository {
	if u.betRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.betRepo
}

func (u *unitOfWork) WagerRepository() interfaces.WagerRepository {
	if u.wagerRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.wagerRepo
}

func (u *unitOfWork) WagerVoteRepository() interfaces.WagerVoteRepository {
	if u.wagerVoteRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.wagerVoteRepo
}

func (u *unitOfWork) GroupWagerRepository() interfaces.GroupWagerRepository {
	if u.groupWagerRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.groupWagerRepo
}

func (u *unitOfWork) GuildSettingsRepository() interfaces.GuildSettingsRepository {
	if u.guildSettingsRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.guildSettingsRepo
}

func (u *unitOfWork) SummonerWatchRepository() interfaces.SummonerWatchRepository {
	if u.summonerWatchRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.summonerWatchRepo
}

func (u *unitOfWork) WordleCompletionRepo() interfaces.WordleCompletionRepository {
	if u.wordleCompletionRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.wordleCompletionRepo
}

func (u *unitOfWork) HighRollerPurchaseRepository() interfaces.HighRollerPurchaseRepository {
	if u.highRollerPurchaseRepo == nil {
		panic("unit of work not started - call Begin() first")
	}
	return u.highRollerPurchaseRepo
}

// EventBus returns the transactional event publisher
func (u *unitOfWork) EventBus() interfaces.EventPublisher {
	return &transactionalEventBus{uow: u}
}
