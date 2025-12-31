package betting

import (
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// processBetAndUpdateMessage processes a bet and updates the message with results
func (f *Feature) processBetAndUpdateMessage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, session *BetSession, betAmount int64) error {
	// Create guild-scoped unit of work for bet placement
	uow, err := f.createUnitOfWork(ctx, i)
	if err != nil {
		log.Errorf("Error creating unit of work for bet: %v", err)
		return fmt.Errorf("unable to create transaction: %w", err)
	}
	defer uow.Rollback()

	// Instantiate gambling service with repositories from UnitOfWork
	gamblingService := services.NewGamblingService(
		uow.UserRepository(),
		uow.BetRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Place the bet (swap the order - PlaceBet expects amount first, then odds)
	result, err := gamblingService.PlaceBet(ctx, session.UserID, session.LastOdds, betAmount)
	if err != nil {
		log.Errorf("Error placing bet for user %d: %v", session.UserID, err)
		return fmt.Errorf("unable to place bet: %w", err)
	}

	// Get updated user to fetch available balance after bet (within same transaction)
	updatedUser, err := uow.UserRepository().GetByDiscordID(ctx, session.UserID)
	if err != nil {
		log.Errorf("Error getting updated user balance: %v", err)
		return fmt.Errorf("unable to get updated balance: %w", err)
	}
	if updatedUser == nil {
		return fmt.Errorf("user not found after bet")
	}

	// Commit the bet transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing bet transaction: %v", err)
		return fmt.Errorf("unable to commit bet: %w", err)
	}

	// Update session with new available balance and bet info
	updateSessionBalance(session.UserID, updatedUser.AvailableBalance, true)
	updateBetSession(session.UserID, session.LastOdds, betAmount)

	// Get display name for embed and logging
	displayName := common.GetDisplayNameInt64(s, i.GuildID, session.UserID)

	// Create result embed based on win/loss
	var embed *discordgo.MessageEmbed
	if result.Won {
		embed = buildWinEmbed(result, session.LastOdds, session, session.UserID)
	} else {
		embed = buildLossEmbed(result, session.LastOdds, session, session.UserID)
	}

	// Create action buttons for next bet (use available balance from session)
	components := CreateActionButtons(betAmount, session.CurrentBalance)

	// Update the original message with bet results
	err = common.UpdateMessage(s, i, embed, components)
	if err != nil {
		log.Errorf("Error updating bet message: %v", err)
		return fmt.Errorf("unable to update message: %w", err)
	}

	// Log the bet
	if result.Won {
		log.Infof("Bet WON: %s wagered %s at %.0f%% odds, won %s. New balance: %s",
			displayName,
			common.FormatBalance(betAmount),
			session.LastOdds*100,
			common.FormatBalance(result.WinAmount),
			common.FormatBalance(result.NewBalance))
	} else {
		log.Infof("Bet LOST: %s wagered %s at %.0f%% odds. New balance: %s",
			displayName,
			common.FormatBalance(betAmount),
			session.LastOdds*100,
			common.FormatBalance(result.NewBalance))
	}

	return nil
}

// processRepeatBet handles repeating bets with multipliers (half, same, double)
func (f *Feature) processRepeatBet(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, multiplier float64) error {
	// Get user session
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID: %v", err)
		return fmt.Errorf("invalid user ID: %w", err)
	}

	session := getBetSession(discordID)
	if session == nil || session.LastAmount == 0 {
		// No session found - show odds selection instead
		// Since interaction is already acknowledged, we use update instead of respond
		if err := f.showOddsSelectionUpdate(ctx, s, i); err != nil {
			return fmt.Errorf("unable to show odds selection: %w", err)
		}
		return nil
	}

	// Calculate new bet amount
	newAmount := max(int64(float64(session.LastAmount) * multiplier), 1)

	// Validate new amount
	if err := validateBetAmount(newAmount, session.CurrentBalance); err != nil {
		return fmt.Errorf("bet validation failed: %w", err)
	}

	// Process bet and update existing message
	return f.processBetAndUpdateMessage(ctx, s, i, session, newAmount)
}

// showOddsSelectionUpdate updates an already-acknowledged interaction with odds selection
func (f *Feature) showOddsSelectionUpdate(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Prepare gamble data
	embed, components, balance, err := f.prepareGambleData(ctx, s, i)
	if err != nil {
		return err
	}

	// Parse user ID for session creation
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get the message from the interaction to create session
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Errorf("Error getting interaction response: %v", err)
		return fmt.Errorf("unable to get message: %w", err)
	}

	// Create new betting session
	createBetSession(discordID, msg.ID, msg.ChannelID, balance)

	// Update message with odds selection
	err = common.UpdateMessage(s, i, embed, components)
	if err != nil {
		log.Errorf("Error updating message with odds selection: %v", err)
		return fmt.Errorf("unable to update message: %w", err)
	}

	return nil
}

// validateBetAmount validates the bet amount against balance and limits
func validateBetAmount(amount int64, balance int64) error {
	if amount <= 0 {
		return fmt.Errorf("bet amount must be positive")
	}

	if amount > balance {
		return fmt.Errorf("insufficient balance. You have %s bits", common.FormatBalance(balance))
	}

	return nil
}
