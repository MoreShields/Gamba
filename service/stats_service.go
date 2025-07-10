package service

import (
	"context"
	"fmt"
	"gambler/models"
	"sort"
)

// statsService implements the StatsService interface
type statsService struct {
	uowFactory UnitOfWorkFactory
}

// NewStatsService creates a new stats service
func NewStatsService(uowFactory UnitOfWorkFactory) StatsService {
	return &statsService{
		uowFactory: uowFactory,
	}
}

// GetScoreboard returns the top users with their statistics
func (s *statsService) GetScoreboard(ctx context.Context, limit int) ([]*models.ScoreboardEntry, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get all users
	users, err := uow.UserRepository().GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	entries := make([]*models.ScoreboardEntry, 0, len(users))
	
	for _, user := range users {
		// Skip users with zero balance
		if user.Balance == 0 {
			continue
		}

		// Get active wagers count
		activeWagers, err := uow.WagerRepository().GetActiveByUser(ctx, user.DiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get active wagers for user %d: %w", user.DiscordID, err)
		}

		// Get wager stats for win rate
		wagerStats, err := uow.WagerRepository().GetStats(ctx, user.DiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get wager stats for user %d: %w", user.DiscordID, err)
		}

		// Get bet stats for win rate
		betStats, err := uow.BetRepository().GetStats(ctx, user.DiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get bet stats for user %d: %w", user.DiscordID, err)
		}

		// Calculate win rates
		var wagerWinRate float64
		if wagerStats.TotalResolved > 0 {
			wagerWinRate = float64(wagerStats.TotalWon) / float64(wagerStats.TotalResolved) * 100
		}

		var betWinRate float64
		if betStats.TotalBets > 0 {
			betWinRate = float64(betStats.TotalWins) / float64(betStats.TotalBets) * 100
		}

		entry := &models.ScoreboardEntry{
			DiscordID:        user.DiscordID,
			Username:         user.Username,
			TotalBalance:     user.Balance,
			AvailableBalance: user.AvailableBalance,
			ActiveWagerCount: len(activeWagers),
			WagerWinRate:     wagerWinRate,
			BetWinRate:       betWinRate,
		}

		entries = append(entries, entry)
	}

	// Sort by total balance descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TotalBalance > entries[j].TotalBalance
	})

	// Add rank
	for i := range entries {
		entries[i].Rank = i + 1
	}

	// Apply limit
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

// GetUserStats returns detailed statistics for a specific user
func (s *statsService) GetUserStats(ctx context.Context, discordID int64) (*models.UserStats, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get user
	user, err := uow.UserRepository().GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Calculate reserved amount
	reservedInWagers := user.Balance - user.AvailableBalance

	// Get bet stats
	betStats, err := uow.BetRepository().GetStats(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bet stats: %w", err)
	}

	// Get wager stats
	wagerStats, err := uow.WagerRepository().GetStats(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager stats: %w", err)
	}

	// Convert to detail structs
	betDetail := &models.BetStatsDetail{
		TotalBets:     betStats.TotalBets,
		TotalWins:     betStats.TotalWins,
		TotalLosses:   betStats.TotalLosses,
		TotalWagered:  betStats.TotalWagered,
		TotalWon:      betStats.TotalWon,
		TotalLost:     betStats.TotalLost,
		NetProfit:     betStats.TotalWon - betStats.TotalLost,
		BiggestWin:    betStats.BiggestWin,
		BiggestLoss:   betStats.BiggestLoss,
	}
	if betStats.TotalBets > 0 {
		betDetail.WinPercentage = float64(betStats.TotalWins) / float64(betStats.TotalBets) * 100
	}

	wagerDetail := &models.WagerStatsDetail{
		TotalWagers:    wagerStats.TotalWagers,
		TotalProposed:  wagerStats.TotalProposed,
		TotalAccepted:  wagerStats.TotalAccepted,
		TotalDeclined:  wagerStats.TotalDeclined,
		TotalResolved:  wagerStats.TotalResolved,
		TotalWon:       wagerStats.TotalWon,
		TotalLost:      wagerStats.TotalLost,
		TotalAmount:    wagerStats.TotalAmount,
		TotalWonAmount: wagerStats.TotalWonAmount,
		BiggestWin:     wagerStats.BiggestWin,
		BiggestLoss:    wagerStats.BiggestLoss,
	}
	if wagerStats.TotalResolved > 0 {
		wagerDetail.WinPercentage = float64(wagerStats.TotalWon) / float64(wagerStats.TotalResolved) * 100
	}

	stats := &models.UserStats{
		User:             user,
		BetStats:         betDetail,
		WagerStats:       wagerDetail,
		ReservedInWagers: reservedInWagers,
	}

	return stats, nil
}