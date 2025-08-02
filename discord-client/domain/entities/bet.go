package entities

import (
	"errors"
	"math"
	"time"
)

// Bet represents a gambling bet
type Bet struct {
	ID               int64    `db:"id"`
	DiscordID        int64    `db:"discord_id"`
	GuildID          int64    `db:"guild_id"`
	Amount           int64    `db:"amount"`
	WinProbability   float64  `db:"win_probability"`
	Won              bool     `db:"won"`
	WinAmount        int64    `db:"win_amount"`
	BalanceHistoryID *int64   `db:"balance_history_id"`
	CreatedAt        time.Time `db:"created_at"`
}

// BetResult represents the outcome of a bet (returned to the user)
type BetResult struct {
	Won        bool
	BetAmount  int64
	WinAmount  int64
	NewBalance int64
}

// CalculateWinAmount calculates the potential win amount based on probability
func (b *Bet) CalculateWinAmount() int64 {
	if b.WinProbability <= 0 {
		return 0
	}
	// Win amount = bet_amount / probability (rounded down)
	return int64(math.Floor(float64(b.Amount) / b.WinProbability))
}

// IsWinning checks if the bet would be winning based on the current state
func (b *Bet) IsWinning() bool {
	return b.Won
}

// GetNetProfit returns the net profit/loss from this bet
func (b *Bet) GetNetProfit() int64 {
	if b.Won {
		return b.WinAmount - b.Amount
	}
	return -b.Amount
}

// ValidateBet performs basic validation on the bet
func (b *Bet) ValidateBet() error {
	if b.Amount <= 0 {
		return errors.New("bet amount must be positive")
	}
	
	if b.WinProbability <= 0 || b.WinProbability > 1 {
		return errors.New("win probability must be between 0 and 1")
	}
	
	if b.Won && b.WinAmount <= 0 {
		return errors.New("winning bet must have positive win amount")
	}
	
	return nil
}

// GetMultiplier returns the payout multiplier for this bet
func (b *Bet) GetMultiplier() float64 {
	if b.Amount == 0 {
		return 0
	}
	return float64(b.WinAmount) / float64(b.Amount)
}

// GetROI returns the return on investment as a percentage
func (b *Bet) GetROI() float64 {
	if b.Amount == 0 {
		return 0
	}
	return (float64(b.GetNetProfit()) / float64(b.Amount)) * 100
}

// IsResolved returns true if the bet has been resolved (win or loss determined)
func (b *Bet) IsResolved() bool {
	return b.WinAmount > 0 || !b.Won
}