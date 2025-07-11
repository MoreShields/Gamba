package groupwagers

import (
	"context"
	"fmt"
	"gambler/bot/common"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleGroupWagerCreate handles the /groupwager create subcommand
func (f *Feature) handleGroupWagerCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Respond with a modal to collect wager details
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "group_wager_create_modal",
			Title:    "Create Group Wager",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "condition",
							Label:       "Wager Condition",
							Style:       discordgo.TextInputShort,
							Placeholder: "Who will win Worlds?",
							Required:    true,
							MaxLength:   200,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "options",
							Label:       "Options (one per line, 2-10 options)",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "T1\nBLG\nGAM\nFLY\nFURIA\nKOI",
							Required:    true,
							MaxLength:   1000,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "voting_period",
							Label:       "Voting Period",
							Style:       discordgo.TextInputShort,
							Placeholder: "24, 1:30, :45",
							Required:    false,
							MaxLength:   10,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error showing group wager modal: %v", err)
	}
}

// handleGroupWagerCreateModal handles the modal submission for creating a group wager
func (f *Feature) handleGroupWagerCreateModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.ModalSubmitData()

	// Extract condition, options, and voting period from modal
	var condition string
	var optionsText string
	var votingPeriodText string

	for _, comp := range data.Components {
		row := comp.(*discordgo.ActionsRow)
		for _, innerComp := range row.Components {
			textInput := innerComp.(*discordgo.TextInput)
			switch textInput.CustomID {
			case "condition":
				condition = strings.TrimSpace(textInput.Value)
			case "options":
				optionsText = strings.TrimSpace(textInput.Value)
			case "voting_period":
				votingPeriodText = strings.TrimSpace(textInput.Value)
			}
		}
	}

	// Parse options (one per line)
	optionLines := strings.Split(optionsText, "\n")
	var options []string
	for _, line := range optionLines {
		line = strings.TrimSpace(line)
		if line != "" {
			options = append(options, line)
		}
	}

	// Validate options count
	if len(options) < 2 {
		common.RespondWithError(s, i, "Please provide at least 2 options.")
		return
	}
	if len(options) > 10 {
		common.RespondWithError(s, i, "Maximum 10 options allowed.")
		return
	}

	// Parse and validate voting period
	votingPeriodMinutes := 1440 // Default value (24 hours)
	if votingPeriodText != "" {
		// Check if it's in hours:minutes format
		if strings.Contains(votingPeriodText, ":") {
			parts := strings.Split(votingPeriodText, ":")
			if len(parts) != 2 {
				common.RespondWithError(s, i, "Invalid time format. Use hours:minutes (e.g., 1:30) or just hours (e.g., 24).")
				return
			}

			// Parse hours
			hoursStr := strings.TrimSpace(parts[0])
			hours := 0
			if hoursStr != "" {
				var err error
				hours, err = strconv.Atoi(hoursStr)
				if err != nil {
					common.RespondWithError(s, i, "Invalid hours value.")
					return
				}
			}

			// Parse minutes
			minutesStr := strings.TrimSpace(parts[1])
			minutes, err := strconv.Atoi(minutesStr)
			if err != nil {
				common.RespondWithError(s, i, "Invalid minutes value.")
				return
			}

			// Validate minutes
			if minutes < 0 || minutes >= 60 {
				common.RespondWithError(s, i, "Minutes must be between 0 and 59.")
				return
			}

			// Convert to total minutes
			votingPeriodMinutes = hours*60 + minutes

			// Validate total time
			if votingPeriodMinutes < 5 {
				common.RespondWithError(s, i, "Voting period must be at least 5 minutes.")
				return
			}
		} else {
			// Parse as plain hours
			hours, err := strconv.Atoi(votingPeriodText)
			if err != nil {
				common.RespondWithError(s, i, "Voting period must be a valid number or time format (e.g., 1:30).")
				return
			}
			votingPeriodMinutes = hours * 60
		}

		// Final validation
		if votingPeriodMinutes > 10080 {
			common.RespondWithError(s, i, "Voting period must not exceed 168 hours (1 week).")
			return
		}
	}

	// Defer response while we process
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring group wager creation: %v", err)
		return
	}

	// Get user ID
	creatorID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing creator ID: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request.")
		return
	}

	// Create the group wager (message ID will be updated after posting)
	groupWagerDetail, err := f.groupWagerService.CreateGroupWager(ctx, creatorID, condition, options, votingPeriodMinutes, 0, 0)
	if err != nil {
		log.Printf("Error creating group wager: %v", err)
		common.FollowUpWithError(s, i, fmt.Sprintf("Failed to create group wager: %v", err))
		return
	}

	// Create the embed
	embed := CreateGroupWagerEmbed(groupWagerDetail)
	components := CreateGroupWagerComponents(groupWagerDetail)

	// Send the follow-up message
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("Error sending group wager message: %v", err)
		return
	}

	// Update the group wager with message and channel IDs
	messageID, err := strconv.ParseInt(msg.ID, 10, 64)
	if err != nil {
		log.Errorf("failed to parse MessageID: %s", err)
		return
	}
	channelID, err := strconv.ParseInt(msg.ChannelID, 10, 64)
	if err != nil {
		log.Errorf("failed to parse ChannelID: %s", err)
		return
	}

	// Update the group wager with the message IDs
	if err := f.groupWagerService.UpdateMessageIDs(ctx, groupWagerDetail.Wager.ID, messageID, channelID); err != nil {
		log.Errorf("failed to update group wager message IDs: %s", err)
	}

}

