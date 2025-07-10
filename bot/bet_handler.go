package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"gambler/config"
	"gambler/service"
	"github.com/bwmarrin/discordgo"
)

// BetSession stores temporary betting state for a user
type BetSession struct {
	UserID         int64
	MessageID      string
	ChannelID      string
	LastOdds       float64
	LastAmount     int64
	CurrentBalance int64
	Timestamp      time.Time
	// Session tracking fields
	StartingBalance int64 // Balance when session began
	SessionPnL      int64 // Running total P&L for this session
	BetCount        int   // Number of bets placed in session
}

var (
	betSessions   = make(map[int64]*BetSession)
	betSessionsMu sync.RWMutex
)

// cleanupSessions removes sessions older than 1 hour
func cleanupSessions() {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()

	now := time.Now()
	for userID, session := range betSessions {
		if now.Sub(session.Timestamp) > time.Hour {
			delete(betSessions, userID)
		}
	}
}

// getBetSession retrieves a bet session for a user
func getBetSession(userID int64) *BetSession {
	betSessionsMu.RLock()
	defer betSessionsMu.RUnlock()
	return betSessions[userID]
}

// saveBetSession stores or updates a bet session
func saveBetSession(session *BetSession) {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()
	session.Timestamp = time.Now()
	betSessions[session.UserID] = session
}

// handleBetCommand handles the initial /gamble slash command
func (b *Bot) handleBetCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Convert Discord ID
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get or create user
	user, err := b.userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Printf("Error getting/creating user %d: %v", discordID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Create initial embed with odds selection
	embed := buildInitialBetEmbed(user.AvailableBalance)
	components := buildOddsButtons()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		log.Printf("Error responding to bet command: %v", err)
		return
	}

	// Store session info for later use
	session := &BetSession{
		UserID:          discordID,
		CurrentBalance:  user.AvailableBalance,
		ChannelID:       i.ChannelID,
		StartingBalance: user.AvailableBalance,
		SessionPnL:      0,
		BetCount:        0,
	}
	saveBetSession(session)
}

// handleBetInteraction handles button clicks and modal submissions
func (b *Bot) handleBetInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID

		// Handle odds selection buttons
		if len(customID) > 9 && customID[:9] == "bet_odds_" {
			b.handleOddsSelection(s, i, customID[9:])
			return
		}

		// Handle action buttons
		switch customID {
		case "bet_new":
			b.handleNewBet(s, i)
		case "bet_repeat":
			b.handleRepeatSameBet(s, i)
		case "bet_double":
			b.handleDoubleBet(s, i)
		case "bet_halve":
			b.handleHalveBet(s, i)
		}

	case discordgo.InteractionModalSubmit:
		if i.ModalSubmitData().CustomID == "bet_amount_modal" {
			b.handleBetModal(s, i)
		}
	}
}

// handleOddsSelection handles when a user selects odds
func (b *Bot) handleOddsSelection(s *discordgo.Session, i *discordgo.InteractionCreate, oddsStr string) {
	ctx := context.Background()

	// Parse odds percentage
	oddsInt, err := strconv.Atoi(oddsStr)
	if err != nil {
		log.Printf("Error parsing odds %s: %v", oddsStr, err)
		return
	}
	odds := float64(oddsInt) / 100.0

	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing Discord ID: %v", err)
		return
	}

	// Get current user balance
	user, err := b.userService.GetUser(ctx, discordID)
	if err != nil {
		log.Printf("Error getting user %d: %v", discordID, err)
		b.respondWithError(s, i, "Unable to fetch current balance. Please try again.")
		return
	}

	session := getBetSession(discordID)
	if session == nil {
		// Create new session if doesn't exist
		session = &BetSession{
			UserID:    discordID,
			ChannelID: i.ChannelID,
		}
	}
	session.LastOdds = odds
	session.MessageID = i.Message.ID
	session.CurrentBalance = user.Balance
	saveBetSession(session)

	// Show bet amount modal
	modal := buildBetAmountModal(odds, user.AvailableBalance)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &modal,
	})

	if err != nil {
		log.Printf("Error showing bet modal: %v", err)
	}
}

