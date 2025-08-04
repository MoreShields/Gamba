package wagers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
)

// handleWagerPropose handles the /wager propose subcommand
func (f *Feature) handleWagerPropose(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) < 2 {
		common.RespondWithError(s, i, "Please specify a user and amount")
		return
	}

	// Extract parameters
	targetUser := options[0].UserValue(s)
	amount := options[1].IntValue()

	if targetUser == nil {
		common.RespondWithError(s, i, "Invalid user specified")
		return
	}

	// Validate amount
	if amount <= 0 {
		common.RespondWithError(s, i, "Amount must be positive")
		return
	}

	// Convert Discord IDs to int64
	proposerID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid user ID")
		return
	}

	targetID, err := strconv.ParseInt(targetUser.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid target user ID")
		return
	}

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate user service with repositories from UnitOfWork
	userService := services.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Get the users to ensure they exist in the DB.
	_, err = userService.GetOrCreateUser(context.Background(), proposerID, i.Member.User.Username)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to create wager: %v", err))
		return
	}
	_, err = userService.GetOrCreateUser(context.Background(), targetID, targetUser.Username)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to create wager: %v", err))
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
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
func (f *Feature) handleWagerConditionModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse custom ID to get proposer, target, and amount
	parts := strings.Split(i.ModalSubmitData().CustomID, "_")
	if len(parts) < 6 {
		common.RespondWithError(s, i, "Invalid modal data")
		return
	}

	proposerID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid proposer ID")
		return
	}

	targetID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid target ID")
		return
	}

	amount, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid amount")
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

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.UpdateMessageWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.UpdateMessageWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate wager service with repositories from UnitOfWork
	wagerService := services.NewWagerService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.WagerVoteRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Create the wager (we'll get message ID after posting)
	channelID, _ := strconv.ParseInt(i.ChannelID, 10, 64)
	wager, err := wagerService.ProposeWager(context.Background(), proposerID, targetID, amount, condition, 0, channelID)
	if err != nil {
		common.UpdateMessageWithError(s, i, fmt.Sprintf("Failed to create wager: %v", err))
		return
	}

	// Get server-specific display names
	proposerName := common.GetDisplayNameInt64(s, i.GuildID, proposerID)
	targetName := common.GetDisplayNameInt64(s, i.GuildID, targetID)
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
		// Still commit the wager even if message sending failed
		if err := uow.Commit(); err != nil {
			log.Errorf("Error committing transaction: %v", err)
		}
		return
	}

	// Update the wager with the message ID in the same transaction
	if msg != nil && msg.ID != "" {
		messageID, _ := strconv.ParseInt(msg.ID, 10, 64)
		channelIDParsed, _ := strconv.ParseInt(msg.ChannelID, 10, 64)

		if err := wagerService.UpdateMessageIDs(context.Background(), wager.ID, messageID, channelIDParsed); err != nil {
			log.Errorf("Error updating wager message IDs: %v", err)
		}
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.UpdateMessageWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Message will be pinned when the wager is accepted
}

// handleWagerList handles the /wager list subcommand
func (f *Feature) handleWagerList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid user ID")
		return
	}

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate wager service with repositories from UnitOfWork
	wagerService := services.NewWagerService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.WagerVoteRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Get active wagers
	wagers, err := wagerService.GetActiveWagersByUser(context.Background(), userID)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to get wagers: %v", err))
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get display name
	displayName := common.GetDisplayName(s, i.GuildID, i.Member.User.ID)

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
func (f *Feature) handleWagerCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Please specify a wager ID")
		return
	}

	wagerID := options[0].IntValue()
	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid user ID")
		return
	}

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate wager service with repositories from UnitOfWork
	wagerService := services.NewWagerService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.WagerVoteRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Get the wager details first to find the message
	wager, err := wagerService.GetWagerByID(context.Background(), wagerID)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to get wager: %v", err))
		return
	}
	if wager == nil {
		common.RespondWithError(s, i, "Wager not found")
		return
	}

	// Cancel the wager
	err = wagerService.CancelWager(context.Background(), wagerID, userID)
	if err != nil {
		common.RespondWithError(s, i, fmt.Sprintf("Failed to cancel wager: %v", err))
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Try to unpin and delete the wager message if we have the IDs
	if wager.MessageID != nil && wager.ChannelID != nil {
		messageID := strconv.FormatInt(*wager.MessageID, 10)
		channelID := strconv.FormatInt(*wager.ChannelID, 10)

		// Only unpin if the wager was accepted
		if wager.State == entities.WagerStateVoting {
			common.UnpinMessage(s, channelID, messageID)
		}

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
func (f *Feature) handleWagerInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "_")

	if len(parts) < 3 {
		common.RespondWithError(s, i, "Invalid interaction data")
		return
	}

	action := parts[1]
	switch action {
	case "accept", "decline":
		f.handleWagerResponse(s, i, action == "accept")
	case "vote":
		f.handleWagerVote(s, i)
	default:
		common.RespondWithError(s, i, "Unknown wager action")
	}
}

