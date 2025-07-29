package transfer

import (
	"github.com/bwmarrin/discordgo"
	"gambler/discord-client/application"
)

type Feature struct {
	uowFactory application.UnitOfWorkFactory
}

func New(uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		uowFactory: uowFactory,
	}
}

func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleDonate(s, i)
}