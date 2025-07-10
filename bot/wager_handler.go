package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gambler/models"

	"github.com/bwmarrin/discordgo"
)

// handleWagerCommand handles the /wager slash command
func (b *Bot) handleWagerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get subcommand
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		b.respondWithError(s, i, "Invalid command usage")
		return
	}

	switch options[0].Name {
	case "propose":
		b.handleWagerPropose(s, i)
	case "list":
		b.handleWagerList(s, i)
	case "cancel":
		b.handleWagerCancel(s, i)
	default:
		b.respondWithError(s, i, "Unknown subcommand")
	}
}

// handleWagerPropose handles the /wager propose subcommand
func (b *Bot) handleWagerPropose(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) < 2 {
		b.respondWithError(s, i, "Please specify a user and amount")
		return
	}

	// Extract parameters
	targetUser := options[0].UserValue(s)
	amount := options[1].IntValue()

	if targetUser == nil {
		b.respondWithError(s, i, "Invalid user specified")
		return
	}

	// Validate amount
	if amount <= 0 {
		b.respondWithError(s, i, "Amount must be positive")
		return
	}

	// Convert Discord IDs to int64
	proposerID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid user ID")
		return
	}

	targetID, err := strconv.ParseInt(targetUser.ID, 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid target user ID")
		return
	}

	// Show modal for condition input
	modal := BuildWagerConditionModal(proposerID, targetID, amount)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &modal,
	})
	if err != nil {
		log.Printf("Error showing wager condition modal: %v", err)
	}
}

// handleWagerConditionModal handles the modal submission for wager condition
func (b *Bot) handleWagerConditionModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse custom ID to get proposer, target, and amount
	parts := strings.Split(i.ModalSubmitData().CustomID, "_")
	if len(parts) < 6 {
		b.respondWithError(s, i, "Invalid modal data")
		return
	}

	proposerID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid proposer ID")
		return
	}

	targetID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid target ID")
		return
	}

	amount, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid amount")
		return
	}

	// Get condition from modal
	condition := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// Defer the response while we process
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring interaction: %v", err)
		return
	}

	// Create the wager (we'll get message ID after posting)
	channelID, _ := strconv.ParseInt(i.ChannelID, 10, 64)
	wager, err := b.wagerService.ProposeWager(context.Background(), proposerID, targetID, amount, condition, 0, channelID)
	if err != nil {
		content := fmt.Sprintf("❌ Failed to create wager: %v", err)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		if err != nil {
			log.Printf("Error editing interaction response: %v", err)
		}
		return
	}

	// Get server-specific display names
	proposerName := GetDisplayNameInt64(s, i.GuildID, proposerID)
	targetName := GetDisplayNameInt64(s, i.GuildID, targetID)

	// Create embed and components
	embed := BuildWagerProposedEmbed(wager, proposerName, targetName)
	components := BuildWagerProposalComponents(wager.ID)

	// Send the message
	msg, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error editing interaction response: %v", err)
		return
	}

	// Update the wager with the message ID
	if msg != nil && msg.ID != "" {
		messageID, _ := strconv.ParseInt(msg.ID, 10, 64)
		wager.MessageID = &messageID
		// Note: In a production system, you'd want to update this in the database
		// For now, the message ID will be available in the interaction context
	}
}

// handleWagerList handles the /wager list subcommand
func (b *Bot) handleWagerList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid user ID")
		return
	}

	// Get active wagers
	wagers, err := b.wagerService.GetActiveWagersByUser(context.Background(), userID)
	if err != nil {
		b.respondWithError(s, i, fmt.Sprintf("Failed to get wagers: %v", err))
		return
	}

	// Get display name
	displayName := GetDisplayName(s, i.GuildID, i.Member.User.ID)

	// Build and send embed
	embed := BuildWagerListEmbed(wagers, userID, displayName)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to wager list: %v", err)
	}
}

// handleWagerCancel handles the /wager cancel subcommand
func (b *Bot) handleWagerCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) == 0 {
		b.respondWithError(s, i, "Please specify a wager ID")
		return
	}

	wagerID := options[0].IntValue()
	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid user ID")
		return
	}

	// Get the wager details first to find the message
	wager, err := b.wagerService.GetWagerByID(context.Background(), wagerID)
	if err != nil {
		b.respondWithError(s, i, fmt.Sprintf("Failed to get wager: %v", err))
		return
	}
	if wager == nil {
		b.respondWithError(s, i, "Wager not found")
		return
	}

	// Cancel the wager
	err = b.wagerService.CancelWager(context.Background(), wagerID, userID)
	if err != nil {
		b.respondWithError(s, i, fmt.Sprintf("Failed to cancel wager: %v", err))
		return
	}

	// Try to delete the wager message if we have the IDs
	if wager.MessageID != nil && wager.ChannelID != nil {
		messageID := strconv.FormatInt(*wager.MessageID, 10)
		channelID := strconv.FormatInt(*wager.ChannelID, 10)
		err = s.ChannelMessageDelete(channelID, messageID)
		if err != nil {
			log.Printf("Error deleting wager message: %v", err)
		}
	}

	// Respond with success
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Wager #%d has been cancelled", wagerID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to wager cancel: %v", err)
	}
}