// handleGroupWagerResolve handles the /groupwager resolve subcommand
func (f *Feature) handleGroupWagerResolve(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	options := i.ApplicationCommandData().Options[0].Options

	var groupWagerID int64
	var winningOptionID int64

	for _, opt := range options {
		switch opt.Name {
		case "id":
			groupWagerID = opt.IntValue()
		case "winning_option":
			winningOptionID = opt.IntValue()
		}
	}

	// Get resolver ID
	resolverID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing resolver ID: %v", err)
		common.RespondWithError(s, i, "Unable to process request.")
		return
	}

	// Check if user is a resolver
	if !f.groupWagerService.IsResolver(resolverID) {
		common.RespondWithError(s, i, "You are not authorized to resolve group wagers.")
		return
	}

	// Defer response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring resolve response: %v", err)
		return
	}

	// Resolve the wager
	result, err := f.groupWagerService.ResolveGroupWager(ctx, groupWagerID, resolverID, winningOptionID)
	if err != nil {
		log.Printf("Error resolving group wager: %v", err)
		common.FollowUpWithError(s, i, fmt.Sprintf("Failed to resolve wager: %v", err))
		return
	}

	// Create success message
	var winnerList []string
	for _, winner := range result.Winners {
		payout := result.PayoutDetails[winner.DiscordID]
		winnerList = append(winnerList, fmt.Sprintf("<@%d> won %s bits", winner.DiscordID, common.FormatBalance(payout)))
	}

	message := fmt.Sprintf(
		"**Group Wager Resolved!**\n\nCondition: %s\nWinning Option: %s\nTotal Pot: %s bits\n\n**Winners:**\n%s",
		result.GroupWager.Condition,
		result.WinningOption.OptionText,
		common.FormatBalance(result.TotalPot),
		strings.Join(winnerList, "\n"),
	)

	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: message,
	})
	if err != nil {
		log.Printf("Error sending resolve message: %v", err)
	}

	// Update the original wager message to show it's resolved
	if result.GroupWager.MessageID != 0 && result.GroupWager.ChannelID != 0 {
		// Get updated wager details
		updatedDetail, err := f.groupWagerService.GetGroupWagerDetail(ctx, groupWagerID)
		if err != nil {
			log.Printf("Error getting updated group wager detail: %v", err)
			return
		}

		// Create updated embed and components
		embed := CreateGroupWagerEmbed(updatedDetail)
		components := CreateGroupWagerComponents(updatedDetail) // Will be empty since wager is resolved

		// Update the original message
		_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    strconv.FormatInt(result.GroupWager.ChannelID, 10),
			ID:         strconv.FormatInt(result.GroupWager.MessageID, 10),
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err != nil {
			log.Printf("Error updating resolved group wager message: %v", err)
		}
	}
}

