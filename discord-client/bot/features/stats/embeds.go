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
func BuildScoreboardEmbed(ctx context.Context, metricsService interfaces.UserMetricsService, entry []*entities.ScoreboardEntry, totalBits int64, session *discordgo.Session, guildID string, currentPage string, userResolver application.UserResolver) (*discordgo.MessageEmbed, []byte) {
	// Default to first page if invalid
	if currentPage != PageBits && currentPage != PageLoL {
		currentPage = PageBits
	}

	embed := &discordgo.MessageEmbed{
		Title: "🏆 Scoreboard 🏆",
		Color: common.ColorPrimary,
	}

	var imageData []byte
	switch currentPage {
	case PageBits:
		imageData = buildBitsPage(ctx, embed, entry, totalBits, guildID, userResolver)
	case PageLoL:
		imageData = buildLoLPage(ctx, embed, metricsService, session, guildID, userResolver)
	}

	// Set footer after page build (so LoL page can add extra text)
	embed.Footer = buildFooter(currentPage)

	return embed, imageData
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

// buildBitsPage populates the embed with bits scoreboard data and returns image data
func buildBitsPage(ctx context.Context, embed *discordgo.MessageEmbed, entry []*entities.ScoreboardEntry, totalBits int64, guildID string, userResolver application.UserResolver) []byte {
	// Add page description
	embed.Description = ""

	if len(entry) == 0 {
		embed.Description += "No players found\n\n" +
			fmt.Sprintf("**Total Server Bits: %s**", common.FormatBalance(totalBits))
		return nil
	}

	// Populate usernames for the entries
	guildIDInt, _ := strconv.ParseInt(guildID, 10, 64)
	for _, user := range entry {
		username, err := userResolver.GetUsernameByID(ctx, guildIDInt, user.DiscordID)
		if err != nil {
			username = fmt.Sprintf("User%d", user.DiscordID)
		}
		user.Username = username
	}

	// Generate the image
	generator := NewScoreboardImageGenerator()
	imageData, err := generator.GenerateBitsScoreboard(entry)
	if err != nil {
		log.WithError(err).Error("Failed to generate bits scoreboard image")
		embed.Description = "⚠️ Error generating scoreboard image"
		return nil
	}

	// Set the image
	embed.Image = &discordgo.MessageEmbedImage{
		URL: "attachment://scoreboard.png",
	}

	// Add total server bits to description
	embed.Description = fmt.Sprintf("**Total Server Bits: %s**", common.FormatBalance(totalBits))

	return imageData
}

// buildLoLPage populates the embed with LoL wager leaderboard using real data
func buildLoLPage(ctx context.Context, embed *discordgo.MessageEmbed, metricsService interfaces.UserMetricsService, session *discordgo.Session, guildID string, userResolver application.UserResolver) []byte {
	// Clear description
	embed.Description = ""

	// Get LoL leaderboard data with minimum wagers
	entries, totalBitsWagered, err := metricsService.GetLOLLeaderboard(ctx, MinLoLWagersForLeaderboard)
	if err != nil {
		log.WithError(err).Error("Failed to get LoL leaderboard data")
		embed.Description = "⚠️ Error loading LoL wager data"
		return nil
	}

	if len(entries) == 0 {
		embed.Description = fmt.Sprintf("No LoL wager data available yet\n\n*Minimum %d LoL wagers to qualify*", MinLoLWagersForLeaderboard)
		return nil
	}

	// Create username map for entries
	guildIDInt, _ := strconv.ParseInt(guildID, 10, 64)
	usernames := make(map[int64]string)
	for _, entry := range entries {
		username, err := userResolver.GetUsernameByID(ctx, guildIDInt, entry.DiscordID)
		if err != nil {
			username = fmt.Sprintf("User%d", entry.DiscordID)
		}
		usernames[entry.DiscordID] = username
	}

	// Generate the image
	generator := NewScoreboardImageGenerator()
	imageData, err := generator.GenerateLoLScoreboard(entries, usernames)
	if err != nil {
		log.WithError(err).Error("Failed to generate LoL scoreboard image")
		embed.Description = "⚠️ Error generating scoreboard image"
		return nil
	}

	// Set the image
	embed.Image = &discordgo.MessageEmbedImage{
		URL: "attachment://scoreboard.png",
	}

	// Add total wagered to description
	embed.Description = fmt.Sprintf("**Total LoL Bits Wagered: %s**", common.FormatBalance(totalBitsWagered))

	return imageData
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
