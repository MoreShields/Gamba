package settings

import (
	"gambler/discord-client/application"

	"github.com/bwmarrin/discordgo"
)

// Feature handles guild settings management
type Feature struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
}

// NewFeature creates a new settings feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
	}
}

// HandleCommand routes settings commands to appropriate handlers
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return
	}

	switch options[0].Name {
	case "high-roller-role":
		f.handleHighRollerRole(s, i)
	case "primary-channel":
		f.handlePrimaryChannel(s, i)
	case "lol-channel":
		f.handleLolChannel(s, i)
	case "wordle-channel":
		f.handleWordleChannel(s, i)
	}
}
