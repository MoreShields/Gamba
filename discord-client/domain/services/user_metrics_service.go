package services

import (
	"context"
	"fmt"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
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
	userRepo            interfaces.UserRepository
	wagerRepo           interfaces.WagerRepository
	betRepo             interfaces.BetRepository
	groupWagerRepo      interfaces.GroupWagerRepository
	balanceHistoryRepo  interfaces.BalanceHistoryRepository
}

// NewUserMetricsService creates a new user metrics service
func NewUserMetricsService(
	userRepo interfaces.UserRepository,
	wagerRepo interfaces.WagerRepository,
	betRepo interfaces.BetRepository,
	groupWagerRepo interfaces.GroupWagerRepository,
	balanceHistoryRepo interfaces.BalanceHistoryRepository,
) interfaces.UserMetricsService {
	return &userMetricsService{
		userRepo:           userRepo,
		wagerRepo:          wagerRepo,
		betRepo:            betRepo,
		groupWagerRepo:     groupWagerRepo,
		balanceHistoryRepo: balanceHistoryRepo,
	}
}

// GetScoreboard returns the top users with their statistics
func (s *userMetricsService) GetScoreboard(ctx context.Context, limit int) ([]*entities.ScoreboardEntry, int64, error) {
	// Single optimized query gets all scoreboard data AND total server bits
	entries, totalBits, err := s.userRepo.GetScoreboardData(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get scoreboard data: %w", err)
	}

	// Apply limit if specified
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, totalBits, nil
}

