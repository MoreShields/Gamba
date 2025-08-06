package application

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"

	log "github.com/sirupsen/logrus"
)

// LoLHandlerImpl implements the LoLEventHandler interface
type LoLHandlerImpl struct {
	baseHandler *BaseHouseWagerHandler
}

// NewLoLHandler creates a new LoL event handler
func NewLoLHandler(
	uowFactory UnitOfWorkFactory,
	discordPoster DiscordPoster,
) *LoLHandlerImpl {
	return &LoLHandlerImpl{
		baseHandler: NewBaseHouseWagerHandler(uowFactory, discordPoster),
	}
}

// formatQueueType converts queue type strings to user-friendly display names
func formatQueueType(queueType string) string {
	switch queueType {
	case "RANKED_SOLO_5x5":
		return "Ranked Solo/Duo"
	case "RANKED_FLEX_SR":
		return "Ranked Flex"
	case "NORMAL_DRAFT", "NORMAL_BLIND":
		return "Normal SR"
	case "ARAM":
		return "ARAM"
	case "CLASH":
		return "Clash"
	case "ARENA":
		return "Arena"
	default:
		return "Match"
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
	tempUow := h.baseHandler.uowFactory.CreateForGuild(0)
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
	for _, guild := range guilds {
		// Format the complete description with summoner info and Porofessor link for active wager
		// URL-encode the game name and tag with %20 for spaces
		encodedGameName := strings.ReplaceAll(gameStarted.SummonerName, " ", "%20")
		porofessorURL := fmt.Sprintf("https://porofessor.gg/live/na/%s-%s", encodedGameName, gameStarted.TagLine)
		condition := fmt.Sprintf("%s - **%s**\n[Match Details](%s)",
			gameStarted.SummonerName, formatQueueType(gameStarted.QueueType), porofessorURL)

		config := WagerCreationConfig{
			ExternalSystem:      entities.SystemLeagueOfLegends,
			GameID:              gameStarted.GameID,
			SummonerName:        gameStarted.SummonerName,
			TagLine:             gameStarted.TagLine,
			Condition:           condition,
			Options:             []string{"Win", "Loss"},
			OddsMultipliers:     []float64{2.0, 2.0}, // 2:1 odds for now
			VotingPeriodMinutes: 5,                   // 5 minutes for betting
			ChannelIDGetter: func(gs *entities.GuildSettings) *int64 {
				return gs.LolChannelID
			},
			ChannelName: "lol-channel",
		}

		if err := h.baseHandler.CreateHouseWagerForGuild(ctx, guild, config); err != nil {
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

	// Query guilds watching this summoner to find relevant wagers
	tempUow := h.baseHandler.uowFactory.CreateForGuild(0)
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
	externalRef := entities.ExternalReference{
		System: entities.SystemLeagueOfLegends,
		ID:     gameEnded.GameID,
	}

	// Look up and resolve wagers for each guild
	resolvedCount := 0
	for _, guild := range guilds {
		// Create a scoped UoW for this guild to query wagers
		guildUow := h.baseHandler.uowFactory.CreateForGuild(guild.GuildID)
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

		// LoL winner selector
		lolWinnerSelector := func(options []entities.GroupWagerOption, result interface{}) int64 {
			gameResult := result.(dto.GameEndedDTO)
			for _, opt := range options {
				if (gameResult.Won && opt.OptionText == "Win") || (!gameResult.Won && opt.OptionText == "Loss") {
					return opt.ID
				}
			}
			return 0
		}

		forfeitThreshold := int32(600) // 10 minutes
		config := WagerResolutionConfig{
			ExternalSystem:        entities.SystemLeagueOfLegends,
			WinnerSelector:        lolWinnerSelector,
			GameResult:            gameEnded,
			CancellationThreshold: &forfeitThreshold,
		}

		// Resolve the wager
		if err := h.baseHandler.ResolveHouseWager(ctx, guild.GuildID, wager.ID, config); err != nil {
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
