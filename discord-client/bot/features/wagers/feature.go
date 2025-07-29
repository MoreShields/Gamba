package wagers

import (
	"strings"

	"gambler/discord-client/application"
	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
)

// Feature represents the wagers feature
type Feature struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
	guildID    string
}

// NewFeature creates a new wagers feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory, guildID string) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
		guildID:    guildID,
	}
}

// HandleCommand handles the /wager command and its subcommands
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Invalid command usage")
		return
	}

	// Route to appropriate subcommand handler
	switch options[0].Name {
	case "propose":
		f.handleWagerPropose(s, i)
	case "list":
		f.handleWagerList(s, i)
	case "cancel":
		f.handleWagerCancel(s, i)
	default:
		common.RespondWithError(s, i, "Unknown subcommand")
	}
}

// HandleInteraction handles wager-related component interactions and modals
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		f.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		f.handleModalSubmit(s, i)
	}
}

// handleComponentInteraction routes component interactions based on custom ID
func (f *Feature) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "_")

	if len(parts) < 3 {
		common.RespondWithError(s, i, "Invalid interaction data")
		return
	}

	action := parts[1]
	switch action {
	case "accept":
		f.handleWagerResponse(s, i, true)
	case "decline":
		f.handleWagerResponse(s, i, false)
	case "vote":
		f.handleWagerVote(s, i)
	default:
		common.RespondWithError(s, i, "Unknown wager action")
	}
}

// handleModalSubmit handles wager condition modal submissions
func (f *Feature) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	if strings.HasPrefix(customID, "wager_condition_modal_") {
		f.handleWagerConditionModal(s, i)
	}
}
