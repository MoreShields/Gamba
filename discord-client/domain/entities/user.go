package entities

import (
	"errors"
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

// CanAfford checks if the user has sufficient available balance for an amount
func (u *User) CanAfford(amount int64) bool {
	return u.AvailableBalance >= amount
}

// HasPositiveBalance checks if the user has a positive balance
func (u *User) HasPositiveBalance() bool {
	return u.Balance > 0
}

// HasSufficientBalance checks if the user has sufficient total balance for an amount
func (u *User) HasSufficientBalance(amount int64) bool {
	return u.Balance >= amount
}

// ValidateAmount checks if an amount is valid (positive and affordable)
func (u *User) ValidateAmount(amount int64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if !u.CanAfford(amount) {
		return errors.New("insufficient available balance")
	}
	return nil
}

// GetPendingAmount calculates the amount tied up in pending wagers
func (u *User) GetPendingAmount() int64 {
	return u.Balance - u.AvailableBalance
}

// HasAvailableBalance checks if the user has any available balance
func (u *User) HasAvailableBalance() bool {
	return u.AvailableBalance > 0
}

// CalculateNewBalance calculates what the balance would be after a change
func (u *User) CalculateNewBalance(changeAmount int64) int64 {
	return u.Balance + changeAmount
}

// CalculateNewAvailableBalance calculates what the available balance would be after a change
func (u *User) CalculateNewAvailableBalance(changeAmount int64) int64 {
	return u.AvailableBalance + changeAmount
}