package betting

import (
	"context"
	"fmt"

	"gambler/bot/common"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// processBetAndUpdateMessage processes a bet and updates the Discord message
func (f *Feature) processBetAndUpdateMessage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, session *BetSession, betAmount int64, responseType discordgo.InteractionResponseType) error {
	// Place the bet (swap the order - PlaceBet expects amount first, then odds)
	result, err := f.gamblingService.PlaceBet(ctx, session.UserID, session.LastOdds, betAmount)
	if err != nil {
		log.Errorf("Error placing bet for user %d: %v", session.UserID, err)
		return fmt.Errorf("unable to place bet: %w", err)
	}

	// Update session with new balance and bet info
	updateSessionBalance(session.UserID, result.NewBalance, true)
	updateBetSession(session.UserID, session.LastOdds, betAmount)

	// Get updated session for embed
	updatedSession := getBetSession(session.UserID)
	if updatedSession == nil {
		return fmt.Errorf("session lost after bet")
	}

	// Create result embed based on win/loss
	var embed *discordgo.MessageEmbed
	if result.Won {
		embed = buildWinEmbed(result, session.LastOdds, updatedSession)
	} else {
		embed = buildLossEmbed(result, session.LastOdds, updatedSession)
	}

	// Create action buttons for next bet
	components := CreateActionButtons(result.NewBalance)

	// Send the response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})

	if err != nil {
		log.Errorf("Error updating bet message: %v", err)
		return fmt.Errorf("unable to update message: %w", err)
	}

	// Get display name for logging
	displayName := common.GetDisplayNameInt64(s, f.guildID, session.UserID)

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

