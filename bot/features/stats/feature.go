package stats

import (
	"gambler/bot/common"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
)

// Feature represents the stats feature
type Feature struct {
	session      *discordgo.Session
	statsService service.StatsService
	userService  service.UserService
	guildID      string
}

// NewFeature creates a new stats feature instance
func NewFeature(session *discordgo.Session, statsService service.StatsService, userService service.UserService, guildID string) *Feature {
	return &Feature{
		session:      session,
		statsService: statsService,
		userService:  userService,
		guildID:      guildID,
	}
}

// HandleCommand handles the /stats command and its subcommands
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please specify a subcommand: scoreboard or balance")
		return
	}

	// Route to appropriate subcommand handler
	switch options[0].Name {
	case "scoreboard":
		f.handleStatsScoreboard(s, i)
	case "balance":
		f.handleStatsBalance(s, i, options[0].Options)
	default:
		common.RespondWithError(s, i, "Unknown subcommand")
	}
}