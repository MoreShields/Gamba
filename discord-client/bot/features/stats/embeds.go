package stats

import (
	"context"
	"fmt"
	"strconv"
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
	PageBits   = "Bits"
	PageLoL    = "LoL"
	PageTFT    = "TFT"
	PageGamble = "Gamble"
)

// ScoreboardPages defines the available pages
var ScoreboardPages = []string{PageBits, PageLoL, PageTFT, PageGamble}

// Constants for leaderboard display
const MinGameWagersForLeaderboard = 5

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
	embed := &discordgo.MessageEmbed{
		Title: "🏆 Scoreboard 🏆",
		Color: common.ColorPrimary,
	}

	var imageData []byte
	switch currentPage {
	case PageBits:
		imageData = buildBitsPage(ctx, embed, entry, totalBits, guildID, userResolver)
	case PageLoL:
		imageData = buildGamePage(ctx, embed, metricsService, session, guildID, userResolver, "LoL")
	case PageTFT:
		imageData = buildGamePage(ctx, embed, metricsService, session, guildID, userResolver, "TFT")
	case PageGamble:
		imageData = buildGamblePage(ctx, embed, metricsService, session, guildID, userResolver)
	default:
		// Default to bits page if unknown page
		imageData = buildBitsPage(ctx, embed, entry, totalBits, guildID, userResolver)
	}

	return embed, imageData
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

// buildGamePage is a generic function to build game-specific wager leaderboard pages
func buildGamePage(ctx context.Context, embed *discordgo.MessageEmbed, metricsService interfaces.UserMetricsService, session *discordgo.Session, guildID string, userResolver application.UserResolver, gameName string) []byte {
	// Clear description
	embed.Description = ""

	// Get leaderboard data based on game type
	var entries []*entities.LOLLeaderboardEntry
	var totalBitsWagered int64
	var err error

	switch gameName {
	case "LoL":
		entries, totalBitsWagered, err = metricsService.GetLOLLeaderboard(ctx, MinGameWagersForLeaderboard)
	case "TFT":
		entries, totalBitsWagered, err = metricsService.GetTFTLeaderboard(ctx, MinGameWagersForLeaderboard)
	default:
		embed.Description = "⚠️ Unknown game type"
		return nil
	}

	if err != nil {
		log.WithError(err).Errorf("Failed to get %s leaderboard data", gameName)
		embed.Description = fmt.Sprintf("⚠️ Error loading %s wager data", gameName)
		return nil
	}

	if len(entries) == 0 {
		embed.Description = fmt.Sprintf("No %s wager data available yet\n\n*Minimum %d %s wagers to qualify*", gameName, MinGameWagersForLeaderboard, gameName)
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
	imageData, err := generator.GenerateGameScoreboard(entries, usernames)
	if err != nil {
		log.WithError(err).Errorf("Failed to generate %s scoreboard image", gameName)
		embed.Description = "⚠️ Error generating scoreboard image"
		return nil
	}

	// Set the image
	embed.Image = &discordgo.MessageEmbedImage{
		URL: "attachment://scoreboard.png",
	}

	// Add total wagered to description
	embed.Description = fmt.Sprintf("**Total %s Bits Wagered: %s**", gameName, common.FormatBalance(totalBitsWagered))

	return imageData
}

// buildGamblePage populates the embed with gambling leaderboard data
func buildGamblePage(ctx context.Context, embed *discordgo.MessageEmbed, metricsService interfaces.UserMetricsService, session *discordgo.Session, guildID string, userResolver application.UserResolver) []byte {
	// Clear description
	embed.Description = ""

	// Get gambling leaderboard data
	entries, totalBitsWagered, err := metricsService.GetGamblingLeaderboard(ctx, MinGameWagersForLeaderboard)
	if err != nil {
		log.WithError(err).Error("Failed to get gambling leaderboard data")
		embed.Description = "⚠️ Error loading gambling data"
		return nil
	}

	if len(entries) == 0 {
		embed.Description = fmt.Sprintf("No gambling data available yet\n\n*Minimum %d bets to qualify*", MinGameWagersForLeaderboard)
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
	imageData, err := generator.GenerateGamblingScoreboard(entries, usernames)
	if err != nil {
		log.WithError(err).Error("Failed to generate gambling scoreboard image")
		embed.Description = "⚠️ Error generating scoreboard image"
		return nil
	}

	// Set the image
	embed.Image = &discordgo.MessageEmbedImage{
		URL: "attachment://scoreboard.png",
	}

	// Add total wagered to description
	embed.Description = fmt.Sprintf("**Total Bits Wagered: %s**", common.FormatBalance(totalBitsWagered))

	return imageData
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
