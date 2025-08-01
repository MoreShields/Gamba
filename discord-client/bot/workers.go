package bot

import (
	"context"
	"fmt"
	"strconv"
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

// StartDailyAwardsWorker starts a background worker to post daily awards summaries
// Returns a cleanup function to stop the worker gracefully
func (b *Bot) StartDailyAwardsWorker(ctx context.Context, notificationHour int) func() {
	stopChan := make(chan struct{})

	// Calculate time until next notification
	calculateNextRun := func() time.Duration {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), notificationHour, 0, 0, 0, time.UTC)
		
		// If the notification time has already passed today, schedule for tomorrow
		if now.After(next) || now.Equal(next) {
			next = next.Add(24 * time.Hour)
		}
		
		return next.Sub(now)
	}

	// Process daily awards for all guilds
	processDailyAwards := func() {
		log.Info("Processing daily awards for all guilds")

		// First, get all guild IDs that have a primary channel configured
		tempUow := b.uowFactory.CreateForGuild(0)
		if err := tempUow.Begin(context.Background()); err != nil {
			log.Errorf("Error beginning transaction to get guild settings: %v", err)
			return
		}

		// Note: We'll need to add a GetAllGuildsWithPrimaryChannel method to the service
		// For now, we'll process guilds that we know have settings
		tempUow.Rollback()

		// Get all guilds the bot is in
		guilds := b.session.State.Guilds
		for _, guild := range guilds {
			guildID, err := strconv.ParseInt(guild.ID, 10, 64)
			if err != nil {
				log.Errorf("Error parsing guild ID %s: %v", guild.ID, err)
				continue
			}

			// Process each guild separately
			uow := b.uowFactory.CreateForGuild(guildID)
			if err := uow.Begin(context.Background()); err != nil {
				log.Errorf("Error beginning transaction for guild %d daily awards: %v", guildID, err)
				continue
			}

			// Get guild settings to check for primary channel
			guildSettingsService := service.NewGuildSettingsService(uow.GuildSettingsRepository())
			settings, err := guildSettingsService.GetOrCreateSettings(context.Background(), guildID)
			if err != nil {
				log.Errorf("Error getting guild settings for %d: %v", guildID, err)
				uow.Rollback()
				continue
			}

			// Skip if no primary channel configured
			if settings.PrimaryChannelID == nil {
				log.Debugf("Guild %d has no primary channel configured, skipping daily awards", guildID)
				uow.Rollback()
				continue
			}

			// Create daily awards service
			rewardService := service.NewWordleRewardService(uow.WordleCompletionRepo(), 0)
			dailyAwardsService := service.NewDailyAwardsService(
				uow.WordleCompletionRepo(),
				uow.UserRepository(),
				rewardService,
			)

			// Get daily awards summary
			summary, err := dailyAwardsService.GetDailyAwardsSummary(context.Background(), guildID)
			if err != nil {
				log.Errorf("Error getting daily awards summary for guild %d: %v", guildID, err)
				uow.Rollback()
				continue
			}

			// Rollback the read-only transaction (no changes were made)
			uow.Rollback()

			// Skip if no awards to post
			if len(summary.Awards) == 0 {
				log.Debugf("No daily awards for guild %d", guildID)
				continue
			}

			// Post the summary to Discord
			channelID := fmt.Sprintf("%d", *settings.PrimaryChannelID)
			if err := b.PostDailyAwardsSummary(context.Background(), channelID, summary); err != nil {
				log.Errorf("Error posting daily awards for guild %d: %v", guildID, err)
			} else {
				log.Infof("Successfully posted daily awards for guild %d", guildID)
			}
		}
	}

	// Start the worker goroutine
	go func() {
		log.Infof("Daily awards worker started, next run at %02d:00 UTC", notificationHour)

		for {
			// Calculate time until next run
			waitDuration := calculateNextRun()
			log.Infof("Daily awards worker waiting %v until next run", waitDuration)

			select {
			case <-ctx.Done():
				log.Info("Daily awards worker shutting down (context cancelled)...")
				return
			case <-stopChan:
				log.Info("Daily awards worker shutting down (stop requested)...")
				return
			case <-time.After(waitDuration):
				processDailyAwards()
			}
		}
	}()

	// Return cleanup function
	return func() {
		close(stopChan)
	}
}
