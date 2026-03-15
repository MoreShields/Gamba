package entities

import "time"

// StagedBet represents a pre-committed bet on a summoner's next game.
// Funds are locked immediately and the bet is auto-placed when a house wager fires.
type StagedBet struct {
	ID         int64     `db:"id"`
	GuildID    int64     `db:"guild_id"`
	DiscordID  int64     `db:"discord_id"`
	SummonerID int64     `db:"summoner_id"`
	Amount     int64     `db:"amount"`
	CreatedAt  time.Time `db:"created_at"`
}
