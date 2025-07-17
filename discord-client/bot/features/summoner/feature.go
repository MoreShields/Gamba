package summoner

import (
	"github.com/bwmarrin/discordgo"
	"gambler/discord-client/service"
	summoner_pb "gambler/discord-client/proto/services"
)

// Feature handles summoner watch commands and interactions
type Feature struct {
	session        *discordgo.Session
	uowFactory     service.UnitOfWorkFactory
	summonerClient summoner_pb.SummonerTrackingServiceClient
	guildID        string
}

// NewFeature creates a new summoner feature instance
func NewFeature(session *discordgo.Session, uowFactory service.UnitOfWorkFactory, summonerClient summoner_pb.SummonerTrackingServiceClient, guildID string) *Feature {
	return &Feature{
		session:        session,
		uowFactory:     uowFactory,
		summonerClient: summonerClient,
		guildID:        guildID,
	}
}

// HandleCommand handles summoner-related slash commands
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	
	// Handle subcommands
	if len(data.Options) > 0 {
		switch data.Options[0].Name {
		case "watch":
			f.handleWatchCommand(s, i)
		case "unwatch":
			f.handleUnwatchCommand(s, i)
		}
	}
}