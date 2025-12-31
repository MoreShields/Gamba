package entities

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// LotteryDraw represents a single lottery draw event
type LotteryDraw struct {
	ID            int64      `db:"id"`
	GuildID       int64      `db:"guild_id"`
	Difficulty    int64      `db:"difficulty"`      // Captured from guild settings at creation
	TicketCost    int64      `db:"ticket_cost"`     // Captured from guild settings at creation
	WinningNumber *int64     `db:"winning_number"`  // NULL until draw completes
	DrawTime      time.Time  `db:"draw_time"`       // When the draw will occur
	TotalPot      int64      `db:"total_pot"`       // Total pot amount
	CompletedAt   *time.Time `db:"completed_at"`    // NULL until draw completes
	MessageID     *int64     `db:"message_id"`      // Discord message ID for the lottery embed
	ChannelID     *int64     `db:"channel_id"`      // Discord channel ID
	CreatedAt     time.Time  `db:"created_at"`
}

// IsCompleted returns true if the draw has been completed
func (d *LotteryDraw) IsCompleted() bool {
	return d.CompletedAt != nil
}

// CanPurchaseTickets returns true if tickets can still be purchased
func (d *LotteryDraw) CanPurchaseTickets() bool {
	return !d.IsCompleted() && time.Now().Before(d.DrawTime)
}

// GetMaxNumber returns the maximum ticket number (2^difficulty - 1)
func (d *LotteryDraw) GetMaxNumber() int64 {
	return (1 << d.Difficulty) - 1
}

// GetTotalNumbers returns the total count of possible numbers (2^difficulty)
func (d *LotteryDraw) GetTotalNumbers() int64 {
	return 1 << d.Difficulty
}

// GenerateWinningNumber generates a cryptographically random winning number
// within the range [0, 2^difficulty - 1]
func (d *LotteryDraw) GenerateWinningNumber() (int64, error) {
	max := big.NewInt(d.GetTotalNumbers())
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("failed to generate winning number: %w", err)
	}
	return n.Int64(), nil
}

// Complete marks the draw as completed with the given winning number
func (d *LotteryDraw) Complete(winningNumber int64) {
	d.WinningNumber = &winningNumber
	now := time.Now()
	d.CompletedAt = &now
}

// SetMessage sets the Discord message tracking info
func (d *LotteryDraw) SetMessage(channelID, messageID int64) {
	d.ChannelID = &channelID
	d.MessageID = &messageID
}

// HasMessage returns true if the draw has a tracked Discord message
func (d *LotteryDraw) HasMessage() bool {
	return d.MessageID != nil && d.ChannelID != nil
}

// FormatBinaryNumber formats a number as a binary string with padding
func FormatBinaryNumber(number, difficulty int64) string {
	format := fmt.Sprintf("%%0%db", difficulty)
	return fmt.Sprintf(format, number)
}
