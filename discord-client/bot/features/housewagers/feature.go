package housewagers

import (
	"context"
	"fmt"
	"strings"

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
func (f *Feature) PostHouseWager(ctx context.Context, dto dto.HouseWagerPostDTO) error {
	log.WithFields(log.Fields{
		"guild":   dto.GuildID,
		"channel": dto.ChannelID,
		"wagerID": dto.WagerID,
	}).Info("Posting house wager to Discord")

	// Validate channel ID
	if dto.ChannelID == 0 {
		return fmt.Errorf("invalid channel ID: %d", dto.ChannelID)
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
		return fmt.Errorf("failed to send house wager message: %w", err)
	}

	log.WithFields(log.Fields{
		"guild":     dto.GuildID,
		"channel":   dto.ChannelID,
		"wagerID":   dto.WagerID,
		"messageID": message.ID,
	}).Info("Successfully posted house wager to Discord")

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