// handleWagerInteraction handles button interactions for wagers
func (b *Bot) handleWagerInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "_")

	if len(parts) < 3 {
		b.respondWithError(s, i, "Invalid interaction data")
		return
	}

	action := parts[1]
	switch action {
	case "accept", "decline":
		b.handleWagerResponse(s, i, action == "accept")
	case "vote":
		b.handleWagerVote(s, i)
	default:
		b.respondWithError(s, i, "Unknown wager action")
	}
}

// handleWagerResponse handles accept/decline button presses
func (b *Bot) handleWagerResponse(s *discordgo.Session, i *discordgo.InteractionCreate, accept bool) {
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 3 {
		b.respondWithError(s, i, "Invalid button data")
		return
	}

	wagerID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid wager ID")
		return
	}

	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid user ID")
		return
	}

	// Defer while processing
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Error deferring interaction: %v", err)
		return
	}

	// Process the response
	wager, err := b.wagerService.RespondToWager(context.Background(), wagerID, userID, accept)
	if err != nil {
		content := fmt.Sprintf("❌ %v", err)
		_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error creating followup message: %v", err)
		}
		return
	}

	// Get server-specific display names
	proposerName := GetDisplayNameInt64(s, i.GuildID, wager.ProposerDiscordID)
	targetName := GetDisplayNameInt64(s, i.GuildID, wager.TargetDiscordID)

	// Update the message based on response
	var embed *discordgo.MessageEmbed
	var components []discordgo.MessageComponent

	if accept {
		// Show voting interface
		voteCounts := &models.VoteCount{} // Start with 0 votes
		embed = BuildWagerVotingEmbed(wager, proposerName, targetName, voteCounts)
		components = BuildWagerVotingComponents(wager, proposerName, targetName)
	} else {
		// Show declined message
		embed = BuildWagerDeclinedEmbed(wager, proposerName, targetName)
		components = DisableComponents(i.Message.Components)
	}

	// Update the message
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error editing message: %v", err)
	}
}

// handleWagerVote handles vote button presses
func (b *Bot) handleWagerVote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 4 {
		b.respondWithError(s, i, "Invalid vote button data")
		return
	}

	wagerID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid wager ID")
		return
	}

	voteForID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid vote target ID")
		return
	}

	voterID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		b.respondWithError(s, i, "Invalid voter ID")
		return
	}

	// Defer while processing
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Error deferring interaction: %v", err)
		return
	}

	// Cast the vote
	_, voteCounts, err := b.wagerService.CastVote(context.Background(), wagerID, voterID, voteForID)
	if err != nil {
		content := fmt.Sprintf("❌ %v", err)
		_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error creating followup message: %v", err)
		}
		return
	}

	// Get updated wager state
	wager, err := b.wagerService.GetWagerByID(context.Background(), wagerID)
	if err != nil {
		log.Printf("Error getting wager after vote: %v", err)
		return
	}

	// Get server-specific display names
	proposerName := GetDisplayNameInt64(s, i.GuildID, wager.ProposerDiscordID)
	targetName := GetDisplayNameInt64(s, i.GuildID, wager.TargetDiscordID)

	// Update the embed based on wager state
	var embed *discordgo.MessageEmbed
	var components []discordgo.MessageComponent

	if wager.State == models.WagerStateResolved {
		// Wager has been resolved
		winnerName := proposerName
		loserName := targetName
		if wager.WinnerDiscordID != nil && *wager.WinnerDiscordID == wager.TargetDiscordID {
			winnerName = targetName
			loserName = proposerName
		}
		embed = BuildWagerResolvedEmbed(wager, proposerName, targetName, winnerName, loserName, voteCounts)
		components = DisableComponents(i.Message.Components)
	} else {
		// Still voting
		embed = BuildWagerVotingEmbed(wager, proposerName, targetName, voteCounts)
		components = i.Message.Components // Keep existing components
	}

	// Update the message
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error editing message: %v", err)
	}

	// Send ephemeral confirmation to voter
	voteTarget := proposerName
	if voteForID == wager.TargetDiscordID {
		voteTarget = targetName
	}
	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: fmt.Sprintf("✅ Your vote for **%s** has been recorded!", voteTarget),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Error creating vote confirmation: %v", err)
	}
}
