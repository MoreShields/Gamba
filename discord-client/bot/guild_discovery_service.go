package bot

import (
	"context"
	"strconv"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// GuildDiscoveryServiceImpl implements the GuildDiscoveryService interface
type GuildDiscoveryServiceImpl struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
}

// NewGuildDiscoveryService creates a new guild discovery service
func NewGuildDiscoveryService(
	session *discordgo.Session,
	uowFactory application.UnitOfWorkFactory,
) *GuildDiscoveryServiceImpl {
	return &GuildDiscoveryServiceImpl{
		session:    session,
		uowFactory: uowFactory,
	}
}

// GetGuildsWithPrimaryChannel returns all guilds with primary channels configured
func (g *GuildDiscoveryServiceImpl) GetGuildsWithPrimaryChannel(ctx context.Context) ([]dto.GuildChannelInfo, error) {
	var guildInfos []dto.GuildChannelInfo

	// Get all guilds the bot is in
	guilds := g.session.State.Guilds
	for _, guild := range guilds {
		guildID, err := strconv.ParseInt(guild.ID, 10, 64)
		if err != nil {
			log.Errorf("Error parsing guild ID %s: %v", guild.ID, err)
			continue
		}

		// Check if this guild has a primary channel
		channelID, err := g.getPrimaryChannelID(ctx, guildID)
		if err != nil {
			log.Errorf("Error getting primary channel for guild %d: %v", guildID, err)
			continue
		}

		guildInfos = append(guildInfos, dto.GuildChannelInfo{
			GuildID:          guildID,
			PrimaryChannelID: channelID,
		})
	}

	return guildInfos, nil
}

// getPrimaryChannelID gets the primary channel ID for a guild
func (g *GuildDiscoveryServiceImpl) getPrimaryChannelID(ctx context.Context, guildID int64) (*int64, error) {
	uow := g.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	guildSettingsService := services.NewGuildSettingsService(uow.GuildSettingsRepository())
	settings, err := guildSettingsService.GetOrCreateSettings(ctx, guildID)
	if err != nil {
		return nil, err
	}

	return settings.PrimaryChannelID, nil
}