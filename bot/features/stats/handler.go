package stats

import (
	"context"
	"fmt"
	"strconv"

	"gambler/bot/common"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleStatsCommand handles the /stats command with subcommands
func (f *Feature) handleStatsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please specify a subcommand: scoreboard or balance")
		return
	}

	switch options[0].Name {
	case "scoreboard":
		f.handleStatsScoreboard(s, i)
	case "balance":
		f.handleStatsBalance(s, i, options[0].Options)
	default:
		common.RespondWithError(s, i, "Unknown subcommand")
	}
}

// handleStatsScoreboard displays the global scoreboard
func (f *Feature) handleStatsScoreboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get scoreboard entries (top 10)
	entries, err := f.statsService.GetScoreboard(ctx, 10)
	if err != nil {
		log.Printf("Error getting scoreboard: %v", err)
		common.RespondWithError(s, i, "Unable to retrieve scoreboard. Please try again.")
		return
	}

	// Create embed using the shared function
	embed := BuildScoreboardEmbed(entries, s, i.GuildID)

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
func (f *Feature) handleStatsBalance(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()

	// Get target user (default to command issuer)
	var targetID int64
	var targetUser *discordgo.User

	if len(options) > 0 && options[0].Name == "user" {
		targetUser = options[0].UserValue(s)
		parsedID, err := strconv.ParseInt(targetUser.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing Discord ID %s: %v", targetUser.ID, err)
			common.RespondWithError(s, i, "Unable to process request. Please try again.")
			return
		}
		targetID = parsedID
	} else {
		// Default to command issuer
		parsedID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
			common.RespondWithError(s, i, "Unable to process request. Please try again.")
			return
		}
		targetID = parsedID
		targetUser = i.Member.User
	}

	// Get user stats
	stats, err := f.statsService.GetUserStats(ctx, targetID)
	if err != nil {
		log.Printf("Error getting user stats for %d: %v", targetID, err)
		common.RespondWithError(s, i, "Unable to retrieve user statistics. Please try again.")
		return
	}

	// Get display name
	displayName := common.GetDisplayNameInt64(s, i.GuildID, targetID)

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 Statistics for %s", displayName),
		Color: 0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "💰 Balance",
				Value: fmt.Sprintf("Total: **%s bits**\nAvailable: **%s bits**\nReserved: %s bits",
					common.FormatBalance(stats.User.Balance),
					common.FormatBalance(stats.User.AvailableBalance),
					common.FormatBalance(stats.ReservedInWagers)),
				Inline: false,
			},
		},
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
				common.FormatBalance(stats.WagerStats.TotalWonAmount))
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
