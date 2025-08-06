package housewagers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// CreateHouseWagerComponents creates the button components for a house wager
func CreateHouseWagerComponents(houseWager dto.HouseWagerPostDTO) []discordgo.MessageComponent {
	// Only show components for active wagers that haven't expired
	// Check if wager is active and voting period is still active
	log.Debugf("CreateHouseWagerComponents: wagerID=%d, state='%s'", houseWager.WagerID, houseWager.State)

	if houseWager.State != "active" {
		// Wager is not active (resolved, cancelled, pending_resolution), no components
		log.Debugf("Hiding components for wager %d because state is '%s' (not 'active')", houseWager.WagerID, houseWager.State)
		return []discordgo.MessageComponent{}
	}

	if houseWager.VotingEndsAt != nil && houseWager.VotingEndsAt.Before(time.Now()) {
		// Voting period has expired, no components
		return []discordgo.MessageComponent{}
	}

	var components []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	// Create buttons for each betting option
	for i, option := range houseWager.Options {
		emoji := getOptionEmoji(i + 1)

		button := discordgo.Button{
			Label:    fmt.Sprintf("%s (%.2fx)", option.Text, option.Multiplier),
			Style:    getBetButtonStyle(i + 1),
			CustomID: fmt.Sprintf("house_wager_bet_%d_%d", houseWager.WagerID, option.ID),
			Emoji: &discordgo.ComponentEmoji{
				Name: emoji,
			},
		}

		currentRow = append(currentRow, button)

		// Max 5 buttons per row, but for house wagers we typically have 2 options (Win/Loss)
		if len(currentRow) == 5 || i == len(houseWager.Options)-1 {
			components = append(components, discordgo.ActionsRow{
				Components: currentRow,
			})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	return components
}

// getBetButtonStyle returns appropriate button style for betting options
func getBetButtonStyle(optionNumber int) discordgo.ButtonStyle {
	switch optionNumber {
	case 1:
		return discordgo.SuccessButton // Green for Win
	case 2:
		return discordgo.DangerButton // Red for Loss
	default:
		return discordgo.SecondaryButton // Gray for additional options
	}
}

// handleHouseWagerBetButton handles when a user clicks a house wager betting button
func (f *Feature) handleHouseWagerBetButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse custom ID: house_wager_bet_<wager_id>_<option_id>
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) != 5 {
		log.Errorf("Invalid house wager bet button customID: %s", i.MessageComponentData().CustomID)
		common.RespondWithError(s, i, "Invalid button configuration")
		return
	}

	wagerID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		log.Errorf("Invalid wager ID in customID: %s", parts[3])
		common.RespondWithError(s, i, "Invalid wager ID")
		return
	}

	optionID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		log.Errorf("Invalid option ID in customID: %s", parts[4])
		common.RespondWithError(s, i, "Invalid option ID")
		return
	}

	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Invalid guild ID: %s", i.GuildID)
		common.RespondWithError(s, i, "Invalid guild ID")
		return
	}

	// Create UoW to get wager details
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		common.RespondWithError(s, i, "Database error occurred")
		return
	}
	defer uow.Rollback()

	// Get wager details
	wagerDetail, err := uow.GroupWagerRepository().GetDetailByID(context.Background(), wagerID)
	if err != nil {
		log.Errorf("Failed to get wager detail: %v", err)
		common.RespondWithError(s, i, "Wager not found")
		return
	}

	if wagerDetail == nil || wagerDetail.Wager == nil {
		common.RespondWithError(s, i, "Wager not found")
		return
	}

	// Find the selected option
	var selectedOption *entities.GroupWagerOption
	for _, opt := range wagerDetail.Options {
		if opt.ID == optionID {
			selectedOption = opt
			break
		}
	}

	if selectedOption == nil {
		common.RespondWithError(s, i, "Invalid betting option")
		return
	}

	// Create betting modal
	modal := f.createHouseWagerBetModal(wagerID, optionID, selectedOption.OptionText, selectedOption.OddsMultiplier, wagerDetail.Wager.Condition)

	// Respond with modal
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   modal.CustomID,
			Title:      modal.Title,
			Components: modal.Components,
		},
	}); err != nil {
		log.Errorf("Failed to respond with modal: %v", err)
		common.RespondWithError(s, i, "Failed to open betting form")
	}
}

