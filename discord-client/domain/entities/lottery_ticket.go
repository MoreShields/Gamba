package entities

import (
	"time"
)

// LotteryTicket represents a single lottery ticket
type LotteryTicket struct {
	ID               int64     `db:"id"`
	DrawID           int64     `db:"draw_id"`
	GuildID          int64     `db:"guild_id"`
	DiscordID        int64     `db:"discord_id"`
	TicketNumber     int64     `db:"ticket_number"`
	PurchasePrice    int64     `db:"purchase_price"`
	PurchasedAt      time.Time `db:"purchased_at"`
	BalanceHistoryID int64     `db:"balance_history_id"`
}

// FormatBinary formats the ticket number as a binary string with padding
func (t *LotteryTicket) FormatBinary(difficulty int64) string {
	return FormatBinaryNumber(t.TicketNumber, difficulty)
}

// IsWinner checks if this ticket matches the winning number
func (t *LotteryTicket) IsWinner(winningNumber int64) bool {
	return t.TicketNumber == winningNumber
}

// LotteryParticipantInfo represents a summary of a participant's tickets
type LotteryParticipantInfo struct {
	DiscordID   int64 `db:"discord_id"`
	TicketCount int64 `db:"ticket_count"`
}