// GetUserStats returns detailed statistics for a specific user
func (s *userMetricsService) GetUserStats(ctx context.Context, discordID int64) (*entities.UserStats, error) {
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
	betDetail := &entities.BetStatsDetail{
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

	wagerDetail := &entities.WagerStatsDetail{
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

	stats := &entities.UserStats{
		User:             user,
		BetStats:         betDetail,
		WagerStats:       wagerDetail,
		GroupWagerStats:  groupWagerStats,
		ReservedInWagers: reservedInWagers,
	}

	return stats, nil
}

// GetLOLPredictionStats calculates LOL-specific prediction stats for all users in a guild
func (s *userMetricsService) GetLOLPredictionStats(ctx context.Context) (map[int64]*entities.LOLPredictionStats, error) {
	// Get all LOL wager predictions
	lolSystem := entities.SystemLeagueOfLegends
	predictions, err := s.groupWagerRepo.GetGroupWagerPredictions(ctx, &lolSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get LOL wager predictions: %w", err)
	}

	// Aggregate by user
	userStats := make(map[int64]*entities.LOLPredictionStats)

	for _, pred := range predictions {
		// Only count predictions on Win/Loss options
		if pred.OptionText != "Win" && pred.OptionText != "Loss" {
			continue
		}

		// Initialize user stats if not exists
		if _, exists := userStats[pred.DiscordID]; !exists {
			userStats[pred.DiscordID] = &entities.LOLPredictionStats{
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
func (s *userMetricsService) GetWagerPredictionStats(ctx context.Context, externalSystem *entities.ExternalSystem) (map[int64]*entities.WagerPredictionStats, error) {
	// Get wager predictions, optionally filtered by external system
	predictions, err := s.groupWagerRepo.GetGroupWagerPredictions(ctx, externalSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager predictions: %w", err)
	}

	// Aggregate by user
	userStats := make(map[int64]*entities.WagerPredictionStats)

	for _, pred := range predictions {
		// Initialize user stats if not exists
		if _, exists := userStats[pred.DiscordID]; !exists {
			userStats[pred.DiscordID] = &entities.WagerPredictionStats{
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

// getGameLeaderboard is a generic method to get prediction leaderboard entries for a specific game system
func (s *userMetricsService) getGameLeaderboard(ctx context.Context, minWagers int, system entities.ExternalSystem) ([]*entities.LOLLeaderboardEntry, int64, error) {
	// Get all wager predictions for the specified system
	predictions, err := s.groupWagerRepo.GetGroupWagerPredictions(ctx, &system)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get %s wager predictions: %w", system, err)
	}

	// Group predictions by user for profit/loss calculation
	userPredictions := make(map[int64][]*entities.GroupWagerPrediction)
	for _, pred := range predictions {
		userPredictions[pred.DiscordID] = append(userPredictions[pred.DiscordID], pred)
	}

	// Build leaderboard entries
	entries := make([]*entities.LOLLeaderboardEntry, 0)
	var totalBitsWagered int64

	for discordID, preds := range userPredictions {
		entry := &entities.LOLLeaderboardEntry{
			DiscordID: discordID,
		}

		// Calculate stats and profit/loss using actual payout data
		for _, pred := range preds {
			entry.TotalPredictions++
			entry.TotalAmountWagered += pred.Amount
			totalBitsWagered += pred.Amount

			if pred.WasCorrect {
				entry.CorrectPredictions++
				// Use actual payout from database
				var payout int64
				if pred.PayoutAmount != nil {
					payout = *pred.PayoutAmount
				}
				// Profit = payout - original bet
				profit := payout - pred.Amount
				entry.ProfitLoss += profit
			} else {
				// Losers lose their bet amount
				entry.ProfitLoss -= pred.Amount
			}
		}

		// Calculate accuracy percentage
		entry.CalculateAccuracy()

		// Only include users who meet minimum wager requirement
		if entry.QualifiesForLeaderboard(minWagers) {
			entries = append(entries, entry)
		}
	}

	// Sort by profit/loss (descending)
	sort.Slice(entries, func(i, j int) bool {
		// If profit/loss is the same, sort by total predictions (more = higher rank)
		if entries[i].ProfitLoss == entries[j].ProfitLoss {
			return entries[i].TotalPredictions > entries[j].TotalPredictions
		}
		return entries[i].ProfitLoss > entries[j].ProfitLoss
	})

	// Assign ranks
	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries, totalBitsWagered, nil
}

// GetLOLLeaderboard returns LoL prediction leaderboard entries
func (s *userMetricsService) GetLOLLeaderboard(ctx context.Context, minWagers int) ([]*entities.LOLLeaderboardEntry, int64, error) {
	return s.getGameLeaderboard(ctx, minWagers, entities.SystemLeagueOfLegends)
}

// GetTFTLeaderboard returns TFT prediction leaderboard entries
func (s *userMetricsService) GetTFTLeaderboard(ctx context.Context, minWagers int) ([]*entities.LOLLeaderboardEntry, int64, error) {
	return s.getGameLeaderboard(ctx, minWagers, entities.SystemTFT)
}

// GetGamblingLeaderboard returns gambling leaderboard entries sorted by net profit
func (s *userMetricsService) GetGamblingLeaderboard(ctx context.Context, minBets int) ([]*entities.GamblingLeaderboardEntry, int64, error) {
	// Get all users
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all users: %w", err)
	}

	// Build leaderboard entries
	entries := make([]*entities.GamblingLeaderboardEntry, 0)
	var totalBitsWagered int64

	for _, user := range users {
		// Get bet stats for this user
		betStats, err := s.betRepo.GetStats(ctx, user.DiscordID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get bet stats for user %d: %w", user.DiscordID, err)
		}

		// Only include users with bets
		if betStats.TotalBets == 0 {
			continue
		}

		entry := &entities.GamblingLeaderboardEntry{
			DiscordID:    user.DiscordID,
			TotalBets:    betStats.TotalBets,
			TotalWins:    betStats.TotalWins,
			TotalWagered: betStats.TotalWagered,
			NetProfit:    betStats.TotalWon - betStats.TotalLost,
		}

		entry.CalculateWinPercentage()

		// Only include users who meet minimum bet requirement
		if entry.QualifiesForLeaderboard(minBets) {
			entries = append(entries, entry)
			totalBitsWagered += betStats.TotalWagered
		}
	}

	// Sort by net profit (descending), then by total bets (descending)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].NetProfit == entries[j].NetProfit {
			return entries[i].TotalBets > entries[j].TotalBets
		}
		return entries[i].NetProfit > entries[j].NetProfit
	})

	// Assign ranks
	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries, totalBitsWagered, nil
}