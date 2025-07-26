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
}

// LoLHandler defines the interface for handling LoL game events
// This is implemented by the application layer and called by the infrastructure layer
type LoLHandler interface {
	// HandleGameStarted processes a game started event
	HandleGameStarted(ctx context.Context, gameStarted dto.GameStartedDTO) error
	
	// HandleGameEnded processes a game ended event
	HandleGameEnded(ctx context.Context, gameEnded dto.GameEndedDTO) error
}