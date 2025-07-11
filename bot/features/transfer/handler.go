package transfer

import (
	"context"
	"fmt"
	"strconv"

	"gambler/bot/common"

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

	// Prevent self-donation
	if fromDiscordID == toDiscordID {
		common.RespondWithError(s, i, "You cannot donate to yourself.")
		return
	}

	// Ensure both users in the DB.
	_, err = f.userService.GetOrCreateUser(context.Background(), fromDiscordID, recipientUser.Username)
	if err != nil {
		common.UpdateMessageWithError(s, i, fmt.Sprintf("Failed to create wager: %v", err))
		return
	}
	_, err = f.userService.GetOrCreateUser(context.Background(), toDiscordID, i.Member.User.Username)
	if err != nil {
		common.UpdateMessageWithError(s, i, fmt.Sprintf("Failed to create wager: %v", err))
		return
	}

	// Process the transfer
	_, err = f.transferService.Transfer(ctx, fromDiscordID, toDiscordID, amount)
	if err != nil {
		log.Errorf("Error processing donation from %d to %d: %v", fromDiscordID, toDiscordID, err)
		common.RespondWithError(s, i, fmt.Sprintf("Transfer failed: %v", err))
		return
	}

	// Get updated sender balance
	sender, err := f.userService.GetOrCreateUser(ctx, fromDiscordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting sender after transfer: %v", err)
		// Transfer succeeded but we couldn't get updated balance - still notify success
		message := fmt.Sprintf("✅ Successfully transferred **%s bits** to **%s**.",
			common.FormatBalance(amount), recipientUser.Username)
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message,
			},
		})
		return
	}

	// Send success response
	message := common.FormatTransferResult(amount, recipientUser.Username, sender.AvailableBalance)
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
