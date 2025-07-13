package betting

import (
	"context"
	"fmt"
	"strconv"

	"gambler/bot/common"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleGamble handles the main /gamble command logic
func (f *Feature) handleGamble(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Parse user ID
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
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

	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate services with repositories from UnitOfWork
	userService := service.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	gamblingService := service.NewGamblingService(
		uow.UserRepository(),
		uow.BetRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Get or create user
	user, err := userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting/creating user %d: %v", discordID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Check daily limit
	remaining, _ := gamblingService.CheckDailyLimit(ctx, discordID, 0)
	if remaining == 0 {
		// Format error message with Discord timestamp for reset time
		cfg := f.config
		nextReset := service.GetNextResetTime(cfg.DailyLimitResetHour)
		common.RespondWithError(s, i, fmt.Sprintf("Daily gambling limit of %s bits reached. Try again %s",
			common.FormatBalance(cfg.DailyGambleLimit),
			common.FormatDiscordTimestamp(nextReset, "R")))

		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create initial embed
	embed := buildInitialBetEmbed(user.AvailableBalance, remaining)
	components := CreateInitialComponents()

	// Send initial response (public message)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})

	if err != nil {
		log.Errorf("Error responding to bet command: %v", err)
		return
	}

	// Create betting session
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Errorf("Error getting interaction response: %v", err)
		return
	}

	createBetSession(discordID, msg.ID, msg.ChannelID, user.AvailableBalance)
}

// handleComponentInteraction handles button clicks for betting
func (f *Feature) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Handle odds selection buttons
	if len(customID) > 9 && customID[:9] == "bet_odds_" {
		f.handleOddsSelection(s, i, customID[9:])
		return
	}

	// Handle action buttons
	switch customID {
	case "bet_new":
		f.handleNewBet(s, i)
	case "bet_repeat":
		f.handleRepeatSameBet(s, i)
	case "bet_double":
		f.handleDoubleBet(s, i)
	case "bet_halve":
		f.handleHalveBet(s, i)
	}
}

// handleModalSubmit handles bet amount modal submissions
func (f *Feature) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ModalSubmitData().CustomID == "bet_amount_modal" {
		f.handleBetModal(s, i)
	}
}

// handleOddsSelection handles when a user selects odds
func (f *Feature) handleOddsSelection(s *discordgo.Session, i *discordgo.InteractionCreate, oddsStr string) {
	ctx := context.Background()

	// Parse odds percentage
	oddsInt, err := strconv.Atoi(oddsStr)
	if err != nil {
		log.Errorf("Error parsing odds %s: %v", oddsStr, err)
		return
	}
	odds := float64(oddsInt) / 100.0

	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID: %v", err)
		return
	}

	// Update session with selected odds
	updateBetSession(discordID, odds, 0)

	// Create guild-scoped unit of work
	uow, err := f.createUnitOfWork(ctx, i)
	if err != nil {
		log.Errorf("Error creating unit of work: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate services with repositories from UnitOfWork
	userService := service.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	gamblingService := service.NewGamblingService(
		uow.UserRepository(),
		uow.BetRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Get current balance
	user, err := userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting user balance: %v", err)
		common.RespondWithError(s, i, "Unable to fetch current balance. Please try again.")
		return
	}

	// Check remaining daily limit
	remainingLimit, _ := gamblingService.CheckDailyLimit(ctx, discordID, 1)
	// If error, remainingLimit will be 0 which is fine - we'll just not show the limit

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Update session balance
	updateSessionBalance(discordID, user.AvailableBalance, false)

	// Show bet amount modal
	modal := buildBetAmountModal(odds, user.AvailableBalance, remainingLimit)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})

	if err != nil {
		log.Errorf("Error showing bet amount modal: %v", err)
	}
}

