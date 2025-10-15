package stats

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/application"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature represents the stats feature
type Feature struct {
	session      *discordgo.Session
	uowFactory   application.UnitOfWorkFactory
	guildID      string
	userResolver application.UserResolver
}

// NewFeature creates a new stats feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory, guildID string, userResolver application.UserResolver) *Feature {
	return &Feature{
		session:      session,
		uowFactory:   uowFactory,
		guildID:      guildID,
		userResolver: userResolver,
	}
}

// HandleCommand handles the /stats command and its subcommands
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please specify a subcommand: scoreboard or balance")
		return
	}

	// Route to appropriate subcommand handler
	switch options[0].Name {
	case "scoreboard":
		f.handleStatsScoreboard(s, i)
	case "balance":
		f.handleStatsBalance(s, i, options[0].Options)
	default:
		common.RespondWithError(s, i, "Unknown subcommand")
	}
}

// HandleInteraction handles button interactions for scoreboard navigation
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Handle page navigation buttons
	if len(customID) > 11 && customID[:11] == "stats_page_" {
		targetPage := customID[11:]

		// Acknowledge the interaction
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		if err != nil {
			log.Errorf("Error acknowledging interaction: %v", err)
			return
		}

		// Update the scoreboard to show the requested page
		f.updateScoreboardPage(s, i.ChannelID, i.Message.ID, i.GuildID, targetPage)
	}
}

// updateScoreboardPage fetches fresh data and updates the embed to show the requested page
func (f *Feature) updateScoreboardPage(s *discordgo.Session, channelID, messageID, guildIDStr string, page string) {
	ctx := context.Background()

	// Parse guild ID
	guildID, err := strconv.ParseInt(guildIDStr, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", guildIDStr, err)
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		return
	}
	defer uow.Rollback()

	// Instantiate user metrics service
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
		log.Errorf("Error getting scoreboard: %v", err)
		return
	}

	// Get high roller info before creating embed
	var highRollerText string
	if page == PageBits { // Only show on bits page
		highRollerService := services.NewHighRollerService(
			uow.HighRollerPurchaseRepository(),
			uow.UserRepository(),
			uow.WagerRepository(),
			uow.GroupWagerRepository(),
			uow.BalanceHistoryRepository(),
			uow.GuildSettingsRepository(),
			uow.EventBus(),
		)

		highRollerInfo, err := highRollerService.GetCurrentHighRoller(ctx, guildID)
		if err == nil && highRollerInfo.CurrentHolder != nil {
			// Get the role ID for mention
			guildSettingsService := services.NewGuildSettingsService(uow.GuildSettingsRepository())
			if roleID, err := guildSettingsService.GetHighRollerRoleID(ctx, guildID); err == nil && roleID != nil {
				highRollerText = fmt.Sprintf("\n<@&%d> - <@%d> - %s - %s",
					*roleID,
					highRollerInfo.CurrentHolder.DiscordID,
					common.FormatBalance(highRollerInfo.CurrentPrice),
					common.FormatDuration(highRollerInfo.CurrentHolderDuration))
			}
		}
	}

	// Create updated embed with the new page
	embed, imageData := BuildScoreboardEmbed(ctx, metricsService, entries, totalBits, s, guildIDStr, page, f.userResolver)
	
	// Add high roller info to the description if available
	if highRollerText != "" && embed.Description != "" {
		embed.Description += highRollerText
	}

	// Commit the transaction after building the embed
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		return
	}

	// Update the message with navigation buttons
	navButtons := BuildScoreboardNavButtons(page)
	editData := &discordgo.MessageEdit{
		Channel:    channelID,
		ID:         messageID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &navButtons,
	}

	// Add image if generated
	if imageData != nil {
		// Clear existing attachments and add new one
		editData.Attachments = &[]*discordgo.MessageAttachment{}
		editData.Files = []*discordgo.File{
			{
				Name:   "scoreboard.png",
				Reader: bytes.NewReader(imageData),
			},
		}
	}

	_, err = s.ChannelMessageEditComplex(editData)
	if err != nil {
		log.Errorf("Error updating scoreboard embed: %v", err)
	}
}
