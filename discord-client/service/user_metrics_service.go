package service

import (
	"context"
	"fmt"
	"gambler/discord-client/models"
	"sort"
)

// calculateWinRate calculates win percentage from wins and total attempts
func calculateWinRate(wins, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(wins) / float64(total) * 100
}

// userMetricsService implements the UserMetricsService interface
type userMetricsService struct {
	userRepo       UserRepository
	wagerRepo      WagerRepository
	betRepo        BetRepository
	groupWagerRepo GroupWagerRepository
}

// NewUserMetricsService creates a new user metrics service
func NewUserMetricsService(
	userRepo UserRepository,
	wagerRepo WagerRepository,
	betRepo BetRepository,
	groupWagerRepo GroupWagerRepository,
) UserMetricsService {
	return &userMetricsService{
		userRepo:       userRepo,
		wagerRepo:      wagerRepo,
		betRepo:        betRepo,
		groupWagerRepo: groupWagerRepo,
	}
}

// GetScoreboard returns the top users with their statistics
func (s *userMetricsService) GetScoreboard(ctx context.Context, limit int) ([]*models.ScoreboardEntry, int64, error) {
	// Get all users
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	entries := make([]*models.ScoreboardEntry, 0, len(users))
	var totalBits int64

	for _, user := range users {
		// Add to total bits count
		totalBits += user.Balance

		// Skip users with zero balance for scoreboard display
		if user.Balance == 0 {
			continue
		}

		// Get active wagers count
		activeWagers, err := s.wagerRepo.GetActiveByUser(ctx, user.DiscordID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get active wagers for user %d: %w", user.DiscordID, err)
		}

		// Get wager stats for win rate
		wagerStats, err := s.wagerRepo.GetStats(ctx, user.DiscordID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get wager stats for user %d: %w", user.DiscordID, err)
		}

		// Get bet stats for win rate
		betStats, err := s.betRepo.GetStats(ctx, user.DiscordID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get bet stats for user %d: %w", user.DiscordID, err)
		}

		// Calculate win rates
		wagerWinRate := calculateWinRate(wagerStats.TotalWon, wagerStats.TotalResolved)
		betWinRate := calculateWinRate(betStats.TotalWins, betStats.TotalBets)

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

	return entries, totalBits, nil
}

// GetUserStats returns detailed statistics for a specific user
func (s *userMetricsService) GetUserStats(ctx context.Context, discordID int64) (*models.UserStats, error) {
	// Get user
	user, err := s.userRepo.GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Calculate reserved amount
	reservedInWagers := user.Balance - user.AvailableBalance

	// Get bet stats
	betStats, err := s.betRepo.GetStats(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bet stats: %w", err)
	}

	// Get wager stats
	wagerStats, err := s.wagerRepo.GetStats(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager stats: %w", err)
	}

	// Get group wager stats
	groupWagerStats, err := s.groupWagerRepo.GetStats(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager stats: %w", err)
	}

	// Convert to detail structs
	betDetail := &models.BetStatsDetail{
		TotalBets:    betStats.TotalBets,
		TotalWins:    betStats.TotalWins,
		TotalLosses:  betStats.TotalLosses,
		TotalWagered: betStats.TotalWagered,
		TotalWon:     betStats.TotalWon,
		TotalLost:    betStats.TotalLost,
		NetProfit:    betStats.TotalWon - betStats.TotalLost,
		BiggestWin:   betStats.BiggestWin,
		BiggestLoss:  betStats.BiggestLoss,
	}
	betDetail.WinPercentage = calculateWinRate(betStats.TotalWins, betStats.TotalBets)

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
	wagerDetail.WinPercentage = calculateWinRate(wagerStats.TotalWon, wagerStats.TotalResolved)

	stats := &models.UserStats{
		User:             user,
		BetStats:         betDetail,
		WagerStats:       wagerDetail,
		GroupWagerStats:  groupWagerStats,
		ReservedInWagers: reservedInWagers,
	}

	return stats, nil
}

// GetLOLPredictionStats calculates LOL-specific prediction stats for all users in a guild
func (s *userMetricsService) GetLOLPredictionStats(ctx context.Context) (map[int64]*models.LOLPredictionStats, error) {
	// Get all LOL wager predictions
	lolSystem := models.SystemLeagueOfLegends
	predictions, err := s.groupWagerRepo.GetGroupWagerPredictions(ctx, &lolSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get LOL wager predictions: %w", err)
	}

	// Aggregate by user
	userStats := make(map[int64]*models.LOLPredictionStats)

	for _, pred := range predictions {
		// Only count predictions on Win/Loss options
		if pred.OptionText != "Win" && pred.OptionText != "Loss" {
			continue
		}

		// Initialize user stats if not exists
		if _, exists := userStats[pred.DiscordID]; !exists {
			userStats[pred.DiscordID] = &models.LOLPredictionStats{
				DiscordID: pred.DiscordID,
			}
		}

		stats := userStats[pred.DiscordID]
		stats.TotalPredictions++
		stats.TotalAmountWagered += pred.Amount

		if pred.WasCorrect {
			stats.CorrectPredictions++
		}
	}

	// Calculate accuracy percentages
	for _, stats := range userStats {
		stats.CalculateAccuracy()
	}

	return userStats, nil
}

// GetWagerPredictionStats calculates generic prediction stats for all users in a guild
func (s *userMetricsService) GetWagerPredictionStats(ctx context.Context, externalSystem *models.ExternalSystem) (map[int64]*models.WagerPredictionStats, error) {
	// Get wager predictions, optionally filtered by external system
	predictions, err := s.groupWagerRepo.GetGroupWagerPredictions(ctx, externalSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager predictions: %w", err)
	}

	// Aggregate by user
	userStats := make(map[int64]*models.WagerPredictionStats)

	for _, pred := range predictions {
		// Initialize user stats if not exists
		if _, exists := userStats[pred.DiscordID]; !exists {
			userStats[pred.DiscordID] = &models.WagerPredictionStats{
				DiscordID: pred.DiscordID,
			}
		}

		stats := userStats[pred.DiscordID]
		stats.TotalPredictions++
		stats.TotalAmountWagered += pred.Amount

		if pred.WasCorrect {
			stats.CorrectPredictions++
		}
	}

	// Calculate accuracy percentages
	for _, stats := range userStats {
		stats.CalculateAccuracy()
	}

	return userStats, nil
}