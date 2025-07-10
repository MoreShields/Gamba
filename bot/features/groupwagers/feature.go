package groupwagers

import (
	"strings"

	"gambler/bot/common"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
)

// Feature represents the group wagers feature
type Feature struct {
	session           *discordgo.Session
	groupWagerService service.GroupWagerService
	userService       service.UserService
	guildID           string
}

// NewFeature creates a new group wagers feature instance
func NewFeature(session *discordgo.Session, groupWagerService service.GroupWagerService, userService service.UserService, guildID string) *Feature {
	return &Feature{
		session:           session,
		groupWagerService: groupWagerService,
		userService:       userService,
		guildID:           guildID,
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
	}
}

// handleComponentInteraction routes button clicks based on custom ID
func (f *Feature) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "_")
	
	if len(parts) < 3 {
		common.RespondWithError(s, i, "Invalid interaction data")
		return
	}

	action := parts[1]
	switch action {
	case "bet":
		f.handleGroupWagerButtonInteraction(s, i)
	case "resolve":
		// Admin resolve button (if implemented)
		common.RespondWithError(s, i, "Please use the /groupwager resolve command")
	case "cancel":
		// Admin cancel button (if implemented)
		common.RespondWithError(s, i, "Cancellation not yet implemented")
	default:
		common.RespondWithError(s, i, "Unknown action")
	}
}

// handleModalSubmit handles the group wager creation modal
func (f *Feature) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	if customID == "groupwager_create_modal" {
		f.handleGroupWagerCreateModal(s, i)
	}
}