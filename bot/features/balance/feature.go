package balance

import (
	"github.com/bwmarrin/discordgo"
	"gambler/service"
)

type Feature struct {
	userService service.UserService
}

func New(userService service.UserService) *Feature {
	return &Feature{
		userService: userService,
	}
}

func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleBalance(s, i)
}