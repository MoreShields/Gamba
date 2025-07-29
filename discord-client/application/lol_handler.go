package application

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/models"
	"gambler/discord-client/service"

	log "github.com/sirupsen/logrus"
)

// LoLHandlerImpl implements the LoLEventHandler interface
type LoLHandlerImpl struct {
	uowFactory    UnitOfWorkFactory
	discordPoster DiscordPoster
}

// NewLoLHandler creates a new LoL event handler
func NewLoLHandler(
	uowFactory UnitOfWorkFactory,
	discordPoster DiscordPoster,
) *LoLHandlerImpl {
	return &LoLHandlerImpl{
		uowFactory:    uowFactory,
		discordPoster: discordPoster,
	}
}

// HandleGameStarted creates house wagers when a game starts
func (h *LoLHandlerImpl) HandleGameStarted(ctx context.Context, gameStarted dto.GameStartedDTO) error {
	log.WithFields(log.Fields{
		"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
		"gameId":   gameStarted.GameID,
		"queue":    gameStarted.QueueType,
	}).Info("handling Game start")

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

	// Create a house wager for each watching guild
	// Currently only creating wagers for ranked games.
	if gameStarted.QueueType == "RANKED_SOLO_5x5" {
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

	// Query guilds watching this summoner to find relevant wagers
	tempUow := h.uowFactory.CreateForGuild(0)
	if err := tempUow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tempUow.Rollback()

	guilds, err := tempUow.SummonerWatchRepository().GetGuildsWatchingSummoner(ctx, gameEnded.SummonerName, gameEnded.TagLine)
	if err != nil {
		return fmt.Errorf("failed to get guilds watching summoner: %w", err)
	}

	if len(guilds) == 0 {
		log.WithFields(log.Fields{
			"summoner": fmt.Sprintf("%s#%s", gameEnded.SummonerName, gameEnded.TagLine),
		}).Debug("No guilds watching this summoner")
		return nil
	}

	// Create external reference for this game
	externalRef := models.ExternalReference{
		System: models.SystemLeagueOfLegends,
		ID:     gameEnded.GameID,
	}

	// Look up and resolve wagers for each guild
	resolvedCount := 0
	for _, guild := range guilds {
		// Create a scoped UoW for this guild to query wagers
		guildUow := h.uowFactory.CreateForGuild(guild.GuildID)
		if err := guildUow.Begin(ctx); err != nil {
			log.WithFields(log.Fields{
				"guild": guild.GuildID,
				"error": err,
			}).Error("Failed to begin transaction for guild")
			continue
		}

		// Find the wager for this game in this guild
		log.WithFields(log.Fields{
			"guild":          guild.GuildID,
			"gameId":         gameEnded.GameID,
			"externalSystem": externalRef.System,
		}).Debug("Looking up wager by external reference")

		wager, err := guildUow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
		if err != nil {
			log.WithFields(log.Fields{
				"guild":  guild.GuildID,
				"gameId": gameEnded.GameID,
				"error":  err,
			}).Error("Failed to query wager by external reference")
			guildUow.Rollback()
			continue
		}

		if wager == nil {
			log.WithFields(log.Fields{
				"guild":  guild.GuildID,
				"gameId": gameEnded.GameID,
			}).Debug("No wager found for this game in guild")
			guildUow.Rollback()
			continue
		}

		log.WithFields(log.Fields{
			"guild":   guild.GuildID,
			"gameId":  gameEnded.GameID,
			"wagerID": wager.ID,
		}).Debug("Found wager for external reference")

		guildUow.Rollback() // Close the query transaction

		// Resolve the wager
		if err := h.resolveHouseWager(ctx, guild.GuildID, wager.ID, gameEnded.Won, gameEnded.DurationSeconds); err != nil {
			log.WithFields(log.Fields{
				"guild":   guild.GuildID,
				"wagerID": wager.ID,
				"error":   err,
			}).Error("Failed to resolve house wager")
			// Continue with other guilds
		} else {
			resolvedCount++
		}
	}

	log.WithFields(log.Fields{
		"gameId":        gameEnded.GameID,
		"resolvedCount": resolvedCount,
		"totalGuilds":   len(guilds),
	}).Info("Completed resolving house wagers for game")

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

	// Format the complete description with summoner info and League of Graphs link
	leagueOfGraphsURL := fmt.Sprintf("https://www.leagueofgraphs.com/match/NA/%s", gameStarted.GameID)
	condition := fmt.Sprintf("%s - **Ranked Match**\n[View Match Details](%s)",
		gameStarted.SummonerName, leagueOfGraphsURL)

	// Create the house wager with formatted condition
	options := []string{"Win", "Loss"}
	oddsMultipliers := []float64{2.0, 2.0} // 2:1 odds for now
	votingPeriodMinutes := 5               // 5 minutes for betting

	// Use nil for system-created house wagers (no specific creator)
	wagerDetail, err := groupWagerService.CreateGroupWager(
		ctx,
		nil,
		condition, // Store the complete formatted description as the condition
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

	// Set the external reference for this game
	wagerDetail.Wager.SetExternalReference(models.SystemLeagueOfLegends, gameStarted.GameID)

	log.WithFields(log.Fields{
		"guild":          guild.GuildID,
		"wagerID":        wagerDetail.Wager.ID,
		"gameID":         gameStarted.GameID,
		"externalSystem": models.SystemLeagueOfLegends,
	}).Debug("Setting external reference for house wager")

	// Update the wager with the external reference
	if err := uow.GroupWagerRepository().Update(ctx, wagerDetail.Wager); err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to update wager with external reference: %w", err)
	}

	// Build DTO for Discord posting
	channelID := int64(0)
	if guildSettings.LolChannelID != nil {
		channelID = *guildSettings.LolChannelID
	} else {
		return fmt.Errorf("failed to create group wager: lol-channel is not set for guild %d", guild.GuildID)
	}

	// Parse the condition to extract title and description
	// Split on first newline - everything before is title, everything after is description
	parts := strings.SplitN(condition, "\n", 2)
	title := parts[0]
	description := ""
	if len(parts) > 1 {
		description = parts[1]
	}

	postDTO := dto.HouseWagerPostDTO{
		GuildID:      guild.GuildID,
		ChannelID:    channelID,
		WagerID:      wagerDetail.Wager.ID,
		Title:        title,       // Title from first line
		Description:  description, // Description from remaining lines
		State:        string(wagerDetail.Wager.State),
		Options:      make([]dto.WagerOptionDTO, len(wagerDetail.Options)),
		VotingEndsAt: wagerDetail.Wager.VotingEndsAt,
	}

	// Convert options to DTOs
	for i, opt := range wagerDetail.Options {
		postDTO.Options[i] = dto.WagerOptionDTO{
			ID:          opt.ID,
			Text:        opt.OptionText,
			Order:       opt.OptionOrder,
			Multiplier:  opt.OddsMultiplier,
			TotalAmount: opt.TotalAmount,
		}
	}

	// Convert participants to DTOs
	postDTO.Participants = make([]dto.ParticipantDTO, len(wagerDetail.Participants))
	for i, participant := range wagerDetail.Participants {
		postDTO.Participants[i] = dto.ParticipantDTO{
			DiscordID: participant.DiscordID,
			OptionID:  participant.OptionID,
			Amount:    participant.Amount,
		}
	}

	// Set total pot
	postDTO.TotalPot = wagerDetail.Wager.TotalPot

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
func (h *LoLHandlerImpl) resolveHouseWager(ctx context.Context, guildID, wagerID int64, won bool, durationSeconds int32) error {
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

	// Create group wager service once
	groupWagerService := service.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Check for edge case: forfeit/remake
	// Games that are very short (< 10 minutes) and lost are likely forfeit/remake scenarios
	const forfeitThresholdSeconds = 600 // 10 minutes
	if !won && durationSeconds < forfeitThresholdSeconds {
		log.WithFields(log.Fields{
			"guild":           guildID,
			"wagerID":         wagerID,
			"durationSeconds": durationSeconds,
		}).Info("Game ended without win and short duration (forfeit/remake), cancelling wager and refunding participants")

		// Cancel the wager (nil indicates system cancellation)
		if err := groupWagerService.CancelGroupWager(ctx, wagerID, nil); err != nil {
			log.WithFields(log.Fields{
				"guild":   guildID,
				"wagerID": wagerID,
				"error":   err,
			}).Error("Failed to cancel group wager")
			uow.Rollback()
			return fmt.Errorf("failed to cancel group wager: %w", err)
		}

		// Commit the transaction
		if err := uow.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.WithFields(log.Fields{
			"guild":   guildID,
			"wagerID": wagerID,
		}).Info("Successfully cancelled house wager and refunded participants")

		return nil
	}

	// Determine winning option based on game result
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

	log.WithFields(log.Fields{
		"guild":           guildID,
		"wagerID":         wagerID,
		"wagerState":      wagerDetail.Wager.State,
		"participants":    len(wagerDetail.Participants),
		"winningOptionID": winningOptionID,
		"won":             won,
	}).Debug("Attempting to resolve house wager")

	// For house wagers, use nil to indicate system resolution (no human resolver)
	// Resolve the wager
	result, err := groupWagerService.ResolveGroupWager(ctx, wagerID, nil, winningOptionID)
	if err != nil {
		log.WithFields(log.Fields{
			"guild":   guildID,
			"wagerID": wagerID,
			"error":   err,
		}).Error("Failed to resolve group wager in service")
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