// createHouseWagerBetModal creates a modal for betting on a house wager option
func (f *Feature) createHouseWagerBetModal(wagerID, optionID int64, optionText string, multiplier float64, condition string) *discordgo.InteractionResponseData {
	// Extract the first line of the condition for context (summoner name and game type)
	var wagerContext string
	if idx := strings.Index(condition, "\n"); idx > 0 {
		wagerContext = condition[:idx]
	} else {
		wagerContext = condition
	}

	// Create a more informative title
	title := fmt.Sprintf("Bet: %s", optionText)
	if wagerContext != "" {
		title = fmt.Sprintf("%s - %s", wagerContext, optionText)
		// Truncate if too long (Discord limit is 45 chars for modal title)
		if len(title) > 45 {
			title = fmt.Sprintf("Bet: %s", optionText)
		}
	}

	return &discordgo.InteractionResponseData{
		CustomID: fmt.Sprintf("house_wager_bet_modal_%d_%d", wagerID, optionID),
		Title:    title,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "amount",
						Label:       fmt.Sprintf("Bet Amount (%.2fx payout)", multiplier),
						Style:       discordgo.TextInputShort,
						Placeholder: "Enter amount in bits (e.g., 1000)",
						Required:    true,
						MinLength:   1,
						MaxLength:   20,
					},
				},
			},
		},
	}
}

// handleHouseWagerBetModal handles the submission of a house wager bet modal
func (f *Feature) handleHouseWagerBetModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse custom ID: house_wager_bet_modal_<wager_id>_<option_id>
	parts := strings.Split(i.ModalSubmitData().CustomID, "_")
	if len(parts) != 6 {
		log.Errorf("Invalid house wager bet modal customID: %s", i.ModalSubmitData().CustomID)
		common.RespondWithError(s, i, "Invalid modal configuration")
		return
	}

	wagerID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		log.Errorf("Invalid wager ID in modal customID: %s", parts[4])
		common.RespondWithError(s, i, "Invalid wager ID")
		return
	}

	optionID, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		log.Errorf("Invalid option ID in modal customID: %s", parts[5])
		common.RespondWithError(s, i, "Invalid option ID")
		return
	}

	// Get bet amount from modal
	var betAmount int64
	for _, component := range i.ModalSubmitData().Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if textInput, ok := comp.(*discordgo.TextInput); ok && textInput.CustomID == "amount" {
					betAmount, err = strconv.ParseInt(textInput.Value, 10, 64)
					if err != nil {
						common.RespondWithError(s, i, "Invalid bet amount. Please enter a valid number.")
						return
					}
				}
			}
		}
	}

	if betAmount <= 0 {
		common.RespondWithError(s, i, "Bet amount must be greater than 0")
		return
	}

	// Get user and guild info
	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID: %s", i.Member.User.ID)
		common.RespondWithError(s, i, "Invalid user ID")
		return
	}

	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Invalid guild ID: %s", i.GuildID)
		common.RespondWithError(s, i, "Invalid guild ID")
		return
	}

	// Create UoW for this guild
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		common.RespondWithError(s, i, "Database error occurred")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Get wager detail to find the option multiplier
	wagerDetail, err := uow.GroupWagerRepository().GetDetailByID(context.Background(), wagerID)
	if err != nil {
		uow.Rollback()
		log.Errorf("Failed to get wager detail: %v", err)
		common.RespondWithError(s, i, "Wager not found")
		return
	}

	// Find selected option
	var selectedOption *entities.GroupWagerOption
	for _, opt := range wagerDetail.Options {
		if opt.ID == optionID {
			selectedOption = opt
			break
		}
	}

	if selectedOption == nil {
		uow.Rollback()
		common.RespondWithError(s, i, "Invalid betting option")
		return
	}

	// Create group wager service
	groupWagerService := services.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Place the bet
	_, err = groupWagerService.PlaceBet(context.Background(), wagerID, userID, optionID, betAmount)
	if err != nil {
		uow.Rollback()
		log.Errorf("Failed to place house wager bet: %v", err)
		common.RespondWithError(s, i, fmt.Sprintf("Failed to place bet: %v", err))
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Failed to commit house wager bet transaction: %v", err)
		common.RespondWithError(s, i, "Failed to save bet")
		return
	}

	// Calculate potential payout
	potentialPayout := float64(betAmount) * selectedOption.OddsMultiplier

	// Create Discord message link to original wager
	wagerLink := common.FormatDiscordMessageLink(guildID, wagerDetail.Wager.ChannelID, wagerDetail.Wager.MessageID)

	// Respond with success
	embed := &discordgo.MessageEmbed{
		Title: "✅ Bet Placed Successfully!",
		Description: fmt.Sprintf("You bet **%s bits** on **%s**\n[View original wager](%s)",
			common.FormatBalance(betAmount), selectedOption.OptionText, wagerLink),
		Color:  common.ColorPrimary,
		Fields: []*discordgo.MessageEmbedField{},
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral, // Only show to the user who placed the bet
		},
	}); err != nil {
		log.Errorf("Failed to respond to house wager bet: %v", err)
	}

	// Update the original house wager message to show new participant
	f.updateHouseWagerMessage(s, i.Message, wagerID, guildID)

	log.WithFields(log.Fields{
		"userID":   userID,
		"wagerID":  wagerID,
		"optionID": optionID,
		"amount":   betAmount,
		"payout":   potentialPayout,
	}).Info("House wager bet placed successfully")
}

