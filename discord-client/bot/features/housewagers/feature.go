package housewagers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature represents the house wagers feature
type Feature struct {
	session    *discordgo.Session
	uowFactory service.UnitOfWorkFactory
}

// NewFeature creates a new house wagers feature instance
func NewFeature(session *discordgo.Session, uowFactory service.UnitOfWorkFactory) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
	}
}

// PostHouseWager implements the application.DiscordPoster interface
func (f *Feature) PostHouseWager(ctx context.Context, dto dto.HouseWagerPostDTO) (*application.PostResult, error) {
	log.WithFields(log.Fields{
		"guild":   dto.GuildID,
		"channel": dto.ChannelID,
		"wagerID": dto.WagerID,
	}).Info("Posting house wager to Discord")

	// Validate channel ID
	if dto.ChannelID == 0 {
		return nil, fmt.Errorf("invalid channel ID: %d", dto.ChannelID)
	}

	// Create embed and components
	embed := CreateHouseWagerEmbed(dto)
	components := CreateHouseWagerComponents(dto)

	// Send message to Discord
	messageData := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}

	channelIDStr := fmt.Sprintf("%d", dto.ChannelID)
	message, err := f.session.ChannelMessageSendComplex(channelIDStr, messageData)
	if err != nil {
		return nil, fmt.Errorf("failed to send house wager message: %w", err)
	}

	// Parse message ID to int64
	messageID, err := strconv.ParseInt(message.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message ID: %w", err)
	}

	log.WithFields(log.Fields{
		"guild":     dto.GuildID,
		"channel":   dto.ChannelID,
		"wagerID":   dto.WagerID,
		"messageID": messageID,
	}).Info("Successfully posted house wager to Discord")

	return &application.PostResult{
		MessageID: messageID,
		ChannelID: dto.ChannelID,
	}, nil
}

// UpdateHouseWager implements the application.DiscordPoster interface
func (f *Feature) UpdateHouseWager(ctx context.Context, messageID, channelID int64, dto dto.HouseWagerPostDTO) error {
	log.WithFields(log.Fields{
		"messageID": messageID,
		"channel":   channelID,
		"wagerID":   dto.WagerID,
		"state":     dto.State,
	}).Info("Updating house wager message in Discord")

	// Validate parameters
	if messageID == 0 {
		return fmt.Errorf("invalid message ID: %d", messageID)
	}
	if channelID == 0 {
		return fmt.Errorf("invalid channel ID: %d", channelID)
	}

	// Create embed and components
	embed := CreateHouseWagerEmbed(dto)
	components := CreateHouseWagerComponents(dto)

	// Convert IDs to strings for Discord API
	channelIDStr := fmt.Sprintf("%d", channelID)
	messageIDStr := fmt.Sprintf("%d", messageID)

	// Update the message
	messageEdit := &discordgo.MessageEdit{
		Channel:    channelIDStr,
		ID:         messageIDStr,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}

	_, err := f.session.ChannelMessageEditComplex(messageEdit)
	if err != nil {
		return fmt.Errorf("failed to update house wager message: %w", err)
	}

	log.WithFields(log.Fields{
		"messageID": messageID,
		"channel":   channelID,
		"wagerID":   dto.WagerID,
		"state":     dto.State,
	}).Info("Successfully updated house wager message")

	return nil
}

// HandleInteraction handles house wager button interactions and modals
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		f.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		f.handleModalSubmit(s, i)
	default:
		log.Warnf("Unknown interaction type in housewagers: %v", i.Type)
	}
}

// handleComponentInteraction routes button clicks based on custom ID
func (f *Feature) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// House wager button interactions use format: house_wager_bet_<wager_id>_<option_id>
	if strings.HasPrefix(customID, "house_wager_bet_") {
		f.handleHouseWagerBetButton(s, i)
		return
	}

	log.Warnf("Unknown house wager component customID: %s", customID)
}

// handleModalSubmit handles the house wager bet modals
func (f *Feature) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	
	if strings.HasPrefix(customID, "house_wager_bet_modal_") {
		f.handleHouseWagerBetModal(s, i)
		return
	}

	log.Warnf("Unknown house wager modal customID: %s", customID)
	common.RespondWithError(s, i, "Unknown house wager modal")
}