// handleBetModal handles the bet amount modal submission
func (b *Bot) handleBetModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing Discord ID: %v", err)
		b.respondWithError(s, i, "Unable to process bet.")
		return
	}

	session := getBetSession(discordID)
	if session == nil {
		b.respondWithError(s, i, "Session expired. Please start a new bet.")
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
							b.respondWithError(s, i, "Invalid bet amount. Please enter a number.")
							return
						}
					}
				}
			}
		}
	}

	// Validate bet amount
	if err := validateBetAmount(betAmount, session.CurrentBalance); err != nil {
		b.respondWithError(s, i, err.Error())
		return
	}

	// Check daily limit
	remaining, err := b.gamblingService.CheckDailyLimit(ctx, discordID, betAmount)
	if err != nil {
		// Format error message with Discord timestamp for reset time
		cfg := config.Get()
		nextReset := service.GetNextResetTime(cfg.DailyLimitResetHour)
		
		if remaining <= 0 {
			b.respondWithError(s, i, fmt.Sprintf("Daily gambling limit of %s bits reached. Try again %s",
				FormatBalance(cfg.DailyGambleLimit),
				FormatDiscordTimestamp(nextReset, "R")))
		} else {
			b.respondWithError(s, i, fmt.Sprintf("Bet would exceed daily limit. You have %s bits remaining (resets %s)",
				FormatBalance(remaining),
				FormatDiscordTimestamp(nextReset, "R")))
		}
		return
	}

	// Process bet and update message
	if err := b.processBetAndUpdateMessage(ctx, s, i, session, betAmount, discordgo.InteractionResponseUpdateMessage); err != nil {
		b.respondWithError(s, i, "Unable to place bet. Please try again.")
	}
}

// handleNewBet starts a fresh bet
func (b *Bot) handleNewBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get current user balance
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing Discord ID: %v", err)
		return
	}

	user, err := b.userService.GetUser(ctx, discordID)
	if err != nil {
		log.Printf("Error getting user %d: %v", discordID, err)
		return
	}

	// Update session - reset PnL tracking for new bet session
	session := getBetSession(discordID)
	if session != nil {
		session.CurrentBalance = user.Balance
		session.StartingBalance = user.Balance
		session.SessionPnL = 0
		session.BetCount = 0
		saveBetSession(session)
	}

	// Show odds selection again as ephemeral
	embed := buildInitialBetEmbed(user.Balance)
	components := buildOddsButtons()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		log.Printf("Error showing new bet interface: %v", err)
	}
}

// handleRepeatSameBet repeats the last bet with the same amount
func (b *Bot) handleRepeatSameBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.handleRepeatBet(s, i, 1.0)
}

// handleDoubleBet repeats the last bet with double amount
func (b *Bot) handleDoubleBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.handleRepeatBet(s, i, 2.0)
}

// handleHalveBet repeats the last bet with half amount
func (b *Bot) handleHalveBet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.handleRepeatBet(s, i, 0.5)
}