// updateHouseWagerMessage updates a house wager message with current participant state
func (f *Feature) updateHouseWagerMessage(s *discordgo.Session, msg *discordgo.Message, wagerID int64, guildID int64) {
	ctx := context.Background()

	// Create unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Printf("Error beginning transaction for house wager update: %v", err)
		return
	}
	defer uow.Rollback()

	// Get updated wager details
	detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, wagerID)
	if err != nil {
		log.Printf("Error getting house wager detail for update: %v", err)
		return
	}
	if detail == nil {
		log.Printf("House wager not found for update: %d", wagerID)
		return
	}

	// Parse the condition to extract title and description
	// Split on first newline - everything before is title, everything after is description
	parts := strings.SplitN(detail.Wager.Condition, "\n", 2)
	title := parts[0]
	description := ""
	if len(parts) > 1 {
		description = parts[1]
	}

	// Convert to HouseWagerPostDTO for embed creation
	houseWagerDTO := dto.HouseWagerPostDTO{
		GuildID:      detail.Wager.GuildID,
		ChannelID:    detail.Wager.ChannelID,
		WagerID:      detail.Wager.ID,
		Title:        title,       // Title from first line
		Description:  description, // Description from remaining lines
		State:        string(detail.Wager.State),
		Options:      make([]dto.WagerOptionDTO, len(detail.Options)),
		VotingEndsAt: detail.Wager.VotingEndsAt,
		Participants: make([]dto.ParticipantDTO, len(detail.Participants)),
		TotalPot:     detail.Wager.TotalPot,
	}

	// Convert options
	for i, opt := range detail.Options {
		houseWagerDTO.Options[i] = dto.WagerOptionDTO{
			ID:          opt.ID,
			Text:        opt.OptionText,
			Order:       opt.OptionOrder,
			Multiplier:  opt.OddsMultiplier,
			TotalAmount: opt.TotalAmount,
		}
	}

	// Convert participants
	for i, participant := range detail.Participants {
		houseWagerDTO.Participants[i] = dto.ParticipantDTO{
			DiscordID: participant.DiscordID,
			OptionID:  participant.OptionID,
			Amount:    participant.Amount,
		}
	}

	// Create updated embed and components
	embed := CreateHouseWagerEmbed(houseWagerDTO)
	components := CreateHouseWagerComponents(houseWagerDTO)

	// Update the Discord message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    msg.ChannelID,
		ID:         msg.ID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		log.Errorf("Failed to update house wager message (channel: %s, message: %s): %v",
			msg.ChannelID, msg.ID, err)
	} else {
		log.Debugf("Successfully updated house wager message for wager %d", wagerID)
	}
}
