package settings

import (
	"gambler/discord-client/service"

	"github.com/bwmarrin/discordgo"
)

// Feature handles guild settings management
type Feature struct {
	session    *discordgo.Session
	uowFactory service.UnitOfWorkFactory
}

// NewFeature creates a new settings feature instance
func NewFeature(session *discordgo.Session, uowFactory service.UnitOfWorkFactory) *Feature {
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
	}
}