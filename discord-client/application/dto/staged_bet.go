package dto

// PlaceStagedBetRequest contains the data needed to stage a bet on a summoner's next game
type PlaceStagedBetRequest struct {
	GuildID      int64
	DiscordID    int64
	Username     string
	SummonerName string
	TagLine      string
	Amount       int64
}

// PlaceStagedBetResponse contains the result of staging a bet
type PlaceStagedBetResponse struct {
	SummonerName     string
	TagLine          string
	Amount           int64
	RemainingBalance int64
	IsUpdate         bool // true if this updated an existing staged bet
}
