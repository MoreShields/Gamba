package services

import (
	"context"
	"fmt"
	"math/rand"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/utils"
)

type gamblingService struct {
	userRepo           interfaces.UserRepository
	betRepo            interfaces.BetRepository
	balanceHistoryRepo interfaces.BalanceHistoryRepository
	eventPublisher     interfaces.EventPublisher
}

// NewGamblingService creates a new gambling service
func NewGamblingService(userRepo interfaces.UserRepository, betRepo interfaces.BetRepository, balanceHistoryRepo interfaces.BalanceHistoryRepository, eventPublisher interfaces.EventPublisher) interfaces.GamblingService {
	return &gamblingService{
		userRepo:           userRepo,
		betRepo:            betRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		eventPublisher:     eventPublisher,
	}
}

func (s *gamblingService) PlaceBet(ctx context.Context, discordID int64, winProbability float64, betAmount int64) (*entities.BetResult, error) {
	// Validate inputs
	if winProbability <= 0 || winProbability >= 1 {
		return nil, fmt.Errorf("win probability must be between 0 and 1 (exclusive)")
	}
	if betAmount <= 0 {
		return nil, fmt.Errorf("bet amount must be positive")
	}

	// Get current user state (for calculating new balance)
	user, err := s.userRepo.GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Calculate potential win amount (no house edge)
	// If you bet X at probability P, you win X * ((1-P)/P) on success
	winAmount := int64(float64(betAmount) * ((1 - winProbability) / winProbability))

	// Roll the dice
	won := rand.Float64() < winProbability

	// Calculate new balance and change
	var newBalance int64
	var changeAmount int64
	var transactionType entities.TransactionType

	if won {
		newBalance = user.Balance + winAmount
		changeAmount = winAmount
		transactionType = entities.TransactionTypeBetWin

		// Update balance with winnings
		if err := s.userRepo.UpdateBalance(ctx, discordID, newBalance); err != nil {
			return nil, fmt.Errorf("failed to update balance with winnings: %w", err)
		}
	} else {
		newBalance = user.Balance - betAmount
		changeAmount = -betAmount
		transactionType = entities.TransactionTypeBetLoss

		// Check if sufficient balance before updating
		if user.AvailableBalance < betAmount {
			return nil, fmt.Errorf("insufficient balance: have %s available, need %s", utils.FormatShortNotation(user.AvailableBalance), utils.FormatShortNotation(betAmount))
		}

		// Update balance with bet deduction
		if err := s.userRepo.UpdateBalance(ctx, discordID, newBalance); err != nil {
			return nil, fmt.Errorf("failed to update balance with bet deduction: %w", err)
		}
	}

	// Create balance history record
	history := &entities.BalanceHistory{
		DiscordID:       discordID,
		GuildID:         0, // Will be set by repository from UoW's guild scope
		BalanceBefore:   user.Balance,
		BalanceAfter:    newBalance,
		ChangeAmount:    changeAmount,
		TransactionType: transactionType,
		TransactionMetadata: map[string]any{
			"bet_amount":      betAmount,
			"win_probability": winProbability,
			"won":             won,
		},
	}

	if err := utils.RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, history); err != nil {
		return nil, fmt.Errorf("failed to record balance change: %w", err)
	}

	// Create bet record
	bet := &entities.Bet{
		DiscordID:        discordID,
		GuildID:          0, // Will be set by repository from UoW's guild scope
		Amount:           betAmount,
		WinProbability:   winProbability,
		Won:              won,
		WinAmount:        winAmount,
		BalanceHistoryID: &history.ID,
	}

	if err := s.betRepo.Create(ctx, bet); err != nil {
		return nil, fmt.Errorf("failed to create bet record: %w", err)
	}

	return &entities.BetResult{
		Won:        won,
		BetAmount:  betAmount,
		WinAmount:  winAmount,
		NewBalance: newBalance,
	}, nil
}

