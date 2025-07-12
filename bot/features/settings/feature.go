package settings

import (
	"gambler/service"

	"github.com/bwmarrin/discordgo"
)

// Feature handles guild settings management
type Feature struct {
	session              *discordgo.Session
	guildSettingsService service.GuildSettingsService
}

// NewFeature creates a new settings feature instance
func NewFeature(session *discordgo.Session, guildSettingsService service.GuildSettingsService) *Feature {
	return &Feature{
		session:              session,
		guildSettingsService: guildSettingsService,
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