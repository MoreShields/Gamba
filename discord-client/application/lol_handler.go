package application

import (
	"context"
	"fmt"
	"sync"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/models"
	"gambler/discord-client/service"

	log "github.com/sirupsen/logrus"
)

// LoLHandlerImpl implements the LoLHandler interface
type LoLHandlerImpl struct {
	uowFactory    service.UnitOfWorkFactory
	discordPoster DiscordPoster

	// Track active game wagers: gameID -> (guildID -> wagerID)
	activeGameWagers map[string]map[int64]int64
	activeGameMu     sync.RWMutex
}

// NewLoLHandler creates a new LoL event handler
func NewLoLHandler(
	uowFactory service.UnitOfWorkFactory,
	discordPoster DiscordPoster,
) LoLHandler {
	return &LoLHandlerImpl{
		uowFactory:       uowFactory,
		discordPoster:    discordPoster,
		activeGameWagers: make(map[string]map[int64]int64),
	}
}

// HandleGameStarted creates house wagers when a game starts
func (h *LoLHandlerImpl) HandleGameStarted(ctx context.Context, gameStarted dto.GameStartedDTO) error {
	log.WithFields(log.Fields{
		"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
		"gameId":   gameStarted.GameID,
		"queue":    gameStarted.QueueType,
	}).Info("Game started, creating house wagers")

	// Query guilds watching this summoner
	// Use a temporary UoW to query without guild scope
	tempUow := h.uowFactory.CreateForGuild(0)
	if err := tempUow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tempUow.Rollback()

	guilds, err := tempUow.SummonerWatchRepository().GetGuildsWatchingSummoner(ctx, gameStarted.SummonerName, gameStarted.TagLine)
	if err != nil {
		return fmt.Errorf("failed to get guilds watching summoner: %w", err)
	}

	if len(guilds) == 0 {
		log.WithFields(log.Fields{
			"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
		}).Debug("No guilds watching this summoner")
		return nil
	}

	// Track wagers for this game
	h.activeGameMu.Lock()
	if _, exists := h.activeGameWagers[gameStarted.GameID]; !exists {
		h.activeGameWagers[gameStarted.GameID] = make(map[int64]int64)
	}
	h.activeGameMu.Unlock()

	// Create a house wager for each watching guild
	for _, guild := range guilds {
		if err := h.createHouseWagerForGuild(ctx, guild, gameStarted); err != nil {
			log.WithFields(log.Fields{
				"guild":    guild.GuildID,
				"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
				"error":    err,
			}).Error("Failed to create house wager for guild")
			// Continue with other guilds
		}
	}

	return nil
}

// HandleGameEnded resolves house wagers when a game ends
func (h *LoLHandlerImpl) HandleGameEnded(ctx context.Context, gameEnded dto.GameEndedDTO) error {
	log.WithFields(log.Fields{
		"summoner": fmt.Sprintf("%s#%s", gameEnded.SummonerName, gameEnded.TagLine),
		"gameId":   gameEnded.GameID,
		"won":      gameEnded.Won,
		"duration": gameEnded.DurationSeconds,
	}).Info("Game ended, resolving house wagers")

	// Look up active wagers for this game
	h.activeGameMu.RLock()
	guildWagers, exists := h.activeGameWagers[gameEnded.GameID]
	h.activeGameMu.RUnlock()

	if !exists || len(guildWagers) == 0 {
		log.WithFields(log.Fields{
			"gameId": gameEnded.GameID,
		}).Debug("No active wagers found for game")
		return nil
	}

	// Resolve wager for each guild
	for guildID, wagerID := range guildWagers {
		if err := h.resolveHouseWager(ctx, guildID, wagerID, gameEnded.Won); err != nil {
			log.WithFields(log.Fields{
				"guild":   guildID,
				"wagerID": wagerID,
				"error":   err,
			}).Error("Failed to resolve house wager")
			// Continue with other guilds
		}
	}

	// Clean up tracking
	h.activeGameMu.Lock()
	delete(h.activeGameWagers, gameEnded.GameID)
	h.activeGameMu.Unlock()

	return nil
}

