package dto

// HouseWagerPostDTO contains all the information needed to post a house wager to Discord
type HouseWagerPostDTO struct {
	// Guild and channel information
	GuildID   int64
	ChannelID int64 // If 0, use guild's primary channel

	// Wager information
	WagerID     int64
	Title       string
	Description string
	Options     []WagerOptionDTO

	// Summoner information for context
	SummonerInfo SummonerInfoDTO
}

// WagerOptionDTO represents a single option in a house wager
type WagerOptionDTO struct {
	ID         int64
	Text       string
	Order      int16
	Multiplier float64
}

// SummonerInfoDTO contains information about the summoner and game
type SummonerInfoDTO struct {
	GameName  string
	TagLine   string
	QueueType string
	GameID    string
}