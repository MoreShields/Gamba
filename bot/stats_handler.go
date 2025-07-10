package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleStatsCommand handles the /stats command with subcommands
func (b *Bot) handleStatsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		b.respondWithError(s, i, "Please specify a subcommand: scoreboard or balance")
		return
	}

	switch options[0].Name {
	case "scoreboard":
		b.handleStatsScoreboard(s, i)
	case "balance":
		b.handleStatsBalance(s, i, options[0].Options)
	default:
		b.respondWithError(s, i, "Unknown subcommand")
	}
}

// handleStatsScoreboard displays the global scoreboard
func (b *Bot) handleStatsScoreboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get scoreboard entries (top 10)
	entries, err := b.statsService.GetScoreboard(ctx, 10)
	if err != nil {
		log.Printf("Error getting scoreboard: %v", err)
		b.respondWithError(s, i, "Unable to retrieve scoreboard. Please try again.")
		return
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title: "🏆 Gamba Scoreboard",
		Color: 0x00ff00,
	}

	if len(entries) == 0 {
		embed.Description = "No players with positive balance found."
	} else {
		// Build scoreboard content as a table
		var tableContent strings.Builder
		
		// Add header
		tableContent.WriteString("```\n")
		tableContent.WriteString(fmt.Sprintf("%-4s %-20s %-12s %s\n", "Rank", "Player", "Balance", "Wagers"))
		tableContent.WriteString(strings.Repeat("-", 50) + "\n")
		
		for _, entry := range entries {
			// Format rank with medal for top 3
			rankStr := fmt.Sprintf("#%d", entry.Rank)
			if entry.Rank == 1 {
				rankStr = "🥇"
			} else if entry.Rank == 2 {
				rankStr = "🥈"
			} else if entry.Rank == 3 {
				rankStr = "🥉"
			}

			// Get display name for the user
			displayName := GetDisplayNameInt64(s, i.GuildID, entry.DiscordID)
			// Truncate name if too long
			if len(displayName) > 18 {
				displayName = displayName[:15] + "..."
			}

			// Format balance
			balanceStr := FormatBalance(entry.TotalBalance)
			
			// Format wager stats
			var wagerStr string
			if entry.ActiveWagerCount > 0 || entry.WagerWinRate > 0 {
				wagerStr = fmt.Sprintf("%d active, %.1f%%", entry.ActiveWagerCount, entry.WagerWinRate)
			} else {
				wagerStr = "-"
			}
			
			// Add row
			tableContent.WriteString(fmt.Sprintf("%-4s %-20s %-12s %s\n", 
				rankStr, displayName, balanceStr, wagerStr))
		}
		
		tableContent.WriteString("```")
		embed.Description = tableContent.String()
	}

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to scoreboard command: %v", err)
	}
}

// handleStatsBalance displays individual user statistics
func (b *Bot) handleStatsBalance(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()

	// Get target user (default to command issuer)
	var targetID int64
	var targetUser *discordgo.User

	if len(options) > 0 && options[0].Name == "user" {
		targetUser = options[0].UserValue(s)
		parsedID, err := strconv.ParseInt(targetUser.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing Discord ID %s: %v", targetUser.ID, err)
			b.respondWithError(s, i, "Unable to process request. Please try again.")
			return
		}
		targetID = parsedID
	} else {
		// Default to command issuer
		parsedID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
			b.respondWithError(s, i, "Unable to process request. Please try again.")
			return
		}
		targetID = parsedID
		targetUser = i.Member.User
	}

	// Get user stats
	stats, err := b.statsService.GetUserStats(ctx, targetID)
	if err != nil {
		log.Printf("Error getting user stats for %d: %v", targetID, err)
		b.respondWithError(s, i, "Unable to retrieve user statistics. Please try again.")
		return
	}

	// Get display name
	displayName := GetDisplayNameInt64(s, i.GuildID, targetID)

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 Statistics for %s", displayName),
		Color: 0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "💰 Balance",
				Value: fmt.Sprintf("Total: **%s bits**\nAvailable: **%s bits**\nReserved: %s bits",
					FormatBalance(stats.User.Balance),
					FormatBalance(stats.User.AvailableBalance),
					FormatBalance(stats.ReservedInWagers)),
				Inline: false,
			},
		},
	}

	// Add bet statistics if any
	if stats.BetStats.TotalBets > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "🎲 Betting Statistics",
			Value: fmt.Sprintf("Win Rate: **%.1f%%** (%d/%d)\nTotal Wagered: %s bits\nNet Profit: %s bits",
				stats.BetStats.WinPercentage,
				stats.BetStats.TotalWins,
				stats.BetStats.TotalBets,
				FormatBalance(stats.BetStats.TotalWagered),
				FormatBalance(stats.BetStats.NetProfit)),
			Inline: true,
		})
	}

	// Add wager statistics if any
	if stats.WagerStats.TotalWagers > 0 {
		// Focus on resolved wagers for stats display
		var wagerStatsStr string
		if stats.WagerStats.TotalResolved > 0 {
			wagerStatsStr = fmt.Sprintf("Win Rate: **%.1f%%** (%d/%d)\nTotal Won: %s bits",
				stats.WagerStats.WinPercentage,
				stats.WagerStats.TotalWon,
				stats.WagerStats.TotalResolved,
				FormatBalance(stats.WagerStats.TotalWonAmount))
		} else {
			wagerStatsStr = fmt.Sprintf("Total Wagers: %d\nNo resolved wagers yet",
				stats.WagerStats.TotalWagers)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🤝 Wager Statistics",
			Value:  wagerStatsStr,
			Inline: true,
		})
	}

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to balance stats command: %v", err)
	}
}
