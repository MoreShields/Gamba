package stats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/models"
	"gambler/discord-client/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Page names for scoreboard navigation
const (
	PageBits = "Bits"
	PageLoL  = "LoL"
)

// ScoreboardPages defines the available pages
var ScoreboardPages = []string{PageBits, PageLoL}

// Constants for leaderboard display
const MinLoLWagersForLeaderboard = 5

// getMedalForRank returns the appropriate medal emoji or rank number
func getMedalForRank(rank int) string {
	switch rank {
	case 1:
		return "🥇"
	case 2:
		return "🥈"
	case 3:
		return "🥉"
	default:
		return fmt.Sprintf("%d.", rank)
	}
}

// formatProfitLoss formats the profit/loss amount with appropriate markdown
func formatProfitLoss(amount int64) string {
	if amount >= 0 {
		return fmt.Sprintf("**+%s**", common.FormatBalance(amount))
	}
	// Negative amount already has minus sign from FormatBalance
	return fmt.Sprintf("*%s*", common.FormatBalance(amount))
}

// BuildScoreboardEmbed creates the scoreboard embed with pagination support
func BuildScoreboardEmbed(ctx context.Context, metricsService service.UserMetricsService, entry []*models.ScoreboardEntry, totalBits int64, session *discordgo.Session, guildID string, currentPage string) *discordgo.MessageEmbed {
	// Default to first page if invalid
	if currentPage != PageBits && currentPage != PageLoL {
		currentPage = PageBits
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🏆 Scoreboard 🏆",
		Color:  common.ColorPrimary,
		Footer: buildFooter(currentPage),
	}

	switch currentPage {
	case PageBits:
		buildBitsPage(embed, entry, totalBits)
	case PageLoL:
		buildLoLPage(ctx, embed, metricsService, session, guildID)
	}

	return embed
}

// buildFooter creates the footer with page navigation indicators
func buildFooter(currentPage string) *discordgo.MessageEmbedFooter {
	var footerParts []string
	for _, page := range ScoreboardPages {
		if page == currentPage {
			footerParts = append(footerParts, fmt.Sprintf("[ %s ]", page))
		} else {
			footerParts = append(footerParts, page)
		}
	}

	return &discordgo.MessageEmbedFooter{
		Text: strings.Join(footerParts, " | "),
	}
}

// buildBitsPage populates the embed with bits scoreboard data
func buildBitsPage(embed *discordgo.MessageEmbed, entry []*models.ScoreboardEntry, totalBits int64) {
	// Add page description
	embed.Description = "**Current Balance Rankings**\n\n"

	if len(entry) == 0 {
		embed.Description += "No players found\n\n" +
			fmt.Sprintf("**Total Server Bits: %s**", common.FormatBalance(totalBits))
		return
	}

	var lines []string
	for i, user := range entry {
		medal := getMedalForRank(i + 1)
		lines = append(lines, fmt.Sprintf("%s <@%d> | %s",
			medal, user.DiscordID, common.FormatBalance(user.TotalBalance)))
	}

	embed.Description += strings.Join(lines, "\n") + "\n\n" +
		fmt.Sprintf("**Total Server Bits: %s**", common.FormatBalance(totalBits))
}

// buildLoLPage populates the embed with LoL wager leaderboard using real data
func buildLoLPage(ctx context.Context, embed *discordgo.MessageEmbed, metricsService service.UserMetricsService, session *discordgo.Session, guildID string) {
	// Add page description with table header
	embed.Description = "**LoL Wager Stats**\n" +
		"*Rank User | Win% (W/L) | Total Wagered (P/L)*\n\n"

	// Get LoL leaderboard data with minimum wagers
	entries, totalBitsWagered, err := metricsService.GetLOLLeaderboard(ctx, MinLoLWagersForLeaderboard)
	if err != nil {
		log.WithError(err).Error("Failed to get LoL leaderboard data")
		embed.Description = "⚠️ Error loading LoL wager data"
		return
	}

	if len(entries) == 0 {
		embed.Description += fmt.Sprintf("No LoL wager data available yet\n\n*Minimum %d LoL wagers to qualify*", MinLoLWagersForLeaderboard)
		return
	}

	var lines []string

	for _, entry := range entries {
		medal := getMedalForRank(entry.Rank)

		profitLossStr := formatProfitLoss(entry.ProfitLoss)

		lines = append(lines, fmt.Sprintf("%s <@%d> | **%.1f%%** (%d/%d) | %s (%s)",
			medal, entry.DiscordID, entry.AccuracyPercentage, entry.CorrectPredictions, entry.TotalPredictions,
			common.FormatBalance(entry.TotalAmountWagered), profitLossStr))
	}

	embed.Description += strings.Join(lines, "\n") + "\n\n" +
		fmt.Sprintf("**Total LoL Bits Wagered: %s**\n\n", common.FormatBalance(totalBitsWagered)) +
		fmt.Sprintf("*Minimum %d LoL wagers to qualify*", MinLoLWagersForLeaderboard)
}

