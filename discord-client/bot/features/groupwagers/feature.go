package groupwagers

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/application"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/entities"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature represents the group wagers feature
type Feature struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
}

// NewFeature creates a new group wagers feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
	}
}

// HandleCommand handles the /groupwager command and its subcommands
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please specify a subcommand.")
		return
	}

	// Route to appropriate subcommand handler
	switch options[0].Name {
	case "create":
		f.handleGroupWagerCreate(s, i)
	case "resolve":
		f.handleGroupWagerResolve(s, i)
	case "cancel":
		f.handleGroupWagerCancel(s, i)
	default:
		common.RespondWithError(s, i, "Unknown subcommand.")
	}
}

// HandleInteraction handles group wager button interactions and modals
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		f.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		f.handleModalSubmit(s, i)
	default:
		log.Warnf("Unknown interaction type in groupwager: %v", i.Type)
	}
}

// handleComponentInteraction routes button clicks based on custom ID
func (f *Feature) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Group wager button interactions use format: group_wager_option_<wager_id>_<option_id>
	if strings.HasPrefix(customID, "group_wager_option_") {
		f.handleGroupWagerButtonInteraction(s, i)
		return
	}

}

// handleModalSubmit handles the group wager modals
func (f *Feature) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	switch {
	case customID == "group_wager_create_modal":

		f.handleGroupWagerCreateModal(s, i)
	case strings.HasPrefix(customID, "group_wager_bet_"):
		f.handleGroupWagerBetModal(s, i)
	default:
		log.Warnf("Unknown group wager modal customID: %s", customID)
		common.RespondWithError(s, i, "Unknown group wager modal")
	}
}

// UpdateGroupWager implements the application.DiscordPoster interface
func (f *Feature) UpdateGroupWager(ctx context.Context, messageID, channelID int64, detail interface{}) error {
	log.WithFields(log.Fields{
		"messageID": messageID,
		"channel":   channelID,
	}).Info("Updating group wager message in Discord")

	// Validate parameters
	if messageID == 0 {
		return fmt.Errorf("invalid message ID: %d", messageID)
	}
	if channelID == 0 {
		return fmt.Errorf("invalid channel ID: %d", channelID)
	}

	// Type assert the detail to GroupWagerDetail
	groupDetail, ok := detail.(*entities.GroupWagerDetail)
	if !ok {
		return fmt.Errorf("invalid detail type, expected *entities.GroupWagerDetail")
	}

	// Create embed and components
	embed := CreateGroupWagerEmbed(groupDetail)
	components := CreateGroupWagerComponents(groupDetail)

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
		return fmt.Errorf("failed to update group wager message: %w", err)
	}

	log.WithFields(log.Fields{
		"messageID": messageID,
		"channel":   channelID,
		"wagerID":   groupDetail.Wager.ID,
		"state":     groupDetail.Wager.State,
	}).Info("Successfully updated group wager message")

	return nil
}
