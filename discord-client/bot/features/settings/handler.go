package settings

import (
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/services"

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
	guildSettingsService := services.NewGuildSettingsService(
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

// handlePrimaryChannel handles the /settings primary-channel command
func (f *Feature) handlePrimaryChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Get the channel option (if provided)
	options := i.ApplicationCommandData().Options[0].Options
	var channelID *int64

	if len(options) > 0 && options[0].Name == "channel" {
		// User provided a channel
		channelIDStr := options[0].ChannelValue(s).ID
		if channelIDStr != "" {
			channelIDInt, err := strconv.ParseInt(channelIDStr, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse channel ID: %v", err)
				common.RespondWithError(s, i, "❌ Invalid channel selected")
				return
			}
			channelID = &channelIDInt
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
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the primary channel setting
	if err := guildSettingsService.UpdatePrimaryChannel(ctx, guildID, channelID); err != nil {
		log.Errorf("Failed to update primary channel: %v", err)
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
	if channelID != nil {
		message = fmt.Sprintf("✅ Primary channel updated to <#%d>", *channelID)
	} else {
		message = "✅ Primary channel feature disabled"
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

// handleLolChannel handles the /settings lol-channel command
func (f *Feature) handleLolChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Get the channel option (if provided)
	options := i.ApplicationCommandData().Options[0].Options
	var channelID *int64

	if len(options) > 0 && options[0].Name == "channel" {
		// User provided a channel
		channelIDStr := options[0].ChannelValue(s).ID
		if channelIDStr != "" {
			channelIDInt, err := strconv.ParseInt(channelIDStr, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse channel ID: %v", err)
				common.RespondWithError(s, i, "❌ Invalid channel selected")
				return
			}
			channelID = &channelIDInt
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
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the LOL channel setting
	if err := guildSettingsService.UpdateLolChannel(ctx, guildID, channelID); err != nil {
		log.Errorf("Failed to update LOL channel: %v", err)
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
	if channelID != nil {
		message = fmt.Sprintf("✅ LOL channel updated to <#%d>", *channelID)
	} else {
		message = "✅ LOL channel feature disabled"
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

// handleTftChannel handles the /settings tft-channel command
func (f *Feature) handleTftChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Get the channel option (if provided)
	options := i.ApplicationCommandData().Options[0].Options
	var channelID *int64

	if len(options) > 0 && options[0].Name == "channel" {
		// User provided a channel
		channelIDStr := options[0].ChannelValue(s).ID
		if channelIDStr != "" {
			channelIDInt, err := strconv.ParseInt(channelIDStr, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse channel ID: %v", err)
				common.RespondWithError(s, i, "❌ Invalid channel selected")
				return
			}
			channelID = &channelIDInt
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
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the TFT channel setting
	if err := guildSettingsService.UpdateTftChannel(ctx, guildID, channelID); err != nil {
		log.Errorf("Failed to update TFT channel: %v", err)
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
	if channelID != nil {
		message = fmt.Sprintf("✅ TFT channel updated to <#%d>", *channelID)
	} else {
		message = "✅ TFT channel feature disabled"
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

// handleWordleChannel handles the /settings wordle-channel command
func (f *Feature) handleWordleChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Get the channel option (if provided)
	options := i.ApplicationCommandData().Options[0].Options
	var channelID *int64

	if len(options) > 0 && options[0].Name == "channel" {
		// User provided a channel
		channelIDStr := options[0].ChannelValue(s).ID
		if channelIDStr != "" {
			channelIDInt, err := strconv.ParseInt(channelIDStr, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse channel ID: %v", err)
				common.RespondWithError(s, i, "❌ Invalid channel selected")
				return
			}
			channelID = &channelIDInt
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
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the Wordle channel setting
	if err := guildSettingsService.UpdateWordleChannel(ctx, guildID, channelID); err != nil {
		log.Errorf("Failed to update Wordle channel: %v", err)
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
	if channelID != nil {
		message = fmt.Sprintf("✅ Wordle channel updated to <#%d>", *channelID)
	} else {
		message = "✅ Wordle channel feature disabled"
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

// handleLottoChannel handles the /settings lotto-channel command
func (f *Feature) handleLottoChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user has admin permissions
	if !common.IsUserAdmin(s, i.GuildID, i.Member.User.ID) {
		common.RespondWithError(s, i, "You need administrator permissions to use this command")
		return
	}

	// Parse guild ID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID: %v", err)
		common.RespondWithError(s, i, "Failed to process command")
		return
	}

	// Get the channel option (if provided)
	options := i.ApplicationCommandData().Options[0].Options
	var channelID *int64

	if len(options) > 0 && options[0].Name == "channel" {
		// User provided a channel
		channelIDStr := options[0].ChannelValue(s).ID
		if channelIDStr != "" {
			channelIDInt, err := strconv.ParseInt(channelIDStr, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse channel ID: %v", err)
				common.RespondWithError(s, i, "Invalid channel selected")
				return
			}
			channelID = &channelIDInt
		}
	}

	ctx := context.Background()

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}
	defer uow.Rollback()

	// Instantiate guild settings service with repositories from UnitOfWork
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the lotto channel setting
	if err := guildSettingsService.UpdateLottoChannel(ctx, guildID, channelID); err != nil {
		log.Errorf("Failed to update lotto channel: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}

	// Respond with success
	var message string
	if channelID != nil {
		message = fmt.Sprintf("Lottery channel updated to <#%d>", *channelID)

		// Post the lottery message to the new channel
		if err := f.postLotteryMessage(ctx, guildID, *channelID); err != nil {
			log.Errorf("Failed to post lottery message: %v", err)
			// Append warning to response but don't fail the settings update
			message += "\n\n⚠️ Failed to post lottery message to channel. The lottery will be posted when a new draw starts."
		} else {
			message += "\n\n✅ Lottery message posted to channel."
		}
	} else {
		message = "Lottery channel feature disabled"
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

// postLotteryMessage posts the lottery message to the configured channel
func (f *Feature) postLotteryMessage(ctx context.Context, guildID, channelID int64) error {
	// Create UoW for lottery operations
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get or create the current lottery draw
	lotteryService := services.NewLotteryService(
		uow.LotteryDrawRepository(),
		uow.LotteryTicketRepository(),
		uow.LotteryWinnerRepository(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
		uow.GuildSettingsRepository(),
		uow.EventBus(),
	)

	// Get draw info (this will create a draw if none exists)
	drawInfo, err := lotteryService.GetDrawInfo(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get draw info: %w", err)
	}

	// Post the lottery message via lotteryPoster
	messageID, err := f.lotteryPoster.PostNewLotteryDraw(ctx, drawInfo, channelID)
	if err != nil {
		return fmt.Errorf("failed to post lottery message: %w", err)
	}

	// Save message ID to the draw
	if err := lotteryService.SetDrawMessage(ctx, drawInfo.Draw.ID, channelID, messageID); err != nil {
		log.Errorf("Failed to save draw message ID: %v", err)
		// Don't fail the whole operation for this
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// handleLottoTicketCost handles the /settings lotto-cost command
func (f *Feature) handleLottoTicketCost(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user has admin permissions
	if !common.IsUserAdmin(s, i.GuildID, i.Member.User.ID) {
		common.RespondWithError(s, i, "You need administrator permissions to use this command")
		return
	}

	// Parse guild ID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID: %v", err)
		common.RespondWithError(s, i, "Failed to process command")
		return
	}

	// Get the cost option
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please provide a ticket cost")
		return
	}

	cost := options[0].IntValue()
	if cost <= 0 {
		common.RespondWithError(s, i, "Ticket cost must be a positive number")
		return
	}

	ctx := context.Background()

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}
	defer uow.Rollback()

	// Instantiate guild settings service with repositories from UnitOfWork
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the lotto ticket cost setting
	if err := guildSettingsService.UpdateLottoTicketCost(ctx, guildID, &cost); err != nil {
		log.Errorf("Failed to update lotto ticket cost: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}

	// Respond with success
	message := fmt.Sprintf("Lottery ticket cost updated to %s", common.FormatBalance(cost))

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

// handleLottoDifficulty handles the /settings lotto-difficulty command
func (f *Feature) handleLottoDifficulty(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user has admin permissions
	if !common.IsUserAdmin(s, i.GuildID, i.Member.User.ID) {
		common.RespondWithError(s, i, "You need administrator permissions to use this command")
		return
	}

	// Parse guild ID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID: %v", err)
		common.RespondWithError(s, i, "Failed to process command")
		return
	}

	// Get the difficulty option
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please provide a difficulty value")
		return
	}

	difficulty := options[0].IntValue()

	ctx := context.Background()

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}
	defer uow.Rollback()

	// Instantiate guild settings service with repositories from UnitOfWork
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Update the lotto difficulty setting
	if err := guildSettingsService.UpdateLottoDifficulty(ctx, guildID, &difficulty); err != nil {
		log.Errorf("Failed to update lotto difficulty: %v", err)
		common.RespondWithError(s, i, fmt.Sprintf("Failed to update settings: %v", err))
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Failed to update settings")
		return
	}

	// Calculate possible numbers for display
	totalNumbers := int64(1) << difficulty

	// Respond with success
	message := fmt.Sprintf("Lottery difficulty updated to %d bits (%d possible numbers)", difficulty, totalNumbers)

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