// GetPageFromFooter extracts the current page from the footer text
func GetPageFromFooter(footerText string) string {
	// Look for pattern [ PageName ]
	start := strings.Index(footerText, "[")
	end := strings.Index(footerText, "]")

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(footerText[start+1 : end])
	}

	// Default to first page if can't parse
	return PageBits
}

// findPageIndex returns the index of the current page in ScoreboardPages
func findPageIndex(currentPage string) int {
	for i, page := range ScoreboardPages {
		if page == currentPage {
			return i
		}
	}
	return 0 // Default to first page
}

// GetNextPage returns the next page in the navigation
func GetNextPage(currentPage string) string {
	currentIndex := findPageIndex(currentPage)
	nextIndex := (currentIndex + 1) % len(ScoreboardPages)
	return ScoreboardPages[nextIndex]
}

// GetPreviousPage returns the previous page in the navigation
func GetPreviousPage(currentPage string) string {
	currentIndex := findPageIndex(currentPage)
	prevIndex := (currentIndex - 1 + len(ScoreboardPages)) % len(ScoreboardPages)
	return ScoreboardPages[prevIndex]
}

// BuildUserStatsEmbed creates the user statistics embed
func BuildUserStatsEmbed(userStats *models.UserStats, targetName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("📊 Stats for %s", targetName),
		Color:     common.ColorPrimary,
		Timestamp: time.Now().Format(time.RFC3339),
		Fields:    []*discordgo.MessageEmbedField{},
	}

	// Balance info
	if userStats.User != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "💰 Balance",
			Value: fmt.Sprintf("Total: **%s bits**\nAvailable: **%s bits**\nReserved: **%s bits**",
				common.FormatBalance(userStats.User.Balance),
				common.FormatBalance(userStats.User.AvailableBalance),
				common.FormatBalance(userStats.ReservedInWagers)),
			Inline: true,
		})
	}

	// Betting stats
	if userStats.BetStats != nil && userStats.BetStats.TotalBets > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "🎲 Betting Stats",
			Value: fmt.Sprintf("Total Bets: **%d**\nWins: **%d** (%.1f%%)\nNet P/L: **%s bits**",
				userStats.BetStats.TotalBets,
				userStats.BetStats.TotalWins,
				userStats.BetStats.WinPercentage,
				common.FormatBalance(userStats.BetStats.NetProfit)),
			Inline: true,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🎲 Betting Stats",
			Value:  "No bets placed yet",
			Inline: true,
		})
	}

	// Wager stats
	if userStats.WagerStats != nil && userStats.WagerStats.TotalWagers > 0 {
		var wagerValue string
		if userStats.WagerStats.TotalResolved > 0 {
			wagerValue = fmt.Sprintf("Total Wagers: **%d**\nWon: **%d** (%.1f%%)",
				userStats.WagerStats.TotalWagers,
				userStats.WagerStats.TotalWon,
				userStats.WagerStats.WinPercentage)
		} else {
			wagerValue = fmt.Sprintf("Total Wagers: **%d**\nNo resolved wagers yet",
				userStats.WagerStats.TotalWagers)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚔️ Wager Stats",
			Value:  wagerValue,
			Inline: true,
		})
	}

	return embed
}
