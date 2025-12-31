package application

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/services"

	log "github.com/sirupsen/logrus"
)

// LotteryDrawWorker handles scheduled lottery draws
type LotteryDrawWorker struct {
	uowFactory    UnitOfWorkFactory
	lotteryPoster LotteryPoster
}

// LotteryPoster defines the interface for posting lottery results to Discord
type LotteryPoster interface {
	// PostLotteryResult updates the existing lottery message with draw results
	PostLotteryResult(ctx context.Context, draw *entities.LotteryDraw, result *interfaces.LotteryDrawResult, participants []*entities.LotteryParticipantInfo) error

	// PostNewLotteryDraw posts a new lottery draw message
	PostNewLotteryDraw(ctx context.Context, drawInfo *interfaces.LotteryDrawInfo, channelID int64) (messageID int64, err error)

	// UpdateLotteryEmbed updates an existing lottery embed
	UpdateLotteryEmbed(ctx context.Context, draw *entities.LotteryDraw, drawInfo *interfaces.LotteryDrawInfo) error
}

// NewLotteryDrawWorker creates a new lottery draw worker
func NewLotteryDrawWorker(uowFactory UnitOfWorkFactory, lotteryPoster LotteryPoster) *LotteryDrawWorker {
	return &LotteryDrawWorker{
		uowFactory:    uowFactory,
		lotteryPoster: lotteryPoster,
	}
}

// Start begins the lottery draw worker
func (w *LotteryDrawWorker) Start(ctx context.Context) func() {
	stopChan := make(chan struct{})

	// Get next draw time from database
	getNextDrawTime := func() *time.Time {
		uow := w.uowFactory.CreateForGuild(0)
		if err := uow.Begin(ctx); err != nil {
			log.Errorf("Failed to begin transaction for next draw time: %v", err)
			return nil
		}
		defer uow.Rollback()

		nextTime, err := uow.LotteryDrawRepository().GetNextPendingDrawTime(ctx)
		if err != nil {
			log.Errorf("Failed to get next draw time: %v", err)
			return nil
		}
		return nextTime
	}

	// Start the worker goroutine
	go func() {
		log.Info("Lottery draw worker started")

		for {
			// First, process any past-due draws
			if err := w.processAllPendingDraws(ctx); err != nil {
				log.Errorf("Error processing pending draws: %v", err)
			}

			// Get next scheduled draw time
			nextDrawTime := getNextDrawTime()
			if nextDrawTime == nil {
				// No pending draws, check again in 1 hour
				log.Info("No pending lottery draws, checking again in 1 hour")
				select {
				case <-ctx.Done():
					log.Info("Lottery draw worker shutting down (context cancelled)...")
					return
				case <-stopChan:
					log.Info("Lottery draw worker shutting down (stop requested)...")
					return
				case <-time.After(1 * time.Hour):
					continue
				}
			}

			waitDuration := time.Until(*nextDrawTime)
			if waitDuration <= 0 {
				// Draw time already passed, loop to process immediately
				continue
			}

			log.Infof("Next lottery draw at %v (in %v)", nextDrawTime.UTC(), waitDuration)

			select {
			case <-ctx.Done():
				log.Info("Lottery draw worker shutting down (context cancelled)...")
				return
			case <-stopChan:
				log.Info("Lottery draw worker shutting down (stop requested)...")
				return
			case <-time.After(waitDuration):
				// Timer fired, loop to process
			}
		}
	}()

	// Return cleanup function
	return func() {
		close(stopChan)
	}
}

