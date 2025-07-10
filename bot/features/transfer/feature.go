package transfer

import (
	"github.com/bwmarrin/discordgo"
	"gambler/service"
)

type Feature struct {
	transferService service.TransferService
	userService     service.UserService
}

func New(transferService service.TransferService, userService service.UserService) *Feature {
	return &Feature{
		transferService: transferService,
		userService:     userService,
	}
}

func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleDonate(s, i)
}