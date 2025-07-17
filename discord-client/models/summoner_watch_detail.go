package models

import (
	"time"
)

// SummonerWatchDetail represents a combined view of summoner and watch information
// Used for API responses that need both summoner details and watch metadata
type SummonerWatchDetail struct {
	// Watch information
	GuildID   int64     `db:"guild_id"`
	WatchedAt time.Time `db:"watched_at"`

	// Summoner information
	SummonerName string    `db:"game_name"`
	TagLine      string    `db:"tag_line"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
