package transfer

import (
	"gambler/discord-client/application"
	"github.com/bwmarrin/discordgo"
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
