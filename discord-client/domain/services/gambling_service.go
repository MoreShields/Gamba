package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"gambler/discord-client/config"
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

	// Check daily risk limit
	cfg := config.Get()
	limitStart := utils.GetCurrentPeriodStart(cfg.DailyLimitResetHour)

	// Get total risk amount for the day
	dailyRisk, err := s.GetDailyRiskAmount(ctx, discordID, limitStart)
	if err != nil {
		return nil, fmt.Errorf("failed to check daily risk amount: %w", err)
	}

	// Check if adding this bet would exceed the daily limit
	if dailyRisk+betAmount > cfg.DailyGambleLimit {
		remainingLimit := cfg.DailyGambleLimit - dailyRisk
		if remainingLimit <= 0 {
			return nil, fmt.Errorf("daily gambling limit of %s bits reached", utils.FormatShortNotation(cfg.DailyGambleLimit))
		}
		return nil, fmt.Errorf("bet amount would exceed daily limit. You have %s bits remaining today", utils.FormatShortNotation(remainingLimit))
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

func (s *gamblingService) GetDailyRiskAmount(ctx context.Context, discordID int64, since time.Time) (int64, error) {
	// Get all bets since the specified time
	bets, err := s.betRepo.GetByUserSince(ctx, discordID, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get bets since %v: %w", since, err)
	}

	// Calculate total amount risked (not won, just the bet amounts)
	var totalRisked int64
	for _, bet := range bets {
		totalRisked += bet.Amount
	}

	return totalRisked, nil
}

func (s *gamblingService) CheckDailyLimit(ctx context.Context, discordID int64, betAmount int64) (remaining int64, err error) {
	cfg := config.Get()

	// Get the current period start time
	periodStart := utils.GetCurrentPeriodStart(cfg.DailyLimitResetHour)

	// Get total risk amount for the current period
	dailyRisk, err := s.GetDailyRiskAmount(ctx, discordID, periodStart)
	if err != nil {
		return 0, fmt.Errorf("failed to check daily risk amount: %w", err)
	}

	// Calculate remaining limit
	remaining = cfg.DailyGambleLimit - dailyRisk

	// Check if adding this bet would exceed the limit
	if dailyRisk+betAmount > cfg.DailyGambleLimit {
		if remaining <= 0 {
			return 0, fmt.Errorf("daily gambling limit of %s bits reached", utils.FormatShortNotation(cfg.DailyGambleLimit))
		}
		return remaining, fmt.Errorf("bet amount of %s would exceed daily limit", utils.FormatShortNotation(betAmount))
	}

	return remaining, nil
}
