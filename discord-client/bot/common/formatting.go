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

// FormatBalanceCompact formats a balance amount in compact form (e.g. 100k, 1.5M)
func FormatBalanceCompact(balance int64) string {
	if balance < 1000 {
		return fmt.Sprintf("%d", balance)
	} else if balance < 1000000 {
		// Format as k (thousands)
		thousands := float64(balance) / 1000.0
		if thousands == float64(int(thousands)) {
			return fmt.Sprintf("%.0fk", thousands)
		}
		return fmt.Sprintf("%.1fk", thousands)
	} else if balance < 1000000000 {
		// Format as M (millions)
		millions := float64(balance) / 1000000.0
		if millions == float64(int(millions)) {
			return fmt.Sprintf("%.0fM", millions)
		}
		return fmt.Sprintf("%.1fM", millions)
	} else {
		// Format as B (billions)
		billions := float64(balance) / 1000000000.0
		if billions == float64(int(billions)) {
			return fmt.Sprintf("%.0fB", billions)
		}
		return fmt.Sprintf("%.1fB", billions)
	}
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

// FormatDiscordMessageLink creates a Discord message link from guild, channel, and message IDs
func FormatDiscordMessageLink(guildID, channelID, messageID int64) string {
	return fmt.Sprintf("https://discord.com/channels/%d/%d/%d", guildID, channelID, messageID)
}

// FormatDuration formats a duration in a human-readable format
// Examples: "2d 14h 30m", "3h 45m", "45m", "1m"
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	return strings.Join(parts, " ")
}
