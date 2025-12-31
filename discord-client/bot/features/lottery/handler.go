package lottery

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleBuyButton handles the buy tickets button click
func (f *Feature) handleBuyButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	customID := i.MessageComponentData().CustomID

	// Parse draw ID from custom ID: lotto_buy_<draw_id>
	parts := strings.Split(customID, "_")
	if len(parts) < 3 {
		common.RespondWithError(s, i, "Invalid button")
		return
	}
	drawID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid draw ID")
		return
	}

	// Parse Discord IDs
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid guild ID")
		return
	}

	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid user ID")
		return
	}

	username := i.Member.User.Username

	// Create UoW
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		common.RespondWithError(s, i, "Failed to process request")
		return
	}
	defer uow.Rollback()

	// Get user balance
	userService := services.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	user, err := userService.GetOrCreateUser(ctx, discordID, username)
	if err != nil {
		log.Errorf("Failed to get user: %v", err)
		common.RespondWithError(s, i, "Failed to get user")
		return
	}

	// Get draw to get ticket cost
	draw, err := uow.LotteryDrawRepository().GetByID(ctx, drawID)
	if err != nil || draw == nil {
		log.Errorf("Failed to get draw: %v", err)
		common.RespondWithError(s, i, "Failed to get lottery draw")
		return
	}

	if !draw.CanPurchaseTickets() {
		common.RespondWithError(s, i, "This lottery draw is no longer accepting tickets")
		return
	}

	// Show modal with ticket cost info
	modal := CreateBuyTicketsModal(drawID, draw.TicketCost, user.Balance)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

// handleBuyModalSubmit handles the buy tickets modal submission
func (f *Feature) handleBuyModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.ModalSubmitData()

	// Defer response to allow time for processing large purchases
	if err := common.DeferResponse(s, i, true); err != nil {
		log.Errorf("Failed to defer response: %v", err)
		return
	}

	// Validate custom ID format: lotto_buy_modal_<draw_id>
	parts := strings.Split(data.CustomID, "_")
	if len(parts) < 4 {
		common.UpdateMessageWithError(s, i, "Invalid modal")
		return
	}
	if _, err := strconv.ParseInt(parts[3], 10, 64); err != nil {
		common.UpdateMessageWithError(s, i, "Invalid draw ID")
		return
	}

	// Parse quantity from modal
	var quantityStr string
	for _, comp := range data.Components {
		row := comp.(*discordgo.ActionsRow)
		for _, innerComp := range row.Components {
			textInput := innerComp.(*discordgo.TextInput)
			if textInput.CustomID == "quantity" {
				quantityStr = strings.TrimSpace(textInput.Value)
			}
		}
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		common.UpdateMessageWithError(s, i, "Please enter a valid positive number")
		return
	}

	// Parse Discord IDs
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		common.UpdateMessageWithError(s, i, "Invalid guild ID")
		return
	}

	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.UpdateMessageWithError(s, i, "Invalid user ID")
		return
	}

	username := i.Member.User.Username

	// Create UoW
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		common.UpdateMessageWithError(s, i, "Failed to process purchase")
		return
	}
	defer uow.Rollback()

	// Ensure user exists
	userService := services.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	_, err = userService.GetOrCreateUser(ctx, discordID, username)
	if err != nil {
		log.Errorf("Failed to get/create user: %v", err)
		common.UpdateMessageWithError(s, i, "Failed to process user")
		return
	}

	// Purchase tickets via lottery service
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

	result, err := lotteryService.PurchaseTickets(ctx, discordID, guildID, quantity)
	if err != nil {
		log.Errorf("Failed to purchase tickets: %v", err)
		common.UpdateMessageWithError(s, i, fmt.Sprintf("Failed to purchase tickets: %v", err))
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Failed to commit transaction: %v", err)
		common.UpdateMessageWithError(s, i, "Failed to complete purchase")
		return
	}

	// Send ephemeral confirmation by editing the deferred response
	embed := CreatePurchaseConfirmationEmbed(result)
	if err := common.UpdateMessage(s, i, embed, nil); err != nil {
		log.Errorf("Failed to send purchase confirmation: %v", err)
	}

	// Update the lottery message with new participant info
	f.updateLotteryMessage(ctx, s, guildID)
}

// updateLotteryMessage updates the lottery embed with current info
func (f *Feature) updateLotteryMessage(ctx context.Context, s *discordgo.Session, guildID int64) {
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Failed to begin transaction for message update: %v", err)
		return
	}
	defer uow.Rollback()

	// Get draw info
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

	drawInfo, err := lotteryService.GetDrawInfo(ctx, guildID)
	if err != nil {
		log.Errorf("Failed to get draw info for update: %v", err)
		return
	}

	// Only update if draw has a message
	if !drawInfo.Draw.HasMessage() {
		return
	}

	embed := CreateLotteryEmbed(drawInfo)
	components := CreateLotteryComponents(drawInfo.Draw)

	channelIDStr := fmt.Sprintf("%d", *drawInfo.Draw.ChannelID)
	messageIDStr := fmt.Sprintf("%d", *drawInfo.Draw.MessageID)

	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelIDStr,
		ID:         messageIDStr,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Errorf("Failed to update lottery message: %v", err)
	}
}