// createHouseWagerForGuild creates a house wager for a specific guild
func (h *LoLHandlerImpl) createHouseWagerForGuild(
	ctx context.Context,
	guild *models.GuildSummonerWatch,
	gameStarted dto.GameStartedDTO,
) error {
	// Create UoW for this guild
	uow := h.uowFactory.CreateForGuild(guild.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := recover(); err != nil {
			uow.Rollback()
			panic(err)
		}
	}()

	// Get guild settings for channel info
	guildSettings, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guild.GuildID)
	if err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Create group wager service
	groupWagerService := service.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Create the house wager
	condition := fmt.Sprintf("%s#%s - %s", gameStarted.SummonerName, gameStarted.TagLine, h.getQueueTypeDisplay(gameStarted.QueueType))
	options := []string{"Win", "Loss"}
	oddsMultipliers := []float64{1.0, 1.0} // Even odds for now
	votingPeriodMinutes := 5               // 5 minutes for betting

	// Use nil for system-created house wagers (no specific creator)
	wagerDetail, err := groupWagerService.CreateGroupWager(
		ctx,
		nil,
		condition,
		options,
		votingPeriodMinutes,
		0, // Message ID will be set after posting
		0, // Channel ID will be set after posting
		models.GroupWagerTypeHouse,
		oddsMultipliers,
	)
	if err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to create group wager: %w", err)
	}

	// Track the wager
	h.activeGameMu.Lock()
	h.activeGameWagers[gameStarted.GameID][guild.GuildID] = wagerDetail.Wager.ID
	h.activeGameMu.Unlock()

	// Build DTO for Discord posting
	channelID := int64(0)
	if guildSettings.LolChannelID != nil {
		channelID = *guildSettings.LolChannelID
	} else {
		return fmt.Errorf("failed to create group wager: lol-channel is not set for guild %d", guild.GuildID)
	}

	postDTO := dto.HouseWagerPostDTO{
		GuildID:      guild.GuildID,
		ChannelID:    channelID,
		WagerID:      wagerDetail.Wager.ID,
		Title:        "🎮 New Game Started!",
		Description:  fmt.Sprintf("Place your bets on %s's game outcome!", gameStarted.SummonerName),
		Options:      make([]dto.WagerOptionDTO, len(wagerDetail.Options)),
		VotingEndsAt: wagerDetail.Wager.VotingEndsAt,
		SummonerInfo: dto.SummonerInfoDTO{
			GameName:  gameStarted.SummonerName,
			TagLine:   gameStarted.TagLine,
			QueueType: h.getQueueTypeDisplay(gameStarted.QueueType),
			GameID:    gameStarted.GameID,
		},
	}

	// Convert options to DTOs
	for i, opt := range wagerDetail.Options {
		postDTO.Options[i] = dto.WagerOptionDTO{
			ID:         opt.ID,
			Text:       opt.OptionText,
			Order:      opt.OptionOrder,
			Multiplier: opt.OddsMultiplier,
		}
	}

	// Post to Discord
	postResult, err := h.discordPoster.PostHouseWager(ctx, postDTO)
	if err != nil {
		log.WithFields(log.Fields{
			"guild":   guild.GuildID,
			"wagerID": wagerDetail.Wager.ID,
			"error":   err,
		}).Error("Failed to post house wager to Discord")
		// Don't return error - wager is created, just not posted
	} else {
		// Update the wager with messageID and channelID
		wagerDetail.Wager.MessageID = postResult.MessageID
		wagerDetail.Wager.ChannelID = postResult.ChannelID
		if err := uow.GroupWagerRepository().Update(ctx, wagerDetail.Wager); err != nil {
			log.WithFields(log.Fields{
				"guild":     guild.GuildID,
				"wagerID":   wagerDetail.Wager.ID,
				"messageID": postResult.MessageID,
				"channelID": postResult.ChannelID,
				"error":     err,
			}).Error("Failed to update wager with message info")
		}
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"guild":    guild.GuildID,
		"wagerID":  wagerDetail.Wager.ID,
		"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
	}).Info("Created house wager for game start")

	return nil
}

// resolveHouseWager resolves a specific house wager
func (h *LoLHandlerImpl) resolveHouseWager(ctx context.Context, guildID, wagerID int64, won bool) error {
	// Create UoW for this guild
	uow := h.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := recover(); err != nil {
			uow.Rollback()
			panic(err)
		}
	}()

	// Get the wager details
	wagerDetail, err := uow.GroupWagerRepository().GetDetailByID(ctx, wagerID)
	if err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to get wager detail: %w", err)
	}

	if wagerDetail == nil || wagerDetail.Wager == nil {
		uow.Rollback()
		return fmt.Errorf("wager not found")
	}

	// Determine winning option
	var winningOptionID int64
	for _, opt := range wagerDetail.Options {
		if (won && opt.OptionText == "Win") || (!won && opt.OptionText == "Loss") {
			winningOptionID = opt.ID
			break
		}
	}

	if winningOptionID == 0 {
		uow.Rollback()
		return fmt.Errorf("could not determine winning option")
	}

	// Create group wager service
	groupWagerService := service.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// For house wagers, use nil to indicate system resolution (no human resolver)
	// Note: This requires the service to be updated to handle system resolution
	// For now, we'll use a special system resolver approach
	systemUserID := int64(-1) // Use -1 to indicate system resolution, different from 0

	// Resolve the wager
	result, err := groupWagerService.ResolveGroupWager(ctx, wagerID, systemUserID, winningOptionID)
	if err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to resolve group wager: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"guild":         guildID,
		"wagerID":       wagerID,
		"winningOption": winningOptionID,
		"totalPot":      result.TotalPot,
		"winnersCount":  len(result.Winners),
	}).Info("Resolved house wager")

	return nil
}

// getQueueTypeDisplay returns a user-friendly display name for queue types
func (h *LoLHandlerImpl) getQueueTypeDisplay(queueType string) string {
	// Map common queue types to display names
	queueMap := map[string]string{
		"RANKED_SOLO_5x5": "Ranked Solo/Duo",
		"RANKED_FLEX_SR":  "Ranked Flex",
		"NORMAL":          "Normal",
		"ARAM":            "ARAM",
		"":                "Game",
	}

	if display, exists := queueMap[queueType]; exists {
		return display
	}
	return queueType
}
