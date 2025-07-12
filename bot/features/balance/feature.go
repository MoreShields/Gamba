package balance

import (
	"github.com/bwmarrin/discordgo"
	"gambler/service"
)

type Feature struct {
	uowFactory service.UnitOfWorkFactory
}

func New(uowFactory service.UnitOfWorkFactory) *Feature {
	return &Feature{
		uowFactory: uowFactory,
	}
}

func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleBalance(s, i)
}