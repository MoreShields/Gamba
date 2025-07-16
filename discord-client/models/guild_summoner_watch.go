package models

import (
	"time"
)

// GuildSummonerWatch represents the many-to-many relationship between guilds and summoners
type GuildSummonerWatch struct {
	ID          int64     `db:"id"`
	GuildID     int64     `db:"guild_id"`
	SummonerID  int64     `db:"summoner_id"`
	CreatedAt   time.Time `db:"created_at"`
}