// handleBetModal processes the bet amount modal submission
func (f *Feature) handleBetModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID: %v", err)
		common.RespondWithError(s, i, "Unable to process bet.")
		return
	}

	session := getBetSession(discordID)
	if session == nil {
		common.RespondWithError(s, i, "Session expired. Please start a new bet.")
		return
	}

	// Extract bet amount from modal
	var betAmount int64
	data := i.ModalSubmitData()
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, innerComp := range row.Components {
				if textInput, ok := innerComp.(*discordgo.TextInput); ok {
					if textInput.CustomID == "bet_amount_input" {
						betAmount, err = strconv.ParseInt(textInput.Value, 10, 64)
						if err != nil {
							common.RespondWithError(s, i, "Invalid bet amount. Please enter a number.")
							return
						}
					}
				}
			}
		}
	}

	// Validate bet amount
	if err := validateBetAmount(betAmount, session.CurrentBalance); err != nil {
		common.RespondWithError(s, i, err.Error())
		return
	}

	// Create guild-scoped unit of work
	uow, err := f.createUnitOfWork(ctx, i)
	if err != nil {
		log.Errorf("Error creating unit of work: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}
	defer uow.Rollback()

	// Instantiate gambling service with repositories from UnitOfWork
	gamblingService := service.NewGamblingService(
		uow.UserRepository(),
		uow.BetRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Check daily limit
	remaining, err := gamblingService.CheckDailyLimit(ctx, discordID, betAmount)
	if err != nil {
		// Format error message with Discord timestamp for reset time
		cfg := f.config
		nextReset := service.GetNextResetTime(cfg.DailyLimitResetHour)

		if remaining <= 0 {
			common.RespondWithError(s, i, fmt.Sprintf("Daily gambling limit of %s bits reached. Try again %s",
				common.FormatBalance(cfg.DailyGambleLimit),
				common.FormatDiscordTimestamp(nextReset, "R")))
		} else {
			common.RespondWithError(s, i, fmt.Sprintf("Bet would exceed daily limit. You have %s bits remaining (resets %s)",
				common.FormatBalance(remaining),
				common.FormatDiscordTimestamp(nextReset, "R")))
		}
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Acknowledge the modal with a deferred update to the original message
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Errorf("Error deferring modal response: %v", err)
		return
	}

	// Process bet and update the original message
	if err := f.processBetAndUpdateMessage(ctx, s, i, session, betAmount); err != nil {
		common.UpdateMessageWithError(s, i, "Unable to place bet. Please try again.")
	}
}

// handleNewBet starts a fresh bet
func (f *Feature) handleNewBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get current user balance
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID: %v", err)
		return
	}

	// Create guild-scoped unit of work
	uow, err := f.createUnitOfWork(ctx, i)
	if err != nil {
		log.Errorf("Error creating unit of work: %v", err)
		return
	}
	defer uow.Rollback()

	// Instantiate services with repositories from UnitOfWork
	userService := service.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	gamblingService := service.NewGamblingService(
		uow.UserRepository(),
		uow.BetRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	user, err := userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting user %d: %v", discordID, err)
		return
	}

	// Check Daily limit
	// error already handled during initial bet
	remaining, _ := gamblingService.CheckDailyLimit(ctx, discordID, 0)

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing transaction: %v", err)
		return
	}

	// Update session - reset PnL tracking for new bet session
	session := getBetSession(discordID)
	if session != nil {
		session.CurrentBalance = user.Balance
		session.StartingBalance = user.Balance
		session.SessionPnL = 0
		session.BetCount = 0
		updateSessionBalance(session.UserID, user.Balance, false)
	}

	// Show odds selection again as public message
	embed := buildInitialBetEmbed(user.Balance, remaining)
	components := CreateInitialComponents()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})

	if err != nil {
		log.Errorf("Error showing new bet interface: %v", err)
	}
}

// handleRepeatSameBet repeats the last bet with the same amount
func (f *Feature) handleRepeatSameBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleRepeatBet(s, i, 1.0)
}

// handleDoubleBet repeats the last bet with double amount
func (f *Feature) handleDoubleBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleRepeatBet(s, i, 2.0)
}

// handleHalveBet repeats the last bet with half amount
func (f *Feature) handleHalveBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleRepeatBet(s, i, 0.5)
}

// handleRepeatBet handles doubling or halving the previous bet
func (f *Feature) handleRepeatBet(s *discordgo.Session, i *discordgo.InteractionCreate, multiplier float64) {
	ctx := context.Background()

	// Acknowledge the button interaction with a deferred update (no new message)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Errorf("Error deferring repeat bet response: %v", err)
		return
	}

	if err := f.processRepeatBet(ctx, s, i, multiplier); err != nil {
		// Handle specific error types with user-friendly messages
		switch {
		case fmt.Sprintf("%v", err)[:4] == "bet ":
			// Bet validation error - show as ephemeral
			common.UpdateMessageWithError(s, i, err.Error()[19:]) // Remove "bet validation failed: " prefix
		case fmt.Sprintf("%v", err)[:5] == "daily":
			// Daily limit error - show as ephemeral
			common.UpdateMessageWithError(s, i, err.Error()[23:]) // Remove "daily limit exceeded: " prefix
		default:
			log.Errorf("Error processing repeat bet: %v", err)
			common.UpdateMessageWithError(s, i, "Unable to place bet. Please try again.")
		}
	}
}
