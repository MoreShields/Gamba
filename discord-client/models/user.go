package models

import (
	"time"
)

// User represents a Discord user with guild-specific balance information
type User struct {
	DiscordID        int64     `db:"discord_id"`
	Username         string    `db:"username"`
	Balance          int64     `db:"-"` // Populated from user_guild_accounts
	AvailableBalance int64     `db:"-"` // Calculated field: balance minus pending wagers
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}