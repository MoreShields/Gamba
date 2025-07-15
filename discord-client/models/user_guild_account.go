package models

import (
	"time"
)

// UserGuildAccount represents a user's account within a specific guild
type UserGuildAccount struct {
	ID               int64     `db:"id"`
	DiscordID        int64     `db:"discord_id"`
	GuildID          int64     `db:"guild_id"`
	Balance          int64     `db:"balance"`
	AvailableBalance int64     `db:"-"` // Calculated field: balance minus pending wagers
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}