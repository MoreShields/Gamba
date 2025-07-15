package models

import (
	"time"
)

// WagerState represents the state of a wager
type WagerState string

const (
	WagerStateProposed WagerState = "proposed"
	WagerStateDeclined WagerState = "declined"
	WagerStateVoting   WagerState = "voting"
	WagerStateResolved WagerState = "resolved"
)

// Wager represents a public wager between two users
type Wager struct {
	ID                     int64      `db:"id"`
	ProposerDiscordID      int64      `db:"proposer_discord_id"`
	TargetDiscordID        int64      `db:"target_discord_id"`
	GuildID                int64      `db:"guild_id"`
	Amount                 int64      `db:"amount"`
	Condition              string     `db:"condition"`
	State                  WagerState `db:"state"`
	WinnerDiscordID        *int64     `db:"winner_discord_id"`
	WinnerBalanceHistoryID *int64     `db:"winner_balance_history_id"`
	LoserBalanceHistoryID  *int64     `db:"loser_balance_history_id"`
	MessageID              *int64     `db:"message_id"`
	ChannelID              *int64     `db:"channel_id"`
	CreatedAt              time.Time  `db:"created_at"`
	AcceptedAt             *time.Time `db:"accepted_at"`
	ResolvedAt             *time.Time `db:"resolved_at"`
}

// WagerResult represents the outcome of a wager operation
type WagerResult struct {
	Wager          *Wager
	WinnerID       int64
	LoserID        int64
	AmountWon      int64
	VotesForWinner int
	VotesForLoser  int
	TotalVotes     int
}

// IsParticipant checks if a user is involved in the wager
func (w *Wager) IsParticipant(discordID int64) bool {
	return w.ProposerDiscordID == discordID || w.TargetDiscordID == discordID
}

// GetOpponent returns the opponent's discord ID for a given participant
func (w *Wager) GetOpponent(discordID int64) int64 {
	if w.ProposerDiscordID == discordID {
		return w.TargetDiscordID
	}
	if w.TargetDiscordID == discordID {
		return w.ProposerDiscordID
	}
	return 0 // Not a participant
}

// CanBeAccepted checks if the wager can be accepted by the given user
func (w *Wager) CanBeAccepted(discordID int64) bool {
	return w.State == WagerStateProposed && w.TargetDiscordID == discordID
}

// CanBeCancelled checks if the wager can be cancelled by the given user
func (w *Wager) CanBeCancelled(discordID int64) bool {
	return w.State == WagerStateProposed && w.ProposerDiscordID == discordID
}

// IsActive checks if the wager is in an active state (not declined or resolved)
func (w *Wager) IsActive() bool {
	return w.State == WagerStateProposed || w.State == WagerStateVoting
}
