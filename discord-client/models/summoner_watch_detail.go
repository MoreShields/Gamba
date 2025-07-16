package models

import (
	"time"
)

// SummonerWatchDetail represents a combined view of summoner and watch information
// Used for API responses that need both summoner details and watch metadata
type SummonerWatchDetail struct {
	// Watch information
	WatchID   int64     `db:"watch_id"`
	GuildID   int64     `db:"guild_id"`
	WatchedAt time.Time `db:"watched_at"`
	
	// Summoner information
	SummonerID   int64     `db:"summoner_id"`
	SummonerName string    `db:"summoner_name"`
	Region       string    `db:"region"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}