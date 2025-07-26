package application

import (
	"context"
	"gambler/discord-client/application/dto"
)

// DiscordPoster defines the interface for posting messages to Discord
// This abstraction allows the application layer to communicate with Discord
// without direct dependency on the Discord API
type DiscordPoster interface {
	// PostHouseWager posts a house wager to the appropriate Discord channel
	// Returns an error if the posting fails
	PostHouseWager(ctx context.Context, dto dto.HouseWagerPostDTO) error
}

// LoLHandler defines the interface for handling LoL game events
// This is implemented by the application layer and called by the infrastructure layer
type LoLHandler interface {
	// HandleGameStarted processes a game started event
	HandleGameStarted(ctx context.Context, gameStarted dto.GameStartedDTO) error
	
	// HandleGameEnded processes a game ended event
	HandleGameEnded(ctx context.Context, gameEnded dto.GameEndedDTO) error
}