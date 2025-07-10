package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
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
func FormatTransferResult(amount int64, recipientName string, newBalance int64) string {
	return fmt.Sprintf("✅ Successfully transferred **%s bits** to **%s**.",
		FormatBalance(amount), recipientName)
}

// FormatDiscordTimestamp formats a time as a Discord timestamp that displays in user's local timezone
// Format types: "t" = short time, "T" = long time, "d" = short date, "D" = long date,
// "f" = short date/time, "F" = long date/time, "R" = relative time
func FormatDiscordTimestamp(t time.Time, format string) string {
	return fmt.Sprintf("<t:%d:%s>", t.Unix(), format)
}

// followUpWithError sends an error message as a follow-up to a deferred interaction
func (b *Bot) followUpWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: fmt.Sprintf("❌ %s", message),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Error sending follow-up error message: %v", err)
	}
}
