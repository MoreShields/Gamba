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

// HandleGameStarted creates house wagers when a TFT game starts
func (h *TFTHandlerImpl) HandleGameStarted(ctx context.Context, gameStarted dto.TFTGameStartedDTO) error {
	log.WithFields(log.Fields{
		"summoner": fmt.Sprintf("%s#%s", gameStarted.SummonerName, gameStarted.TagLine),
		"gameId":   gameStarted.GameID,
		"queue":    gameStarted.QueueType,
	}).Info("handling TFT game start")

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
		// Format the condition without external match URL initially
		condition := fmt.Sprintf("%s - **TFT Match**", gameStarted.SummonerName)

		config := WagerCreationConfig{
			ExternalSystem:      entities.SystemTFT,
			GameID:              gameStarted.GameID,
			SummonerName:        gameStarted.SummonerName,
			TagLine:             gameStarted.TagLine,
			Condition:           condition,
			Options:             []string{"Top 4", "Bottom 4"}, // TFT-specific options
			OddsMultipliers:     []float64{2.0, 2.0},           // 2:1 odds for both
			VotingPeriodMinutes: 5,                             // 5 minutes for betting
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

		// TFT winner selector: Top 4 (placement <= 4) vs Bottom 4 (placement > 4)
		tftWinnerSelector := func(options []entities.GroupWagerOption, result interface{}) int64 {
			gameResult := result.(dto.TFTGameEndedDTO)
			isTop4 := gameResult.Placement <= 4
			
			for _, opt := range options {
				if (isTop4 && opt.OptionText == "Top 4") || (!isTop4 && opt.OptionText == "Bottom 4") {
					return opt.ID
				}
			}
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