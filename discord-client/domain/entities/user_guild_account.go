package entities

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

// CanAfford checks if the user has sufficient available balance for an amount
func (u *UserGuildAccount) CanAfford(amount int64) bool {
	return u.AvailableBalance >= amount
}

// HasPositiveBalance checks if the user has a positive balance
func (u *UserGuildAccount) HasPositiveBalance() bool {
	return u.Balance > 0
}