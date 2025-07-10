package models

import "time"

// Bet represents a gambling bet in the database
type Bet struct {
	ID               int64     `db:"id"`
	DiscordID        int64     `db:"discord_id"`
	Amount           int64     `db:"amount"`
	WinProbability   float64   `db:"win_probability"`
	Won              bool      `db:"won"`
	WinAmount        int64     `db:"win_amount"`
	BalanceHistoryID *int64    `db:"balance_history_id"`
	CreatedAt        time.Time `db:"created_at"`
}

// BetResult represents the outcome of a bet (returned to the user)
type BetResult struct {
	Won        bool
	BetAmount  int64
	WinAmount  int64
	NewBalance int64
}

// TransferResult represents the outcome of a transfer (returned to the user)
type TransferResult struct {
	Amount        int64
	RecipientName string
	NewBalance    int64
}