// handleWagerResponse handles accept/decline button presses
func (f *Feature) handleWagerResponse(s *discordgo.Session, i *discordgo.InteractionCreate, accept bool) {
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 3 {
		common.RespondWithError(s, i, "Invalid button data")
		return
	}

	wagerID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid wager ID")
		return
	}

	userID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid user ID")
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

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate services with repositories from UnitOfWork
	userService := services.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	wagerService := services.NewWagerService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.WagerVoteRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Ensure user exists in the database
	_, err = userService.GetOrCreateUser(context.Background(), userID, i.Member.User.Username)
	if err != nil {
		common.FollowUpWithError(s, i, "Unable to get user from DB")
		return
	}

	// Process the response
	wager, err := wagerService.RespondToWager(context.Background(), wagerID, userID, accept)
	if err != nil {
		common.FollowUpWithError(s, i, err.Error())
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get server-specific display names
	proposerName := common.GetDisplayNameInt64(s, i.GuildID, wager.ProposerDiscordID)
	targetName := common.GetDisplayNameInt64(s, i.GuildID, wager.TargetDiscordID)

	// Update the message based on response
	var embed *discordgo.MessageEmbed
	var components []discordgo.MessageComponent

	if accept {
		// Wager was accepted - pin the message
		if wager.MessageID != nil && wager.ChannelID != nil {
			messageID := strconv.FormatInt(*wager.MessageID, 10)
			channelID := strconv.FormatInt(*wager.ChannelID, 10)
			common.PinMessage(s, channelID, messageID)
		} else {
			log.Warnf("Cannot pin wager %d - MessageID: %v, ChannelID: %v", wager.ID, wager.MessageID, wager.ChannelID)
		}

		// Show voting interface
		voteCounts := &entities.VoteCount{} // Start with 0 votes
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
func (f *Feature) handleWagerVote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 4 {
		common.RespondWithError(s, i, "Invalid vote button data")
		return
	}

	wagerID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid wager ID")
		return
	}

	voteForID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid vote target ID")
		return
	}

	voterID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		common.RespondWithError(s, i, "Invalid voter ID")
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

	// Extract guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(context.Background()); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate services with repositories from UnitOfWork
	userService := services.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	wagerService := services.NewWagerService(
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.WagerVoteRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Ensure user exists in the database
	_, err = userService.GetOrCreateUser(context.Background(), voterID, i.Member.User.Username)
	if err != nil {
		common.FollowUpWithError(s, i, "Unable to get user from DB")
		return
	}

	// Cast the vote
	_, voteCounts, err := wagerService.CastVote(context.Background(), wagerID, voterID, voteForID)
	if err != nil {
		common.FollowUpWithError(s, i, err.Error())
		return
	}

	// Get updated wager state before committing
	wager, err := wagerService.GetWagerByID(context.Background(), wagerID)
	if err != nil {
		log.Printf("Error getting wager after vote: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.FollowUpWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get server-specific display names
	proposerName := common.GetDisplayNameInt64(s, i.GuildID, wager.ProposerDiscordID)
	targetName := common.GetDisplayNameInt64(s, i.GuildID, wager.TargetDiscordID)

	// Update the embed based on wager state
	var embed *discordgo.MessageEmbed
	var components []discordgo.MessageComponent

	if wager.State == entities.WagerStateResolved {
		// Wager has been resolved - unpin the message and send notification
		if wager.MessageID != nil && wager.ChannelID != nil {
			messageID := strconv.FormatInt(*wager.MessageID, 10)
			channelID := strconv.FormatInt(*wager.ChannelID, 10)
			common.UnpinMessage(s, channelID, messageID)
		}

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
