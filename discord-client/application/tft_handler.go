package application

import (
	"context"
	"fmt"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"

	log "github.com/sirupsen/logrus"
)

// TFTHandlerImpl implements the TFTEventHandler interface
type TFTHandlerImpl struct {
	baseHandler *BaseHouseWagerHandler
}

// NewTFTHandler creates a new TFT event handler
func NewTFTHandler(
	uowFactory UnitOfWorkFactory,
	discordPoster DiscordPoster,
) *TFTHandlerImpl {
	return &TFTHandlerImpl{
		baseHandler: NewBaseHouseWagerHandler(uowFactory, discordPoster),
	}
}

// formatTFTQueueType converts TFT queue type strings to user-friendly display names.
// Returns an empty string for unknown queue types.
// Only ranked queue types are supported to restrict wagers to competitive games.
func formatTFTQueueType(queueType string) string {
	switch queueType {
	case "TFT_RANKED":
		return "Ranked TFT"
	case "TFT_RANKED_DOUBLE_UP":
		return "Ranked Double Up"
	default:
		return "" // Unknown or non-ranked queue type
	}
}

// isDoubleUpQueue checks if the queue type is a Double Up mode
func isDoubleUpQueue(queueType string) bool {
	switch queueType {
	case "TFT_DOUBLE_UP", "TFT_NORMAL_DOUBLE_UP", "TFT_RANKED_DOUBLE_UP":
		return true
	default:
		return false
	}
}

// HandleGameStarted creates house wagers when a TFT game starts
func (h *TFTHandlerImpl) HandleGameStarted(ctx context.Context, gameStarted dto.TFTGameStartedDTO) error {
	log.WithFields(log.Fields{
		"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
		"gameId":   gameStarted.GameID,
		"queue":    gameStarted.QueueType,
	}).Info("handling TFT game start")

	// Validate queue type - drop event if unknown
	formattedQueue := formatTFTQueueType(gameStarted.QueueType)
	if formattedQueue == "" {
		log.WithFields(log.Fields{
			"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
			"gameId":   gameStarted.GameID,
			"queue":    gameStarted.QueueType,
		}).Info("Dropping TFT game start event for unknown queue type")
		return nil
	}

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
	for _, guild := range guilds {
		// Format the condition with the queue type
		condition := fmt.Sprintf("%s - **%s**", gameStarted.SummonerName, formattedQueue)

		// Check if this is a Double Up mode (4 teams instead of 8 players)
		var options []string
		var oddsMultipliers []float64
		if isDoubleUpQueue(gameStarted.QueueType) {
			// Double Up has 4 teams, so placements are 1-4
			options = []string{"1", "2", "3", "4"}
			oddsMultipliers = []float64{4.0, 4.0, 4.0, 4.0}
		} else {
			// Regular TFT has 8 players with placement ranges
			options = []string{"1-2", "3-4", "5-6", "7-8"}
			oddsMultipliers = []float64{4.0, 4.0, 4.0, 4.0}
		}

		config := WagerCreationConfig{
			ExternalSystem:      entities.SystemTFT,
			GameID:              gameStarted.GameID,
			SummonerName:        gameStarted.SummonerName,
			TagLine:             gameStarted.TagLine,
			Condition:           condition,
			Options:             options,
			OddsMultipliers:     oddsMultipliers,
			VotingPeriodMinutes: 5, // 5 minutes for betting
			ChannelIDGetter: func(gs *entities.GuildSettings) *int64 {
				return gs.TftChannelID
			},
			ChannelName: "tft-channel",
		}

		if err := h.baseHandler.CreateHouseWagerForGuild(ctx, guild, config); err != nil {
			log.WithFields(log.Fields{
				"guild":    guild.GuildID,
				"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
				"error":    err,
			}).Error("Failed to create TFT house wager for guild")
			// Continue with other guilds
		}
	}

	return nil
}

// HandleGameEnded resolves house wagers when a TFT game ends
func (h *TFTHandlerImpl) HandleGameEnded(ctx context.Context, gameEnded dto.TFTGameEndedDTO) error {
	log.WithFields(log.Fields{
		"summoner":  fmt.Sprintf("%s#%s", gameEnded.SummonerName, gameEnded.TagLine),
		"gameId":    gameEnded.GameID,
		"placement": gameEnded.Placement,
		"duration":  gameEnded.DurationSeconds,
	}).Info("TFT game ended, resolving house wagers")

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
		System: entities.SystemTFT,
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
		}).Debug("Looking up TFT wager by external reference")

		wager, err := guildUow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
		if err != nil {
			log.WithFields(log.Fields{
				"guild":  guild.GuildID,
				"gameId": gameEnded.GameID,
				"error":  err,
			}).Error("Failed to query TFT wager by external reference")
			guildUow.Rollback()
			continue
		}

		if wager == nil {
			log.WithFields(log.Fields{
				"guild":  guild.GuildID,
				"gameId": gameEnded.GameID,
			}).Debug("No TFT wager found for this game in guild")
			guildUow.Rollback()
			continue
		}

		log.WithFields(log.Fields{
			"guild":   guild.GuildID,
			"gameId":  gameEnded.GameID,
			"wagerID": wager.ID,
		}).Debug("Found TFT wager for external reference")

		guildUow.Rollback() // Close the query transaction

		// TFT winner selector: Match placement to the correct option
		tftWinnerSelector := func(options []entities.GroupWagerOption, result interface{}) int64 {
			gameResult := result.(dto.TFTGameEndedDTO)
			placement := gameResult.Placement
			
			// Find the winning option by checking if the placement matches
			for _, opt := range options {
				// Check for exact match (Double Up: "1", "2", "3", "4")
				if opt.OptionText == fmt.Sprintf("%d", placement) {
					return opt.ID
				}
				
				// Check for range match (Regular TFT: "1-2", "3-4", "5-6", "7-8")
				switch opt.OptionText {
				case "1-2":
					if placement == 1 || placement == 2 {
						return opt.ID
					}
				case "3-4":
					if placement == 3 || placement == 4 {
						return opt.ID
					}
				case "5-6":
					if placement == 5 || placement == 6 {
						return opt.ID
					}
				case "7-8":
					if placement == 7 || placement == 8 {
						return opt.ID
					}
				}
			}
			
			// No matching option found
			return 0
		}

		// TFT has no 10-minute cancellation logic (unlike LoL)
		config := WagerResolutionConfig{
			ExternalSystem:        entities.SystemTFT,
			WinnerSelector:        tftWinnerSelector,
			GameResult:            gameEnded,
			CancellationThreshold: nil, // No cancellation logic for TFT
		}

		// Resolve the wager
		if err := h.baseHandler.ResolveHouseWager(ctx, guild.GuildID, wager.ID, config); err != nil {
			log.WithFields(log.Fields{
				"guild":   guild.GuildID,
				"wagerID": wager.ID,
				"error":   err,
			}).Error("Failed to resolve TFT house wager")
			// Continue with other guilds
		} else {
			resolvedCount++
		}
	}

	log.WithFields(log.Fields{
		"gameId":        gameEnded.GameID,
		"resolvedCount": resolvedCount,
		"totalGuilds":   len(guilds),
	}).Info("Completed resolving TFT house wagers for game")

	return nil
}