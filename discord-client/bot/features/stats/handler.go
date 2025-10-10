package stats

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/services"

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

	// Defer the response immediately to avoid timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Errorf("Error deferring interaction response: %v", err)
		return
	}

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate user metrics service with repositories from UnitOfWork
	metricsService := services.NewUserMetricsService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.BetRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
	)

	// Get scoreboard entries
	entries, totalBits, err := metricsService.GetScoreboard(ctx, 0)
	if err != nil {
		log.Printf("Error getting scoreboard: %v", err)
		common.FollowUpWithError(s, i, "Unable to retrieve scoreboard. Please try again.")
		return
	}

	// Get high roller info before creating embed
	var highRollerText string
	highRollerService := services.NewHighRollerService(
		uow.HighRollerPurchaseRepository(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	highRollerInfo, err := highRollerService.GetCurrentHighRoller(ctx, guildID)
	if err == nil && highRollerInfo.CurrentHolder != nil {
		// Get the role ID for mention
		guildSettingsService := services.NewGuildSettingsService(uow.GuildSettingsRepository())
		if roleID, err := guildSettingsService.GetHighRollerRoleID(ctx, guildID); err == nil && roleID != nil {
			highRollerText = fmt.Sprintf("\n<@&%d> - <@%d> - %s", *roleID, highRollerInfo.CurrentHolder.DiscordID, common.FormatBalance(highRollerInfo.CurrentPrice))
		}
	}

	// Create embed using the shared function (start with first page)
	embed, imageData := BuildScoreboardEmbed(ctx, metricsService, entries, totalBits, s, i.GuildID, PageBits, f.userResolver)

	// Add high roller info to the description if available
	if highRollerText != "" && embed.Description != "" {
		embed.Description += highRollerText
	}

	// Commit the transaction after building the embed
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Send follow-up message with the actual content
	webhookData := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}

	// Add image if generated
	if imageData != nil {
		webhookData.Files = []*discordgo.File{
			{
				Name:   "scoreboard.png",
				Reader: bytes.NewReader(imageData),
			},
		}
	}

	msg, err := s.InteractionResponseEdit(i.Interaction, webhookData)
	if err != nil {
		log.Printf("Error editing interaction response: %v", err)
		return
	}

	// Add navigation reactions to the message
	if msg != nil {
		_ = s.MessageReactionAdd(i.ChannelID, msg.ID, "⬅️")
		_ = s.MessageReactionAdd(i.ChannelID, msg.ID, "➡️")
	}
}

// handleStatsBalance displays individual user statistics
func (f *Feature) handleStatsBalance(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

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

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate user metrics service with repositories from UnitOfWork
	metricsService := services.NewUserMetricsService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.BetRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
	)

	// Get user stats
	stats, err := metricsService.GetUserStats(ctx, targetID)
	if err != nil {
		log.Printf("Error getting user stats for %d: %v", targetID, err)
		common.RespondWithError(s, i, "Unable to retrieve user statistics. Please try again.")
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
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
