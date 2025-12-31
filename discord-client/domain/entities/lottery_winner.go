package entities

import "time"

// LotteryWinner represents a winner in a lottery draw
type LotteryWinner struct {
	ID               int64     `db:"id"`
	DrawID           int64     `db:"draw_id"`
	DiscordID        int64     `db:"discord_id"`
	TicketID         int64     `db:"ticket_id"`
	WinningAmount    int64     `db:"winning_amount"`
	BalanceHistoryID int64     `db:"balance_history_id"`
	CreatedAt        time.Time `db:"created_at"`
}