// processAllPendingDraws processes all lottery draws that are ready
func (w *LotteryDrawWorker) processAllPendingDraws(ctx context.Context) error {
	// Create a simple UoW just to query pending draws (not guild-scoped)
	// We need to use a general query to get all pending draws across all guilds
	uow := w.uowFactory.CreateForGuild(0) // 0 guildID for cross-guild query
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Get all pending draws
	pendingDraws, err := uow.LotteryDrawRepository().GetPendingDrawsForTime(ctx, time.Now().UTC())
	if err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to get pending draws: %w", err)
	}
	uow.Rollback() // Close the read transaction

	if len(pendingDraws) == 0 {
		log.Info("No pending lottery draws to process")
		return nil
	}

	log.Infof("Found %d pending lottery draws to process", len(pendingDraws))

	// Track results
	var successCount, failureCount int

	// Process each draw in its own transaction
	for _, draw := range pendingDraws {
		if err := w.processGuildDraw(ctx, draw); err != nil {
			log.Errorf("Error processing lottery draw %d for guild %d: %v", draw.ID, draw.GuildID, err)
			failureCount++
		} else {
			successCount++
		}
	}

	log.WithFields(log.Fields{
		"total_draws": len(pendingDraws),
		"successful":  successCount,
		"failed":      failureCount,
	}).Info("Completed lottery draw processing")

	return nil
}

// processGuildDraw processes a single lottery draw
func (w *LotteryDrawWorker) processGuildDraw(ctx context.Context, draw *entities.LotteryDraw) error {
	// Create UoW for this guild
	uow := w.uowFactory.CreateForGuild(draw.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Check if lottery channel is configured
	guildSettings, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, draw.GuildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	if !guildSettings.HasLottoChannel() {
		log.Warnf("Guild %d has no lottery channel configured, skipping draw %d", draw.GuildID, draw.ID)
		return nil
	}

	channelID := guildSettings.GetLottoChannelID()

	// Create lottery service
	lotteryService := services.NewLotteryService(
		uow.LotteryDrawRepository(),
		uow.LotteryTicketRepository(),
		uow.LotteryWinnerRepository(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
		uow.GuildSettingsRepository(),
		uow.EventBus(),
	)

	// Conduct the draw
	result, err := lotteryService.ConductDraw(ctx, draw)
	if err != nil {
		return fmt.Errorf("failed to conduct draw: %w", err)
	}

	// Get participants for the result embed (before commit)
	participants, err := uow.LotteryTicketRepository().GetParticipantSummary(ctx, draw.ID)
	if err != nil {
		log.Warnf("Failed to get participants for draw %d: %v", draw.ID, err)
		participants = nil
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Post result to Discord (outside transaction)
	if err := w.lotteryPoster.PostLotteryResult(ctx, draw, result, participants); err != nil {
		log.Errorf("Failed to post lottery result to Discord: %v", err)
		// Don't return error - the draw was processed successfully
	}

	// Post new draw message
	if result.NextDraw != nil {
		if err := w.postNewDrawMessage(ctx, result.NextDraw, channelID); err != nil {
			log.Errorf("Failed to post new draw message: %v", err)
		}
	}

	log.WithFields(log.Fields{
		"draw_id":        draw.ID,
		"guild_id":       draw.GuildID,
		"winning_number": result.WinningNumber,
		"pot_amount":     result.PotAmount,
		"winner_count":   len(result.Winners),
		"rolled_over":    result.RolledOver,
	}).Info("Lottery draw completed")

	return nil
}

// postNewDrawMessage posts a new lottery draw message for rollover
func (w *LotteryDrawWorker) postNewDrawMessage(ctx context.Context, draw *entities.LotteryDraw, channelID int64) error {
	// Create UoW for this guild
	uow := w.uowFactory.CreateForGuild(draw.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get draw info
	lotteryService := services.NewLotteryService(
		uow.LotteryDrawRepository(),
		uow.LotteryTicketRepository(),
		uow.LotteryWinnerRepository(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
		uow.GuildSettingsRepository(),
		uow.EventBus(),
	)

	drawInfo, err := lotteryService.GetDrawInfo(ctx, draw.GuildID)
	if err != nil {
		return fmt.Errorf("failed to get draw info: %w", err)
	}

	// Post new draw message
	messageID, err := w.lotteryPoster.PostNewLotteryDraw(ctx, drawInfo, channelID)
	if err != nil {
		return fmt.Errorf("failed to post new draw message: %w", err)
	}

	// Save message ID to draw - use drawInfo.Draw.ID to ensure consistency
	if err := lotteryService.SetDrawMessage(ctx, drawInfo.Draw.ID, channelID, messageID); err != nil {
		return fmt.Errorf("failed to save draw message ID: %w", err)
	}

	// Commit
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
