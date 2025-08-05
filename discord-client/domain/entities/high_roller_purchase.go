package entities

import (
	"errors"
	"time"
)

// HighRollerPurchase represents a purchase of the high roller role
type HighRollerPurchase struct {
	ID            int64     `db:"id"`
	GuildID       int64     `db:"guild_id"`
	DiscordID     int64     `db:"discord_id"`
	PurchasePrice int64     `db:"purchase_price"`
	PurchasedAt   time.Time `db:"purchased_at"`
}

// Validate ensures the purchase is valid
func (h *HighRollerPurchase) Validate() error {
	if h.GuildID <= 0 {
		return errors.New("invalid guild ID")
	}
	if h.DiscordID <= 0 {
		return errors.New("invalid discord ID")
	}
	if h.PurchasePrice < 0 {
		return errors.New("purchase price cannot be negative")
	}
	return nil
}

// IsMoreExpensiveThan checks if this purchase price is higher than another
func (h *HighRollerPurchase) IsMoreExpensiveThan(otherPrice int64) bool {
	return h.PurchasePrice > otherPrice
}

// GetMinimumNextPrice returns the minimum price for the next purchase
func (h *HighRollerPurchase) GetMinimumNextPrice() int64 {
	return h.PurchasePrice + 1
}