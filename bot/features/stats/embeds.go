package stats

import (
	"fmt"
	"strings"
	"time"

	"gambler/bot/common"
	"gambler/models"

	"github.com/bwmarrin/discordgo"
)

// BuildScoreboardEmbed creates the scoreboard embed
func BuildScoreboardEmbed(users []models.User, session *discordgo.Session, guildID string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "🏆 Top Players Scoreboard 🏆",
		Color:       common.ColorPrimary,
		Description: "",
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if len(users) == 0 {
		embed.Description = "No players found"
		return embed
	}

	var lines []string
	for i, user := range users {
		medal := ""
		switch i {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("%d.", i+1)
		}

		displayName := common.GetDisplayNameInt64(session, guildID, user.DiscordID)
		lines = append(lines, fmt.Sprintf("%s **%s** - %s bits",
			medal, displayName, common.FormatBalance(user.Balance)))
	}

	embed.Description = strings.Join(lines, "\n")
	return embed
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
			Name:   "💰 Balance",
			Value:  fmt.Sprintf("Total: **%s bits**\nAvailable: **%s bits**\nReserved: **%s bits**", 
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