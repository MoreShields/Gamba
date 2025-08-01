package application

import (
	"context"
	"gambler/discord-client/application/dto"
)

// PostResult contains the result of posting a message to Discord
type PostResult struct {
	MessageID int64
	ChannelID int64
}

// DiscordPoster defines the interface for posting messages to Discord
// This abstraction allows the application layer to communicate with Discord
// without direct dependency on the Discord API
type DiscordPoster interface {
	// PostHouseWager posts a house wager to the appropriate Discord channel
	// Returns the messageID and channelID of the posted message, or an error if posting fails
	PostHouseWager(ctx context.Context, dto dto.HouseWagerPostDTO) (*PostResult, error)

	// UpdateHouseWager updates an existing house wager message in Discord
	UpdateHouseWager(ctx context.Context, messageID, channelID int64, dto dto.HouseWagerPostDTO) error

	// UpdateGroupWager updates an existing group wager message in Discord
	UpdateGroupWager(ctx context.Context, messageID, channelID int64, detail interface{}) error

	// PostDailyAwards posts daily awards summary to the appropriate Discord channel
	PostDailyAwards(ctx context.Context, dto dto.DailyAwardsPostDTO) error
}

// WagerStateEventHandler defines the interface for handling internal wager state change events
// This handler processes events from the service layer and orchestrates Discord updates
type WagerStateEventHandler interface {
	// HandleGroupWagerStateChange handles GroupWagerStateChangeEvent from internal service operations
	// It fetches updated wager data, creates appropriate DTOs, and updates Discord messages
	HandleGroupWagerStateChange(ctx context.Context, event interface{}) error
}

// LoLEventHandler defines the interface for handling LoL game events
// This interface receives domain DTOs, not raw bytes
type LoLEventHandler interface {
	// HandleGameStarted processes a game started event
	HandleGameStarted(ctx context.Context, gameStarted dto.GameStartedDTO) error

	// HandleGameEnded processes a game ended event
	HandleGameEnded(ctx context.Context, gameEnded dto.GameEndedDTO) error
}

// GuildDiscoveryService discovers guilds and their channel configurations
type GuildDiscoveryService interface {
	// GetGuildsWithPrimaryChannel returns all guilds that have a primary channel configured
	GetGuildsWithPrimaryChannel(ctx context.Context) ([]dto.GuildChannelInfo, error)
}

// UserResolver resolves Discord user information from various identifiers
type UserResolver interface {
	// ResolveUsersByNick finds Discord user IDs by their server nickname
	// Returns a slice since nicknames may not be unique
	// Returns empty slice if no users found with the given nickname
	ResolveUsersByNick(ctx context.Context, guildID int64, nickname string) ([]int64, error)
}
