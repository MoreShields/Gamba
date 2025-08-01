package application

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/service"

	log "github.com/sirupsen/logrus"
)

// DailyAwardsWorkerImpl handles the scheduling and orchestration of daily awards
type DailyAwardsWorkerImpl struct {
	uowFactory        UnitOfWorkFactory
	guildDiscovery    GuildDiscoveryService
	dailyAwardsPoster DiscordPoster
}

// NewDailyAwardsWorker creates a new daily awards worker
func NewDailyAwardsWorker(
	uowFactory UnitOfWorkFactory,
	guildDiscovery GuildDiscoveryService,
	dailyAwardsPoster DiscordPoster,
) *DailyAwardsWorkerImpl {
	return &DailyAwardsWorkerImpl{
		uowFactory:        uowFactory,
		guildDiscovery:    guildDiscovery,
		dailyAwardsPoster: dailyAwardsPoster,
	}
}

// Start begins the daily awards worker
func (w *DailyAwardsWorkerImpl) Start(ctx context.Context, notificationHour int) func() {
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

		if err := w.processAllGuilds(ctx); err != nil {
			log.Errorf("Error processing daily awards: %v", err)
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

// processAllGuilds processes daily awards for all guilds
func (w *DailyAwardsWorkerImpl) processAllGuilds(ctx context.Context) error {
	// Get all guilds with primary channels
	guilds, err := w.guildDiscovery.GetGuildsWithPrimaryChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get guilds: %w", err)
	}

	// Track successes and failures for monitoring
	var successCount, failureCount int

	// Process each guild separately
	for _, guild := range guilds {
		if err := w.processGuildAwards(ctx, guild); err != nil {
			log.Errorf("Error processing daily awards for guild %d: %v", guild.GuildID, err)
			failureCount++
			// Continue with other guilds
		} else {
			successCount++
		}
	}

	log.WithFields(log.Fields{
		"total_guilds":    len(guilds),
		"successful":      successCount,
		"failed":          failureCount,
		"processing_time": time.Now().UTC().Format("15:04:05"),
	}).Info("Completed daily awards processing")

	return nil
}

// processGuildAwards processes daily awards for a specific guild
func (w *DailyAwardsWorkerImpl) processGuildAwards(ctx context.Context, guild dto.GuildChannelInfo) error {
	if guild.PrimaryChannelID == nil {
		log.Debugf("Guild %d has no primary channel configured, skipping", guild.GuildID)
		return nil
	}

	// Create UoW for this guild
	uow := w.uowFactory.CreateForGuild(guild.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Create daily awards service
	dailyAwardsService := service.NewDailyAwardsService(
		uow.WordleCompletionRepo(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.BetRepository(),
		uow.GroupWagerRepository(),
	)

	// Get daily awards summary
	summary, err := dailyAwardsService.GetDailyAwardsSummary(ctx, guild.GuildID)
	if err != nil {
		return fmt.Errorf("failed to get daily awards summary: %w", err)
	}

	// Rollback the read-only transaction
	uow.Rollback()

	// Skip if no awards
	if len(summary.Awards) == 0 {
		log.Debugf("No daily awards for guild %d", guild.GuildID)
		return nil
	}

	// Convert to DTO and post
	summaryDTO := w.convertSummaryToDTO(summary)
	postDTO := dto.DailyAwardsPostDTO{
		GuildID:   guild.GuildID,
		ChannelID: *guild.PrimaryChannelID,
		Summary:   summaryDTO,
	}

	if err := w.dailyAwardsPoster.PostDailyAwards(ctx, postDTO); err != nil {
		return fmt.Errorf("failed to post daily awards: %w", err)
	}

	log.WithFields(log.Fields{
		"guild_id":   guild.GuildID,
		"channel_id": *guild.PrimaryChannelID,
		"awards":     len(summary.Awards),
		"source":     "daily_worker",
	}).Info("Daily awards summary posted")

	return nil
}

// convertSummaryToDTO converts service summary to application DTO
func (w *DailyAwardsWorkerImpl) convertSummaryToDTO(summary *service.DailyAwardsSummary) *dto.DailyAwardsSummaryDTO {
	awards := make([]dto.DailyAwardDTO, len(summary.Awards))
	for i, award := range summary.Awards {
		awards[i] = dto.DailyAwardDTO{
			Type:      string(award.GetType()),
			DiscordID: award.GetDiscordID(),
			Reward:    award.GetReward(),
			Details:   award.GetDetails(),
		}
	}

	return &dto.DailyAwardsSummaryDTO{
		GuildID:         summary.GuildID,
		Date:            summary.Date,
		Awards:          awards,
		TotalPayout:     summary.TotalPayout,
		TotalServerBits: summary.TotalServerBits,
	}
}