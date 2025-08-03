package application

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/services"

	log "github.com/sirupsen/logrus"
)

// BaseHouseWagerHandler provides common functionality for house wager systems
type BaseHouseWagerHandler struct {
	uowFactory    UnitOfWorkFactory
	discordPoster DiscordPoster
}

// NewBaseHouseWagerHandler creates a new base house wager handler
func NewBaseHouseWagerHandler(
	uowFactory UnitOfWorkFactory,
	discordPoster DiscordPoster,
) *BaseHouseWagerHandler {
	return &BaseHouseWagerHandler{
		uowFactory:    uowFactory,
		discordPoster: discordPoster,
	}
}

// WagerCreationConfig holds configuration for creating a house wager
type WagerCreationConfig struct {
	ExternalSystem      entities.ExternalSystem
	GameID              string
	SummonerName        string
	TagLine             string
	Condition           string
	Options             []string
	OddsMultipliers     []float64
	VotingPeriodMinutes int
	ChannelIDGetter     func(*entities.GuildSettings) *int64
	ChannelName         string // For error messages (e.g., "lol-channel", "tft-channel")
}

// CreateHouseWagerForGuild creates a house wager for a specific guild using the provided configuration
func (h *BaseHouseWagerHandler) CreateHouseWagerForGuild(
	ctx context.Context,
	guild *entities.GuildSummonerWatch,
	config WagerCreationConfig,
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
	groupWagerService := services.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Use nil for system-created house wagers (no specific creator)
	wagerDetail, err := groupWagerService.CreateGroupWager(
		ctx,
		nil,
		config.Condition,
		config.Options,
		config.VotingPeriodMinutes,
		0, // Message ID will be set after posting
		0, // Channel ID will be set after posting
		entities.GroupWagerTypeHouse,
		config.OddsMultipliers,
	)
	if err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to create group wager: %w", err)
	}

	// Set the external reference for this game
	wagerDetail.Wager.SetExternalReference(config.ExternalSystem, config.GameID)

	log.WithFields(log.Fields{
		"guild":          guild.GuildID,
		"wagerID":        wagerDetail.Wager.ID,
		"gameID":         config.GameID,
		"externalSystem": config.ExternalSystem,
	}).Debug("Setting external reference for house wager")

	// Update the wager with the external reference
	if err := uow.GroupWagerRepository().Update(ctx, wagerDetail.Wager); err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to update wager with external reference: %w", err)
	}

	// Build DTO for Discord posting
	channelID := int64(0)
	channelIDPtr := config.ChannelIDGetter(guildSettings)
	if channelIDPtr != nil {
		channelID = *channelIDPtr
	} else {
		uow.Rollback()
		return fmt.Errorf("failed to create group wager: %s is not set for guild %d", config.ChannelName, guild.GuildID)
	}

	// Build DTO using the helper function
	postDTO := h.BuildHouseWagerDTO(wagerDetail)
	// Override the channel ID since it might not be set in the wager yet
	postDTO.ChannelID = channelID
	// Ensure guild ID is set correctly (in case it's not set in the wager)
	postDTO.GuildID = guild.GuildID

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
		"summoner": fmt.Sprintf("%s#%s", config.SummonerName, config.TagLine),
	}).Info("Created house wager for game start")

	return nil
}

// WagerResolutionConfig holds configuration for resolving a house wager
type WagerResolutionConfig struct {
	ExternalSystem        entities.ExternalSystem
	WinnerSelector        func([]entities.GroupWagerOption, interface{}) int64
	GameResult            interface{}
	CancellationThreshold *int32 // nil means no cancellation logic
}

