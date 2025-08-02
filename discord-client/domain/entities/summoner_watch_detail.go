package entities

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
	SummonerName string    `db:"summoner_name"`
	TagLine      string    `db:"tag_line"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// GetFullName returns the full summoner name in "SummonerName#TagLine" format
func (swd *SummonerWatchDetail) GetFullName() string {
	return swd.SummonerName + "#" + swd.TagLine
}

// IsValid checks if the detail has valid summoner information
func (swd *SummonerWatchDetail) IsValid() bool {
	return swd.SummonerName != "" && swd.TagLine != "" && swd.GuildID > 0
}