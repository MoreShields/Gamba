package settings

import (
	"context"
	"fmt"
	"strconv"

	"gambler/bot/common"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleHighRollerRole handles the /settings high-roller-role command
func (f *Feature) handleHighRollerRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user has admin permissions
	if !common.IsUserAdmin(s, i.GuildID, i.Member.User.ID) {
		common.RespondWithError(s, i, "❌ You need administrator permissions to use this command")
		return
	}

	// Parse guild ID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID: %v", err)
		common.RespondWithError(s, i, "❌ Failed to process command")
		return
	}

	// Get the role option (if provided)
	options := i.ApplicationCommandData().Options[0].Options
	var roleID *int64

	if len(options) > 0 && options[0].Name == "role" {
		// User provided a role
		roleIDStr := options[0].RoleValue(s, "").ID
		if roleIDStr != "" {
			roleIDInt, err := strconv.ParseInt(roleIDStr, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse role ID: %v", err)
				common.RespondWithError(s, i, "❌ Invalid role selected")
				return
			}
			roleID = &roleIDInt
		}
	}

	ctx := context.Background()

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "❌ Failed to update settings")
		return
	}
	defer uow.Rollback()

	// Instantiate guild settings service with repositories from UnitOfWork
	guildSettingsService := service.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the high roller role setting
	if err := guildSettingsService.UpdateHighRollerRole(ctx, guildID, roleID); err != nil {
		log.Errorf("Failed to update high roller role: %v", err)
		common.RespondWithError(s, i, "❌ Failed to update settings")
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "❌ Failed to update settings")
		return
	}

	// Respond with success
	var message string
	if roleID != nil {
		message = fmt.Sprintf("✅ High roller role updated to <@&%d>", *roleID)
	} else {
		message = "✅ High roller role feature disabled"
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		log.Errorf("Failed to respond to interaction: %v", err)
	}
}