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

	// Get current balance
	user, err := f.userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting user balance: %v", err)
		common.RespondWithError(s, i, "Unable to fetch current balance. Please try again.")
		return
	}

	// Update session balance
	updateSessionBalance(discordID, user.AvailableBalance, false)

	// Show bet amount modal
	modal := buildBetAmountModal(odds, user.AvailableBalance)
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

	// Check daily limit
	remaining, err := f.gamblingService.CheckDailyLimit(ctx, discordID, betAmount)
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

	// Process bet and update message
	if err := f.processBetAndUpdateMessage(ctx, s, i, session, betAmount, discordgo.InteractionResponseUpdateMessage); err != nil {
		common.RespondWithError(s, i, "Unable to place bet. Please try again.")
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

	user, err := f.userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting user %d: %v", discordID, err)
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

	// Show odds selection again as ephemeral
	embed := buildInitialBetEmbed(user.Balance)
	components := CreateInitialComponents()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
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

	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID: %v", err)
		return
	}

	session := getBetSession(discordID)
	if session == nil || session.LastAmount == 0 {
		common.RespondWithError(s, i, "No previous bet to repeat.")
		return
	}

	// Calculate new bet amount
	newAmount := int64(float64(session.LastAmount) * multiplier)
	if newAmount < 1 {
		newAmount = 1
	}

	// Validate new amount
	if err := validateBetAmount(newAmount, session.CurrentBalance); err != nil {
		// Show error as a new ephemeral message
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ %s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Errorf("Error sending validation error: %v", err)
		}
		return
	}

	// Check daily limit
	remaining, err := f.gamblingService.CheckDailyLimit(ctx, discordID, newAmount)
	if err != nil {
		cfg := f.config
		nextReset := service.GetNextResetTime(cfg.DailyLimitResetHour)
		
		var errorMsg string
		if remaining <= 0 {
			errorMsg = fmt.Sprintf("❌ Daily gambling limit of %s bits reached. Try again %s",
				common.FormatBalance(cfg.DailyGambleLimit),
				common.FormatDiscordTimestamp(nextReset, "R"))
		} else {
			errorMsg = fmt.Sprintf("❌ Bet would exceed daily limit. You have %s bits remaining (resets %s)",
				common.FormatBalance(remaining),
				common.FormatDiscordTimestamp(nextReset, "R"))
		}
		
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errorMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Errorf("Error sending limit error: %v", err)
		}
		return
	}

	// Process bet and update message
	if err := f.processBetAndUpdateMessage(ctx, s, i, session, newAmount, discordgo.InteractionResponseUpdateMessage); err != nil {
		log.Errorf("Error processing repeat bet: %v", err)
		common.RespondWithError(s, i, "Unable to place bet. Please try again.")
	}
}