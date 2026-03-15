package betwin

import (
	"gambler/discord-client/application"

	"github.com/bwmarrin/discordgo"
)

// Feature handles the /betwin slash command for pre-staging bets on summoners
type Feature struct {
	uowFactory       application.UnitOfWorkFactory
	stagedBetHandler *application.StagedBetHandler
}

// New creates a new betwin feature
func New(uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		uowFactory:       uowFactory,
		stagedBetHandler: application.NewStagedBetHandler(uowFactory),
	}
}

// HandleCommand routes the /betwin slash command
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleBetwin(s, i)
}

// HandleAutocomplete handles autocomplete requests for the summoner option
func (f *Feature) HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleAutocomplete(s, i)
}