// handleRepeatBet handles doubling or halving the previous bet
func (b *Bot) handleRepeatBet(s *discordgo.Session, i *discordgo.InteractionCreate, multiplier float64) {
	ctx := context.Background()

	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing Discord ID: %v", err)
		return
	}

	session := getBetSession(discordID)
	if session == nil || session.LastAmount == 0 {
		b.respondWithError(s, i, "No previous bet to repeat.")
		return
	}

	// Calculate new bet amount
	newAmount := int64(float64(session.LastAmount) * multiplier)
	if newAmount < 1 {
		newAmount = 1
	}

	// Validate new amount
	if err := validateBetAmount(newAmount, session.CurrentBalance); err != nil {
		// Show error as ephemeral follow-up
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})

		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Check daily limit
	remaining, err := b.gamblingService.CheckDailyLimit(ctx, discordID, newAmount)
	if err != nil {
		// Defer response first
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})

		// Format error message with Discord timestamp
		cfg := config.Get()
		nextReset := service.GetNextResetTime(cfg.DailyLimitResetHour)
		
		var errorMsg string
		if remaining <= 0 {
			errorMsg = fmt.Sprintf("Daily gambling limit of %s bits reached. Try again %s",
				FormatBalance(cfg.DailyGambleLimit),
				FormatDiscordTimestamp(nextReset, "R"))
		} else {
			errorMsg = fmt.Sprintf("Bet would exceed daily limit. You have %s bits remaining (resets %s)",
				FormatBalance(remaining),
				FormatDiscordTimestamp(nextReset, "R"))
		}

		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: errorMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Defer the response while we process the bet
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Error deferring interaction: %v", err)
		return
	}

	// Process bet and update message using deferred response
	if err := b.processBetAndUpdateMessage(ctx, s, i, session, newAmount, discordgo.InteractionResponseDeferredMessageUpdate); err != nil {
		// Send error as ephemeral follow-up
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "Unable to place bet. Please try again.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
	}
}

// validateBetAmount validates the bet amount
func validateBetAmount(amount, balance int64) error {
	if amount <= 0 {
		return fmt.Errorf("Bet amount must be greater than 0")
	}
	if amount > balance {
		return fmt.Errorf("Insufficient balance. You have %s bits", FormatBalance(balance))
	}
	return nil
}

// processBetAndUpdateMessage handles the common logic for placing a bet, updating session, and responding
func (b *Bot) processBetAndUpdateMessage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate,
	session *BetSession, betAmount int64, responseType discordgo.InteractionResponseType) error {

	// Place the bet
	result, err := b.gamblingService.PlaceBet(ctx, session.UserID, session.LastOdds, betAmount)
	if err != nil {
		log.Printf("Error placing bet for user %d: %v", session.UserID, err)
		return fmt.Errorf("unable to place bet: %w", err)
	}

	// Update session with bet info and PnL tracking
	session.LastAmount = betAmount
	session.CurrentBalance = result.NewBalance
	session.BetCount++

	// Update session PnL
	if result.Won {
		session.SessionPnL += result.WinAmount
	} else {
		session.SessionPnL -= betAmount
	}

	saveBetSession(session)

	// Build result embed
	var embed *discordgo.MessageEmbed
	if result.Won {
		embed = buildWinEmbed(result, session.LastOdds, session)
	} else {
		embed = buildLossEmbed(result, session.LastOdds, session)
	}

	// Add action buttons
	components := buildActionButtons(betAmount, result.NewBalance)

	// For the initial bet from modal (coming from ephemeral message), 
	// we need to create a new public message with the result
	if responseType == discordgo.InteractionResponseUpdateMessage && i.Message != nil && i.Message.Flags&discordgo.MessageFlagsEphemeral != 0 {
		// First acknowledge the interaction by updating the ephemeral message
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: responseType,
			Data: &discordgo.InteractionResponseData{
				Content: "Bet placed! See result below.",
				Embeds:  []*discordgo.MessageEmbed{},
				Components: []discordgo.MessageComponent{},
			},
		})
		if err != nil {
			log.Printf("Error updating ephemeral message: %v", err)
		}

		// Then create a new public message with the bet result
		newMsg, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
		if err != nil {
			log.Printf("Error creating public bet result message: %v", err)
			return fmt.Errorf("failed to create result message: %w", err)
		}

		// Update session with new message ID for future interactions
		session.MessageID = newMsg.ID
		saveBetSession(session)
	} else if responseType == discordgo.InteractionResponseUpdateMessage {
		// For button interactions on public messages - update the existing message
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: responseType,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
	} else {
		// For deferred interactions - edit the response
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
	}

	if err != nil {
		log.Printf("Error updating bet result: %v", err)
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}
