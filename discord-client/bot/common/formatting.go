package common

import (
	"fmt"
	"strings"
	"time"
)

// FormatBalance formats a balance amount with thousand separators
func FormatBalance(balance int64) string {
	// Convert to string
	str := fmt.Sprintf("%d", balance)

	// Add commas for thousands
	n := len(str)
	if n <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// FormatBetResult formats the result of a bet
func FormatBetResult(won bool, betAmount, winAmount, newBalance int64) string {
	if won {
		return fmt.Sprintf("🎉 **You won!** You gained **%s bits**. New balance: **%s bits**",
			FormatBalance(winAmount), FormatBalance(newBalance))
	}
	return fmt.Sprintf("😔 **You lost!** You lost **%s bits**. New balance: **%s bits**",
		FormatBalance(betAmount), FormatBalance(newBalance))
}

// FormatTransferResult formats the result of a transfer
func FormatTransferResult(amount int64, recipientID string) string {
	return fmt.Sprintf("✅ donated **%s bits** to <@%s>",
		FormatBalance(amount), recipientID)
}

// FormatDiscordTimestamp formats a time as a Discord timestamp that displays in user's local timezone
// Format types: "t" = short time, "T" = long time, "d" = short date, "D" = long date,
// "f" = short date/time, "F" = long date/time, "R" = relative time
func FormatDiscordTimestamp(t time.Time, format string) string {
	return fmt.Sprintf("<t:%d:%s>", t.Unix(), format)
}
