package highroller

import (
	"gambler/discord-client/application"
	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature represents the high roller feature
type Feature struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
}

// NewFeature creates a new high roller feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
	}
}

// HandleCommand handles high roller commands
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return nil
	}

	switch options[0].Name {
	case "buy":
		return f.handleBuy(s, i)
	case "info":
		return f.handleInfo(s, i)
	default:
		log.Warnf("Unknown high roller subcommand: %s", options[0].Name)
		common.RespondWithError(s, i, "Unknown subcommand")
	}

	return nil
}

// HandleInteraction is not used for high roller feature (no button interactions)
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}