package groupwagers

import (
	"strings"

	"gambler/bot/common"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature represents the group wagers feature
type Feature struct {
	session    *discordgo.Session
	uowFactory service.UnitOfWorkFactory
}

// NewFeature creates a new group wagers feature instance
func NewFeature(session *discordgo.Session, uowFactory service.UnitOfWorkFactory) *Feature {
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
