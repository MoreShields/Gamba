package stats

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gambler/discord-client/application"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"

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

// formatProfitLoss formats the profit/loss amount with appropriate markdown and compact formatting
func formatProfitLoss(amount int64) string {
	if amount >= 0 {
		return fmt.Sprintf("**+%s**", common.FormatBalanceCompact(amount))
	}
	// Negative amount already has minus sign from FormatBalanceCompact
	return fmt.Sprintf("*%s*", common.FormatBalanceCompact(amount))
}


// BuildScoreboardEmbed creates the scoreboard embed with pagination support
func BuildScoreboardEmbed(ctx context.Context, metricsService interfaces.UserMetricsService, entry []*entities.ScoreboardEntry, totalBits int64, session *discordgo.Session, guildID string, currentPage string, userResolver application.UserResolver) *discordgo.MessageEmbed {
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
		buildBitsPage(ctx, embed, entry, totalBits, guildID, userResolver)
	case PageLoL:
		buildLoLPage(ctx, embed, metricsService, session, guildID, userResolver)
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
func buildBitsPage(ctx context.Context, embed *discordgo.MessageEmbed, entry []*entities.ScoreboardEntry, totalBits int64, guildID string, userResolver application.UserResolver) {
	// Add page description
	embed.Description = ""

	if len(entry) == 0 {
		embed.Description += "No players found\n\n" +
			fmt.Sprintf("**Total Server Bits: %s**", common.FormatBalance(totalBits))
		return
	}

	// Find the highest volume and highest donations to mark with icons
	var highestVolume int64
	var highestDonations int64
	for _, user := range entry {
		if user.TotalVolume > highestVolume {
			highestVolume = user.TotalVolume
		}
		if user.TotalDonations > highestDonations {
			highestDonations = user.TotalDonations
		}
	}

	// Build the formatted table
	var tableContent strings.Builder
	tableContent.WriteString("```\n")
	tableContent.WriteString("User               Balance   Volume     Donated\n")
	tableContent.WriteString("─────────────────  ────────  ─────────  ─────────\n")

	for i, user := range entry {
		medal := getMedalForRank(i + 1)
		
		// Get username instead of using mention
		guildIDInt, _ := strconv.ParseInt(guildID, 10, 64)
		username, err := userResolver.GetUsernameByID(ctx, guildIDInt, user.DiscordID)
		if err != nil {
			username = fmt.Sprintf("User%d", user.DiscordID)
		}
		if len(username) > 15 {
			username = username[:12] + "..."
		}
		
		// Format values
		balanceStr := common.FormatBalanceCompact(user.TotalBalance)
		volumeStr := common.FormatBalanceCompact(user.TotalVolume)
		donationStr := common.FormatBalanceCompact(user.TotalDonations)
		
		// Add icons for highest values
		if user.TotalVolume == highestVolume && highestVolume > 0 {
			volumeStr = "🎲" + volumeStr
		}
		if user.TotalDonations == highestDonations && highestDonations > 0 {
			donationStr = "🎁" + donationStr
		}
		
		// Format the row with medal right next to username
		userWithMedal := fmt.Sprintf("%s %s", medal, username)
		tableContent.WriteString(fmt.Sprintf("%-17s  %8s  %9s  %9s\n",
			userWithMedal, balanceStr, volumeStr, donationStr))
	}
	
	tableContent.WriteString("```")

	// Add the table as a single field
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Leaderboard",
			Value:  tableContent.String(),
			Inline: false,
		},
	}

	// Add total server bits to description
	embed.Description = fmt.Sprintf("**Total Server Bits: %s**", common.FormatBalance(totalBits))
}

// buildLoLPage populates the embed with LoL wager leaderboard using real data
func buildLoLPage(ctx context.Context, embed *discordgo.MessageEmbed, metricsService interfaces.UserMetricsService, session *discordgo.Session, guildID string, userResolver application.UserResolver) {
	// Clear description
	embed.Description = ""

	// Get LoL leaderboard data with minimum wagers
	entries, totalBitsWagered, err := metricsService.GetLOLLeaderboard(ctx, MinLoLWagersForLeaderboard)
	if err != nil {
		log.WithError(err).Error("Failed to get LoL leaderboard data")
		embed.Description = "⚠️ Error loading LoL wager data"
		return
	}

	if len(entries) == 0 {
		embed.Description = fmt.Sprintf("No LoL wager data available yet\n\n*Minimum %d LoL wagers to qualify*", MinLoLWagersForLeaderboard)
		return
	}

	// Build the formatted table
	var tableContent strings.Builder
	tableContent.WriteString("```\n")
	tableContent.WriteString("User               P/L       Wagered   Win%\n")
	tableContent.WriteString("─────────────────  ────────  ────────  ────────────\n")

	for _, entry := range entries {
		medal := getMedalForRank(entry.Rank)
		
		// Get username instead of using mention
		guildIDInt, _ := strconv.ParseInt(guildID, 10, 64)
		username, err := userResolver.GetUsernameByID(ctx, guildIDInt, entry.DiscordID)
		if err != nil {
			username = fmt.Sprintf("User%d", entry.DiscordID)
		}
		if len(username) > 15 {
			username = username[:12] + "..."
		}
		
		// Format values
		profitLossStr := ""
		if entry.ProfitLoss >= 0 {
			profitLossStr = fmt.Sprintf("+%s", common.FormatBalanceCompact(entry.ProfitLoss))
		} else {
			profitLossStr = common.FormatBalanceCompact(entry.ProfitLoss)
		}
		
		wageredStr := common.FormatBalanceCompact(entry.TotalAmountWagered)
		winRateStr := fmt.Sprintf("%.1f%% (%d/%d)", entry.AccuracyPercentage, entry.CorrectPredictions, entry.TotalPredictions)
		
		// Format the row with medal right next to username
		userWithMedal := fmt.Sprintf("%s %s", medal, username)
		tableContent.WriteString(fmt.Sprintf("%-17s  %8s  %8s  %12s\n",
			userWithMedal, profitLossStr, wageredStr, winRateStr))
	}
	
	tableContent.WriteString("```")
	tableContent.WriteString(fmt.Sprintf("\n*Minimum %d wagers to qualify*", MinLoLWagersForLeaderboard))

	// Add the table as a single field
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "LoL Wager Leaderboard",
			Value:  tableContent.String(),
			Inline: false,
		},
	}

	// Add footer info to description
	embed.Description = fmt.Sprintf("**Total LoL Bits Wagered: %s**", common.FormatBalance(totalBitsWagered))
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
func BuildUserStatsEmbed(userStats *entities.UserStats, targetName string) *discordgo.MessageEmbed {
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
