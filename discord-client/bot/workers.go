package bot

import (
	"context"
	"time"

	"gambler/discord-client/service"

	log "github.com/sirupsen/logrus"
)

// StartGroupWagerExpirationWorker starts a background worker to check for expired group wagers
// Returns a cleanup function to stop the worker gracefully
func (b *Bot) StartGroupWagerExpirationWorker(ctx context.Context) func() {
	ticker := time.NewTicker(1 * time.Minute)
	stopChan := make(chan struct{})

	// Run immediately on startup
	processExpiredWagers := func() {
		// First, get all guild IDs that have group wagers
		// Use a temporary UnitOfWork to query all guilds
		tempUow := b.uowFactory.CreateForGuild(0)
		if err := tempUow.Begin(context.Background()); err != nil {
			log.Errorf("Error beginning transaction to get guild list: %v", err)
			return
		}

		// Get all guild IDs with active group wagers
		guildIDs, err := tempUow.GroupWagerRepository().GetGuildsWithActiveWagers(context.Background())
		tempUow.Rollback()

		if err != nil {
			log.Errorf("Error getting guilds with active wagers: %v", err)
			return
		}

		// Process expired wagers for each guild separately
		for _, guildID := range guildIDs {
			uow := b.uowFactory.CreateForGuild(guildID)
			if err := uow.Begin(context.Background()); err != nil {
				log.Errorf("Error beginning transaction for guild %d expired group wagers: %v", guildID, err)
				continue
			}

			// Instantiate service with repositories from UnitOfWork
			groupWagerService := service.NewGroupWagerService(
				uow.GroupWagerRepository(),
				uow.UserRepository(),
				uow.BalanceHistoryRepository(),
				uow.EventBus(),
			)

			if err := groupWagerService.TransitionExpiredWagers(context.Background()); err != nil {
				log.Errorf("Error transitioning expired group wagers for guild %d: %v", guildID, err)
				uow.Rollback()
				continue
			}

			if err := uow.Commit(); err != nil {
				log.Errorf("Error committing expired group wagers transaction for guild %d: %v", guildID, err)
			}
		}
	}

	// Start the worker goroutine
	go func() {
		log.Info("Group wager expiration worker started")

		// Run immediately on startup
		processExpiredWagers()

		for {
			select {
			case <-ctx.Done():
				log.Info("Group wager expiration worker shutting down (context cancelled)...")
				return
			case <-stopChan:
				log.Info("Group wager expiration worker shutting down (stop requested)...")
				return
			case <-ticker.C:
				processExpiredWagers()
			}
		}
	}()

	// Return cleanup function
	return func() {
		ticker.Stop()
		close(stopChan)
	}
}