// handleGroupWagerButtonInteraction handles button clicks on group wager messages
func (f *Feature) handleGroupWagerButtonInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Parse custom ID: group_wager_option_<wager_id>_<option_id>
	parts := strings.Split(customID, "_")
	if len(parts) != 5 || parts[0] != "group" || parts[1] != "wager" || parts[2] != "option" {
		return
	}

	groupWagerID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		log.Errorf("Error parsing group wager ID from %s: %v", parts[3], err)
		return
	}

	optionID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		log.Errorf("Error parsing option ID from %s: %v", parts[4], err)
		return
	}

	// Show modal for bet amount
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("group_wager_bet_%d_%d", groupWagerID, optionID),
			Title:    "Place Your Bet",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "amount",
							Label:       "Bet Amount (in bits)",
							Style:       discordgo.TextInputShort,
							Placeholder: "1000",
							Required:    true,
							MaxLength:   10,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Errorf("Error showing bet modal: %v", err)
	}
}

// handleGroupWagerBetModal handles the bet amount modal submission
func (f *Feature) handleGroupWagerBetModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.ModalSubmitData()

	// Parse custom ID: group_wager_bet_<wager_id>_<option_id>
	parts := strings.Split(data.CustomID, "_")
	if len(parts) != 5 {
		return
	}

	groupWagerID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return
	}

	optionID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return
	}

	// Get amount from modal
	var amountStr string
	for _, comp := range data.Components {
		row := comp.(*discordgo.ActionsRow)
		for _, innerComp := range row.Components {
			textInput := innerComp.(*discordgo.TextInput)
			if textInput.CustomID == "amount" {
				amountStr = strings.TrimSpace(textInput.Value)
			}
		}
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		common.RespondWithError(s, i, "Please enter a valid positive amount.")
		return
	}

	// Get user ID
	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Unable to process request.")
		return
	}
	// Ensure user is present in the database.
	_, err = f.userService.GetOrCreateUser(ctx, userID, i.Member.User.Username)
	if err != nil {
		common.RespondWithError(s, i, "Unable to get user from DB")
		return
	}

	// Place the bet
	_, err = f.groupWagerService.PlaceBet(ctx, groupWagerID, userID, optionID, amount)
	if err != nil {
		log.Errorf("Error placing bet: %v", err)
		common.RespondWithError(s, i, fmt.Sprintf("Failed to place bet: %v", err))

		// Check if the error is due to voting period expiration
		if strings.Contains(err.Error(), "voting period has ended") {
			// Update the message to reflect the expired state
			f.updateGroupWagerMessage(s, i.Message, groupWagerID)
			return
		}
	}

	// Respond with success
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully placed a bet of %s bits!", common.FormatBalance(amount)),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Errorf("Error responding to bet: %v", err)
	}

	// Update the original message
	f.updateGroupWagerMessage(s, i.Message, groupWagerID)
}

// updateGroupWagerMessage updates a group wager message with current state
func (f *Feature) updateGroupWagerMessage(s *discordgo.Session, msg *discordgo.Message, groupWagerID int64) {
	ctx := context.Background()

	// Get updated wager details
	detail, err := f.groupWagerService.GetGroupWagerDetail(ctx, groupWagerID)
	if err != nil {
		log.Printf("Error getting group wager detail: %v", err)
		return
	}

	// Update the message
	embed := CreateGroupWagerEmbed(detail)
	components := CreateGroupWagerComponents(detail)

	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    msg.ChannelID,
		ID:         msg.ID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Errorf("Error updating group wager message: %v", err)
	}
}