// ResolveHouseWager resolves a specific house wager using the provided configuration
func (h *BaseHouseWagerHandler) ResolveHouseWager(
	ctx context.Context,
	guildID, wagerID int64,
	config WagerResolutionConfig,
) error {
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
	groupWagerService := services.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Check for cancellation conditions if threshold is provided
	if config.CancellationThreshold != nil {
		if durationResult, ok := config.GameResult.(dto.GameEndedDTO); ok {
			if durationResult.DurationSeconds < *config.CancellationThreshold {
				return h.cancelWager(ctx, uow, groupWagerService, wagerDetail, guildID, wagerID, durationResult.DurationSeconds)
			}
		}
	}

	// Determine winning option using the provided selector
	// Convert []*entities.GroupWagerOption to []entities.GroupWagerOption
	options := make([]entities.GroupWagerOption, len(wagerDetail.Options))
	for i, opt := range wagerDetail.Options {
		if opt != nil {
			options[i] = *opt
		}
	}
	winningOptionID := config.WinnerSelector(options, config.GameResult)

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
	}).Debug("Attempting to resolve house wager")

	// For house wagers, use nil to indicate system resolution (no human resolver)
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

// cancelWager handles the cancellation logic for short games
func (h *BaseHouseWagerHandler) cancelWager(
	ctx context.Context,
	uow UnitOfWork,
	groupWagerService interfaces.GroupWagerService,
	wagerDetail *entities.GroupWagerDetail,
	guildID, wagerID int64,
	durationSeconds int32,
) error {
	log.WithFields(log.Fields{
		"guild":           guildID,
		"wagerID":         wagerID,
		"durationSeconds": durationSeconds,
	}).Info("Game ended with short duration (forfeit/remake), cancelling wager and refunding participants")

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

	// Store message info before committing
	messageID := wagerDetail.Wager.MessageID
	channelID := wagerDetail.Wager.ChannelID

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"guild":   guildID,
		"wagerID": wagerID,
	}).Info("Successfully cancelled house wager and refunded participants")

	// Update the Discord message to show cancelled state
	if messageID != 0 && channelID != 0 {
		// Update the state to cancelled since we know it was just cancelled
		wagerDetail.Wager.State = entities.GroupWagerStateCancelled

		// Build DTO for Discord update using the helper function
		updateDTO := h.BuildHouseWagerDTO(wagerDetail)

		// Update the Discord message
		if err := h.discordPoster.UpdateHouseWager(ctx, messageID, channelID, updateDTO); err != nil {
			log.WithFields(log.Fields{
				"guild":     guildID,
				"wagerID":   wagerID,
				"messageID": messageID,
				"channelID": channelID,
				"error":     err,
			}).Error("Failed to update Discord message for cancelled house wager")
		} else {
			log.WithFields(log.Fields{
				"guild":     guildID,
				"wagerID":   wagerID,
				"messageID": messageID,
			}).Info("Successfully updated Discord message for cancelled house wager")
		}
	}

	return nil
}

// BuildHouseWagerDTO builds a HouseWagerPostDTO from a GroupWagerDetail
func (h *BaseHouseWagerHandler) BuildHouseWagerDTO(detail *entities.GroupWagerDetail) dto.HouseWagerPostDTO {
	// Parse the condition to extract title and description
	parts := strings.SplitN(detail.Wager.Condition, "\n", 2)
	title := parts[0]
	description := ""
	if len(parts) > 1 {
		description = parts[1]
	}

	// Build DTO
	result := dto.HouseWagerPostDTO{
		GuildID:      detail.Wager.GuildID,
		ChannelID:    detail.Wager.ChannelID,
		WagerID:      detail.Wager.ID,
		Title:        title,
		Description:  description,
		State:        string(detail.Wager.State),
		Options:      make([]dto.WagerOptionDTO, len(detail.Options)),
		VotingEndsAt: detail.Wager.VotingEndsAt,
		Participants: make([]dto.ParticipantDTO, len(detail.Participants)),
		TotalPot:     detail.Wager.TotalPot,
	}

	// Convert options
	for i, opt := range detail.Options {
		result.Options[i] = dto.WagerOptionDTO{
			ID:          opt.ID,
			Text:        opt.OptionText,
			Order:       opt.OptionOrder,
			Multiplier:  opt.OddsMultiplier,
			TotalAmount: opt.TotalAmount,
		}
	}

	// Convert participants
	for i, participant := range detail.Participants {
		result.Participants[i] = dto.ParticipantDTO{
			DiscordID: participant.DiscordID,
			OptionID:  participant.OptionID,
			Amount:    participant.Amount,
		}
	}

	return result
}