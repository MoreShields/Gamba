package transfer

import (
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

func (f *Feature) handleDonate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Extract command options
	options := i.ApplicationCommandData().Options
	if len(options) != 2 {
		common.RespondWithError(s, i, "Invalid command options. Please provide both amount and user.")
		return
	}

	// Extract amount and recipient
	var amount int64
	var recipientUser *discordgo.User
	for _, opt := range options {
		switch opt.Name {
		case "amount":
			amount = opt.IntValue()
		case "user":
			recipientUser = opt.UserValue(s)
		}
	}

	if amount <= 0 {
		common.RespondWithError(s, i, "Amount must be positive.")
		return
	}

	if recipientUser == nil {
		common.RespondWithError(s, i, "Invalid recipient user.")
		return
	}

	// Convert Discord string IDs to int64
	fromDiscordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing sender Discord ID %s: %v", i.Member.User.ID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	toDiscordID, err := strconv.ParseInt(recipientUser.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing recipient Discord ID %s: %v", recipientUser.ID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Prevent self-donation
	if fromDiscordID == toDiscordID {
		common.RespondWithError(s, i, "You cannot donate to yourself.")
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

	// Instantiate user service with repositories from UnitOfWork
	userService := service.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Ensure both users in the DB.
	_, err = userService.GetOrCreateUser(ctx, fromDiscordID, i.Member.User.Username)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to create user: %v", err))
		return
	}
	_, err = userService.GetOrCreateUser(ctx, toDiscordID, recipientUser.Username)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to create user: %v", err))
		return
	}

	// Process the transfer
	err = userService.TransferBetweenUsers(ctx, fromDiscordID, toDiscordID, amount, i.Member.User.Username, recipientUser.Username)
	if err != nil {
		log.Errorf("Error processing donation from %d to %d: %v", fromDiscordID, toDiscordID, err)
		common.RespondWithError(s, i, fmt.Sprintf("Transfer failed: %v", err))
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Send success response
	message := common.FormatTransferResult(amount, recipientUser.ID)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		log.Errorf("Error responding to donate command: %v", err)
	}
}
