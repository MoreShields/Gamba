package services

import (
	"gambler/discord-client/domain/testhelpers"
	"context"
	"fmt"
	"testing"

	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a pointer to an int64
func ptr(i int64) *int64 {
	return &i
}

func TestUserMetricsService_GetLOLPredictionStats(t *testing.T) {
	ctx := context.Background()

	t.Run("calculates accuracy for LOL predictions correctly", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Mock data: 3 users with different prediction patterns
		predictions := []*entities.GroupWagerPrediction{
			// User 1: 2/3 correct (66.67%)
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "Win", WinningOptionID: 1, Amount: 1000, WasCorrect: true},
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "Loss", WinningOptionID: 3, Amount: 500, WasCorrect: true},
			{DiscordID: 100, GroupWagerID: 3, OptionID: 5, OptionText: "Win", WinningOptionID: 6, Amount: 750, WasCorrect: false},
			
			// User 2: 1/2 correct (50%)
			{DiscordID: 200, GroupWagerID: 1, OptionID: 2, OptionText: "Loss", WinningOptionID: 1, Amount: 2000, WasCorrect: false},
			{DiscordID: 200, GroupWagerID: 2, OptionID: 3, OptionText: "Loss", WinningOptionID: 3, Amount: 1500, WasCorrect: true},
			
			// User 3: 3/3 correct (100%)
			{DiscordID: 300, GroupWagerID: 1, OptionID: 1, OptionText: "Win", WinningOptionID: 1, Amount: 500, WasCorrect: true},
			{DiscordID: 300, GroupWagerID: 2, OptionID: 3, OptionText: "Loss", WinningOptionID: 3, Amount: 500, WasCorrect: true},
			{DiscordID: 300, GroupWagerID: 3, OptionID: 6, OptionText: "Loss", WinningOptionID: 6, Amount: 500, WasCorrect: true},
		}

		lolSystem := entities.SystemLeagueOfLegends
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &lolSystem).Return(predictions, nil)

		// Execute
		stats, err := service.GetLOLPredictionStats(ctx)

		// Assert
		require.NoError(t, err)
		require.Len(t, stats, 3)

		// User 1 stats
		user1Stats := stats[100]
		assert.Equal(t, int64(100), user1Stats.DiscordID)
		assert.Equal(t, 3, user1Stats.TotalPredictions)
		assert.Equal(t, 2, user1Stats.CorrectPredictions)
		assert.InDelta(t, 66.67, user1Stats.AccuracyPercentage, 0.01)
		assert.Equal(t, int64(2250), user1Stats.TotalAmountWagered)

		// User 2 stats
		user2Stats := stats[200]
		assert.Equal(t, int64(200), user2Stats.DiscordID)
		assert.Equal(t, 2, user2Stats.TotalPredictions)
		assert.Equal(t, 1, user2Stats.CorrectPredictions)
		assert.Equal(t, float64(50), user2Stats.AccuracyPercentage)
		assert.Equal(t, int64(3500), user2Stats.TotalAmountWagered)

		// User 3 stats
		user3Stats := stats[300]
		assert.Equal(t, int64(300), user3Stats.DiscordID)
		assert.Equal(t, 3, user3Stats.TotalPredictions)
		assert.Equal(t, 3, user3Stats.CorrectPredictions)
		assert.Equal(t, float64(100), user3Stats.AccuracyPercentage)
		assert.Equal(t, int64(1500), user3Stats.TotalAmountWagered)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("filters out non-Win/Loss options", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Include some non-Win/Loss options that should be filtered
		predictions := []*entities.GroupWagerPrediction{
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "Win", WinningOptionID: 1, Amount: 1000, WasCorrect: true},
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "Draw", WinningOptionID: 3, Amount: 500, WasCorrect: true}, // Should be filtered
			{DiscordID: 100, GroupWagerID: 3, OptionID: 5, OptionText: "Loss", WinningOptionID: 6, Amount: 750, WasCorrect: false},
			{DiscordID: 100, GroupWagerID: 4, OptionID: 7, OptionText: "Remake", WinningOptionID: 7, Amount: 250, WasCorrect: true}, // Should be filtered
		}

		lolSystem := entities.SystemLeagueOfLegends
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &lolSystem).Return(predictions, nil)

		// Execute
		stats, err := service.GetLOLPredictionStats(ctx)

		// Assert
		require.NoError(t, err)
		require.Len(t, stats, 1)

		userStats := stats[100]
		assert.Equal(t, 2, userStats.TotalPredictions) // Only Win and Loss counted
		assert.Equal(t, 1, userStats.CorrectPredictions)
		assert.Equal(t, float64(50), userStats.AccuracyPercentage)
		assert.Equal(t, int64(1750), userStats.TotalAmountWagered) // Only Win and Loss amounts

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		expectedErr := fmt.Errorf("database error")
		lolSystem := entities.SystemLeagueOfLegends
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &lolSystem).Return(nil, expectedErr)

		// Execute
		stats, err := service.GetLOLPredictionStats(ctx)

		// Assert
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "failed to get LOL wager predictions")
		assert.Contains(t, err.Error(), "database error")

		mockGroupWagerRepo.AssertExpectations(t)
	})
}

func TestUserMetricsService_GetWagerPredictionStats(t *testing.T) {
	ctx := context.Background()

	t.Run("calculates stats for all wagers when no filter", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		predictions := []*entities.GroupWagerPrediction{
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "Option A", WinningOptionID: 1, Amount: 1000, WasCorrect: true},
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "Option B", WinningOptionID: 4, Amount: 500, WasCorrect: false},
			{DiscordID: 200, GroupWagerID: 1, OptionID: 2, OptionText: "Option B", WinningOptionID: 1, Amount: 2000, WasCorrect: false},
		}

		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, (*entities.ExternalSystem)(nil)).Return(predictions, nil)

		// Execute
		stats, err := service.GetWagerPredictionStats(ctx, nil)

		// Assert
		require.NoError(t, err)
		require.Len(t, stats, 2)

		user1Stats := stats[100]
		assert.Equal(t, 2, user1Stats.TotalPredictions)
		assert.Equal(t, 1, user1Stats.CorrectPredictions)
		assert.Equal(t, float64(50), user1Stats.AccuracyPercentage)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("filters by external system when specified", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		tftSystem := entities.SystemTFT
		predictions := []*entities.GroupWagerPrediction{
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "Top 4", WinningOptionID: 1, Amount: 1000, WasCorrect: true, ExternalSystem: &tftSystem},
		}

		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute
		stats, err := service.GetWagerPredictionStats(ctx, &tftSystem)

		// Assert
		require.NoError(t, err)
		require.Len(t, stats, 1)

		mockGroupWagerRepo.AssertExpectations(t)
	})
}

func TestUserMetricsService_GetScoreboard(t *testing.T) {
	ctx := context.Background()

	t.Run("returns scoreboard with correct statistics", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Mock scoreboard data from the optimized query
		scoreboardEntries := []*entities.ScoreboardEntry{
			{
				Rank:             1,
				DiscordID:        100,
				Username:         "user1",
				TotalBalance:     5000,
				AvailableBalance: 4000,
				ActiveWagerCount: 2,
				WagerWinRate:     60,
				BetWinRate:       75,
				TotalVolume:      10000,
				TotalDonations:   2000,
			},
			{
				Rank:             2,
				DiscordID:        200,
				Username:         "user2",
				TotalBalance:     3000,
				AvailableBalance: 3000,
				ActiveWagerCount: 1,
				WagerWinRate:     40,
				BetWinRate:       30,
				TotalVolume:      5000,
				TotalDonations:   1000,
			},
		}
		// Mock GetScoreboardData to return entries and total
		mockUserRepo.On("GetScoreboardData", ctx).Return(scoreboardEntries, int64(8000), nil)

		// Execute
		entries, totalBits, err := service.GetScoreboard(ctx, 10)

		// Assert
		require.NoError(t, err)
		assert.Len(t, entries, 2) // user3 filtered out
		assert.Equal(t, int64(8000), totalBits) // 5000 + 3000 + 0

		// Check rankings
		assert.Equal(t, 1, entries[0].Rank)
		assert.Equal(t, int64(100), entries[0].DiscordID)
		assert.Equal(t, int64(5000), entries[0].TotalBalance)
		assert.Equal(t, 2, entries[0].ActiveWagerCount)
		assert.Equal(t, float64(60), entries[0].WagerWinRate)
		assert.Equal(t, float64(75), entries[0].BetWinRate)
		assert.Equal(t, int64(10000), entries[0].TotalVolume)
		assert.Equal(t, int64(2000), entries[0].TotalDonations)

		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, int64(200), entries[1].DiscordID)
		assert.Equal(t, int64(3000), entries[1].TotalBalance)
		assert.Equal(t, 1, entries[1].ActiveWagerCount)
		assert.Equal(t, float64(40), entries[1].WagerWinRate)
		assert.Equal(t, float64(30), entries[1].BetWinRate)
		assert.Equal(t, int64(5000), entries[1].TotalVolume)
		assert.Equal(t, int64(1000), entries[1].TotalDonations)

		mockUserRepo.AssertExpectations(t)
		mockWagerRepo.AssertExpectations(t)
		mockBetRepo.AssertExpectations(t)
	})

	t.Run("applies limit correctly", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Mock scoreboard entries (5 users)
		scoreboardEntries := make([]*entities.ScoreboardEntry, 5)
		for i := 0; i < 5; i++ {
			scoreboardEntries[i] = &entities.ScoreboardEntry{
				Rank:             i + 1,
				DiscordID:        int64(100 + i),
				Username:         fmt.Sprintf("user%d", i+1),
				TotalBalance:     int64(5000 - i*1000),
				AvailableBalance: int64(5000 - i*1000),
				ActiveWagerCount: 0,
				WagerWinRate:     0,
				BetWinRate:       0,
				TotalVolume:      0,
				TotalDonations:   0,
			}
		}
		// Mock GetScoreboardData to return entries and total
		var totalBits int64
		for _, entry := range scoreboardEntries {
			totalBits += entry.TotalBalance
		}
		mockUserRepo.On("GetScoreboardData", ctx).Return(scoreboardEntries, totalBits, nil)

		// Execute with limit
		entries, _, err := service.GetScoreboard(ctx, 3)

		// Assert
		require.NoError(t, err)
		assert.Len(t, entries, 3) // Limited to 3

		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserMetricsService_GetUserStats(t *testing.T) {
	ctx := context.Background()

	t.Run("returns complete user stats", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Mock user
		user := &entities.User{
			DiscordID:        100,
			Username:         "testuser",
			Balance:          5000,
			AvailableBalance: 4000,
		}
		mockUserRepo.On("GetByDiscordID", ctx, int64(100)).Return(user, nil)

		// Mock bet stats
		betStats := &entities.BetStats{
			TotalBets:   20,
			TotalWins:   15,
			TotalLosses: 5,
			TotalWagered: 10000,
			TotalWon:     12000,
			TotalLost:    2000,
			BiggestWin:   3000,
			BiggestLoss:  500,
		}
		mockBetRepo.On("GetStats", ctx, int64(100)).Return(betStats, nil)

		// Mock wager stats
		wagerStats := &entities.WagerStats{
			TotalWagers:    30,
			TotalProposed:  15,
			TotalAccepted:  10,
			TotalDeclined:  5,
			TotalResolved:  10,
			TotalWon:       6,
			TotalLost:      4,
			TotalAmount:    8000,
			TotalWonAmount: 9000,
			BiggestWin:     2000,
			BiggestLoss:    1000,
		}
		mockWagerRepo.On("GetStats", ctx, int64(100)).Return(wagerStats, nil)

		// Mock group wager stats
		groupWagerStats := &entities.GroupWagerStats{
			TotalGroupWagers: 5,
			TotalProposed:    2,
			TotalWon:         3,
			TotalWonAmount:   1500,
		}
		mockGroupWagerRepo.On("GetStats", ctx, int64(100)).Return(groupWagerStats, nil)

		// Execute
		stats, err := service.GetUserStats(ctx, 100)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, user, stats.User)
		assert.Equal(t, int64(1000), stats.ReservedInWagers) // 5000 - 4000

		// Check bet stats
		assert.Equal(t, 20, stats.BetStats.TotalBets)
		assert.Equal(t, float64(75), stats.BetStats.WinPercentage)
		assert.Equal(t, int64(10000), stats.BetStats.NetProfit) // 12000 - 2000

		// Check wager stats
		assert.Equal(t, 30, stats.WagerStats.TotalWagers)
		assert.Equal(t, float64(60), stats.WagerStats.WinPercentage)

		// Check group wager stats
		assert.Equal(t, groupWagerStats, stats.GroupWagerStats)

		mockUserRepo.AssertExpectations(t)
		mockWagerRepo.AssertExpectations(t)
		mockBetRepo.AssertExpectations(t)
		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		mockUserRepo.On("GetByDiscordID", ctx, int64(999)).Return(nil, nil)

		// Execute
		stats, err := service.GetUserStats(ctx, 999)

		// Assert
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "user not found")

		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserMetricsService_GetTFTLeaderboard(t *testing.T) {
	ctx := context.Background()

	t.Run("calculates TFT leaderboard with placement options and 4:1 odds", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Mock TFT predictions with placement options and 4:1 odds (4x payout)
		predictions := []*entities.GroupWagerPrediction{
			// User 1: Mixed results - net loss
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 1000, WasCorrect: true, PayoutAmount: ptr(4000)},  // +3000 profit
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 5, Amount: 500, WasCorrect: false, PayoutAmount: ptr(0)}, // -500 loss
			{DiscordID: 100, GroupWagerID: 3, OptionID: 5, OptionText: "5-6", WinningOptionID: 7, Amount: 750, WasCorrect: false, PayoutAmount: ptr(0)}, // -750 loss
			{DiscordID: 100, GroupWagerID: 4, OptionID: 7, OptionText: "7-8", WinningOptionID: 1, Amount: 800, WasCorrect: false, PayoutAmount: ptr(0)}, // -800 loss
			// Net: +3000 - 500 - 750 - 800 = +950

			// User 2: Better performance - net gain
			{DiscordID: 200, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 2000, WasCorrect: true, PayoutAmount: ptr(8000)}, // +6000 profit
			{DiscordID: 200, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 3, Amount: 1500, WasCorrect: true, PayoutAmount: ptr(6000)}, // +4500 profit
			{DiscordID: 200, GroupWagerID: 3, OptionID: 5, OptionText: "5-6", WinningOptionID: 7, Amount: 1000, WasCorrect: false, PayoutAmount: ptr(0)}, // -1000 loss
			// Net: +6000 + 4500 - 1000 = +9500

			// User 3: Perfect record but fewer predictions
			{DiscordID: 300, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 500, WasCorrect: true, PayoutAmount: ptr(2000)}, // +1500 profit
			{DiscordID: 300, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 3, Amount: 500, WasCorrect: true, PayoutAmount: ptr(2000)}, // +1500 profit
			// Net: +1500 + 1500 = +3000
		}

		tftSystem := entities.SystemTFT
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute
		entries, totalBits, err := service.GetTFTLeaderboard(ctx, 1)

		// Assert
		require.NoError(t, err)
		require.Len(t, entries, 3)
		assert.Equal(t, int64(8550), totalBits) // User1: 3050 + User2: 4500 + User3: 1000 = 8550

		// Check sorting by profit/loss (descending)
		assert.Equal(t, 1, entries[0].Rank)
		assert.Equal(t, int64(200), entries[0].DiscordID)
		assert.Equal(t, int64(9500), entries[0].ProfitLoss)
		assert.Equal(t, 3, entries[0].TotalPredictions)
		assert.Equal(t, 2, entries[0].CorrectPredictions)
		assert.InDelta(t, 66.67, entries[0].AccuracyPercentage, 0.01)
		assert.Equal(t, int64(4500), entries[0].TotalAmountWagered)

		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, int64(300), entries[1].DiscordID)
		assert.Equal(t, int64(3000), entries[1].ProfitLoss)
		assert.Equal(t, 2, entries[1].TotalPredictions)
		assert.Equal(t, 2, entries[1].CorrectPredictions)
		assert.Equal(t, float64(100), entries[1].AccuracyPercentage)
		assert.Equal(t, int64(1000), entries[1].TotalAmountWagered)

		assert.Equal(t, 3, entries[2].Rank)
		assert.Equal(t, int64(100), entries[2].DiscordID)
		assert.Equal(t, int64(950), entries[2].ProfitLoss)
		assert.Equal(t, 4, entries[2].TotalPredictions)
		assert.Equal(t, 1, entries[2].CorrectPredictions)
		assert.Equal(t, float64(25), entries[2].AccuracyPercentage)
		assert.Equal(t, int64(3050), entries[2].TotalAmountWagered)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("handles null payout amounts gracefully", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Include predictions with various payout states
		predictions := []*entities.GroupWagerPrediction{
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 1000, WasCorrect: true, PayoutAmount: ptr(4000)}, // Normal winner
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 5, Amount: 500, WasCorrect: false, PayoutAmount: ptr(0)},    // Normal loser
			{DiscordID: 100, GroupWagerID: 3, OptionID: 5, OptionText: "5-6", WinningOptionID: 5, Amount: 750, WasCorrect: true, PayoutAmount: nil},        // Null payout (treated as 0)
			{DiscordID: 100, GroupWagerID: 4, OptionID: 7, OptionText: "7-8", WinningOptionID: 9, Amount: 250, WasCorrect: false, PayoutAmount: nil},       // Null payout loser
		}

		tftSystem := entities.SystemTFT
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute
		entries, totalBits, err := service.GetTFTLeaderboard(ctx, 1)

		// Assert
		require.NoError(t, err)
		require.Len(t, entries, 1)

		userStats := entries[0]
		assert.Equal(t, int64(100), userStats.DiscordID)
		assert.Equal(t, 4, userStats.TotalPredictions)
		assert.Equal(t, 2, userStats.CorrectPredictions)
		assert.Equal(t, float64(50), userStats.AccuracyPercentage)
		assert.Equal(t, int64(2500), userStats.TotalAmountWagered)
		// Profit: (4000-1000) + (0-500) + (0-750) + (0-250) = 3000 - 1500 = 1500
		assert.Equal(t, int64(1500), userStats.ProfitLoss)
		assert.Equal(t, int64(2500), totalBits)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("applies minimum wager requirement correctly", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		predictions := []*entities.GroupWagerPrediction{
			// User with 3 predictions - should qualify for minWagers=3
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 1000, WasCorrect: true},
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 5, Amount: 500, WasCorrect: false},
			{DiscordID: 100, GroupWagerID: 3, OptionID: 5, OptionText: "5-6", WinningOptionID: 5, Amount: 750, WasCorrect: true},

			// User with only 2 predictions - should NOT qualify for minWagers=3
			{DiscordID: 200, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 2000, WasCorrect: true},
			{DiscordID: 200, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 3, Amount: 1500, WasCorrect: true},
		}

		tftSystem := entities.SystemTFT
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute with minimum 3 wagers
		entries, totalBits, err := service.GetTFTLeaderboard(ctx, 3)

		// Assert
		require.NoError(t, err)
		require.Len(t, entries, 1) // Only user 100 qualifies
		assert.Equal(t, int64(5750), totalBits) // All predictions counted in total

		userStats := entries[0]
		assert.Equal(t, int64(100), userStats.DiscordID)
		assert.Equal(t, 3, userStats.TotalPredictions)
		assert.Equal(t, 2, userStats.CorrectPredictions)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("handles tiebreaker logic - same profit/loss sorted by total predictions", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		predictions := []*entities.GroupWagerPrediction{
			// User 1: 1 correct prediction = +2000 profit/loss
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 1000, WasCorrect: true, PayoutAmount: ptr(4000)},
			{DiscordID: 100, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 5, Amount: 1000, WasCorrect: false, PayoutAmount: ptr(0)},

			// User 2: 1 correct prediction = +2000 profit/loss but more predictions
			{DiscordID: 200, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 1000, WasCorrect: true, PayoutAmount: ptr(4000)},
			{DiscordID: 200, GroupWagerID: 2, OptionID: 3, OptionText: "3-4", WinningOptionID: 5, Amount: 500, WasCorrect: false, PayoutAmount: ptr(0)},
			{DiscordID: 200, GroupWagerID: 3, OptionID: 5, OptionText: "5-6", WinningOptionID: 7, Amount: 500, WasCorrect: false, PayoutAmount: ptr(0)},
		}

		tftSystem := entities.SystemTFT
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute
		entries, _, err := service.GetTFTLeaderboard(ctx, 1)

		// Assert
		require.NoError(t, err)
		require.Len(t, entries, 2)

		// User 200 should be ranked higher due to more total predictions (tiebreaker)
		assert.Equal(t, 1, entries[0].Rank)
		assert.Equal(t, int64(200), entries[0].DiscordID)
		assert.Equal(t, int64(2000), entries[0].ProfitLoss) // (4000-1000) - 500 - 500
		assert.Equal(t, 3, entries[0].TotalPredictions)

		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, int64(100), entries[1].DiscordID)
		assert.Equal(t, int64(2000), entries[1].ProfitLoss) // (4000-1000) - 1000
		assert.Equal(t, 2, entries[1].TotalPredictions)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("handles empty predictions", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// No predictions
		var predictions []*entities.GroupWagerPrediction

		tftSystem := entities.SystemTFT
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute
		entries, totalBits, err := service.GetTFTLeaderboard(ctx, 1)

		// Assert
		require.NoError(t, err)
		assert.Len(t, entries, 0)
		assert.Equal(t, int64(0), totalBits)

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		expectedErr := fmt.Errorf("database connection failed")
		tftSystem := entities.SystemTFT
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(nil, expectedErr)

		// Execute
		entries, totalBits, err := service.GetTFTLeaderboard(ctx, 1)

		// Assert
		require.Error(t, err)
		assert.Nil(t, entries)
		assert.Equal(t, int64(0), totalBits)
		assert.Contains(t, err.Error(), "failed to get teamfight_tactics wager predictions")
		assert.Contains(t, err.Error(), "database connection failed")

		mockGroupWagerRepo.AssertExpectations(t)
	})

	t.Run("verifies SystemTFT is passed correctly to repository", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		predictions := []*entities.GroupWagerPrediction{
			{DiscordID: 100, GroupWagerID: 1, OptionID: 1, OptionText: "1-2", WinningOptionID: 1, Amount: 1000, WasCorrect: true, PayoutAmount: ptr(4000)},
		}

		tftSystem := entities.SystemTFT
		// Verify that SystemTFT constant is passed exactly
		mockGroupWagerRepo.On("GetGroupWagerPredictions", ctx, &tftSystem).Return(predictions, nil)

		// Execute
		_, _, err := service.GetTFTLeaderboard(ctx, 1)

		// Assert
		require.NoError(t, err)
		mockGroupWagerRepo.AssertExpectations(t)
	})
}

func TestUserMetricsService_GetGamblingLeaderboard(t *testing.T) {
	ctx := context.Background()

	t.Run("calculates gambling leaderboard with correct net profit", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		// Mock users
		users := []*entities.User{
			{DiscordID: 100, Username: "user1", Balance: 5000},
			{DiscordID: 200, Username: "user2", Balance: 3000},
			{DiscordID: 300, Username: "user3", Balance: 2000},
		}
		mockUserRepo.On("GetAll", ctx).Return(users, nil)

		// Mock bet stats for each user
		mockBetRepo.On("GetStats", ctx, int64(100)).Return(&entities.BetStats{
			TotalBets:    20,
			TotalWins:    15,
			TotalLosses:  5,
			TotalWagered: 10000,
			TotalWon:     12000,
			TotalLost:    2000,
			BiggestWin:   3000,
			BiggestLoss:  500,
		}, nil)

		mockBetRepo.On("GetStats", ctx, int64(200)).Return(&entities.BetStats{
			TotalBets:    10,
			TotalWins:    4,
			TotalLosses:  6,
			TotalWagered: 5000,
			TotalWon:     5500,
			TotalLost:    3000,
			BiggestWin:   1000,
			BiggestLoss:  800,
		}, nil)

		mockBetRepo.On("GetStats", ctx, int64(300)).Return(&entities.BetStats{
			TotalBets:    8,
			TotalWins:    3,
			TotalLosses:  5,
			TotalWagered: 4000,
			TotalWon:     4200,
			TotalLost:    2500,
			BiggestWin:   800,
			BiggestLoss:  600,
		}, nil)

		// Execute
		entries, totalBitsWagered, err := service.GetGamblingLeaderboard(ctx, 5)

		// Assert
		require.NoError(t, err)
		require.Len(t, entries, 3)
		assert.Equal(t, int64(19000), totalBitsWagered)

		// Check ranking by net profit
		// User 1: 12000 - 2000 = 10000 profit
		// User 2: 5500 - 3000 = 2500 profit
		// User 3: 4200 - 2500 = 1700 profit
		assert.Equal(t, 1, entries[0].Rank)
		assert.Equal(t, int64(100), entries[0].DiscordID)
		assert.Equal(t, 20, entries[0].TotalBets)
		assert.Equal(t, 15, entries[0].TotalWins)
		assert.Equal(t, float64(75), entries[0].WinPercentage)
		assert.Equal(t, int64(10000), entries[0].NetProfit)

		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, int64(200), entries[1].DiscordID)
		assert.Equal(t, 10, entries[1].TotalBets)
		assert.Equal(t, 4, entries[1].TotalWins)
		assert.Equal(t, float64(40), entries[1].WinPercentage)
		assert.Equal(t, int64(2500), entries[1].NetProfit)

		assert.Equal(t, 3, entries[2].Rank)
		assert.Equal(t, int64(300), entries[2].DiscordID)
		assert.Equal(t, 8, entries[2].TotalBets)
		assert.Equal(t, 3, entries[2].TotalWins)
		assert.InDelta(t, 37.5, entries[2].WinPercentage, 0.01)
		assert.Equal(t, int64(1700), entries[2].NetProfit)

		mockUserRepo.AssertExpectations(t)
		mockBetRepo.AssertExpectations(t)
	})

	t.Run("filters by minimum bets requirement", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		users := []*entities.User{
			{DiscordID: 100, Username: "user1", Balance: 5000},
			{DiscordID: 200, Username: "user2", Balance: 3000},
			{DiscordID: 300, Username: "user3", Balance: 2000},
		}
		mockUserRepo.On("GetAll", ctx).Return(users, nil)

		// User 1: 20 bets (qualifies for minBets=5)
		mockBetRepo.On("GetStats", ctx, int64(100)).Return(&entities.BetStats{
			TotalBets:    20,
			TotalWins:    15,
			TotalWagered: 10000,
			TotalWon:     12000,
			TotalLost:    2000,
		}, nil)

		// User 2: 2 bets (doesn't qualify for minBets=5)
		mockBetRepo.On("GetStats", ctx, int64(200)).Return(&entities.BetStats{
			TotalBets:    2,
			TotalWins:    1,
			TotalWagered: 1000,
			TotalWon:     1500,
			TotalLost:    500,
		}, nil)

		// User 3: 0 bets (doesn't qualify - no bets at all)
		mockBetRepo.On("GetStats", ctx, int64(300)).Return(&entities.BetStats{
			TotalBets:    0,
			TotalWins:    0,
			TotalWagered: 0,
			TotalWon:     0,
			TotalLost:    0,
		}, nil)

		// Execute with minBets=5
		entries, totalBitsWagered, err := service.GetGamblingLeaderboard(ctx, 5)

		// Assert
		require.NoError(t, err)
		require.Len(t, entries, 1) // Only user 1 qualifies
		assert.Equal(t, int64(10000), totalBitsWagered)
		assert.Equal(t, int64(100), entries[0].DiscordID)

		mockUserRepo.AssertExpectations(t)
		mockBetRepo.AssertExpectations(t)
	})

	t.Run("handles empty user list", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		var users []*entities.User
		mockUserRepo.On("GetAll", ctx).Return(users, nil)

		// Execute
		entries, totalBitsWagered, err := service.GetGamblingLeaderboard(ctx, 5)

		// Assert
		require.NoError(t, err)
		assert.Len(t, entries, 0)
		assert.Equal(t, int64(0), totalBitsWagered)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("handles repository error from GetAll", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockWagerRepo := new(testhelpers.MockWagerRepository)
		mockBetRepo := new(testhelpers.MockBetRepository)
		mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
		mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
		service := NewUserMetricsService(mockUserRepo, mockWagerRepo, mockBetRepo, mockGroupWagerRepo, mockBalanceHistoryRepo)

		expectedErr := fmt.Errorf("database connection failed")
		mockUserRepo.On("GetAll", ctx).Return(nil, expectedErr)

		// Execute
		entries, totalBitsWagered, err := service.GetGamblingLeaderboard(ctx, 5)

		// Assert
		require.Error(t, err)
		assert.Nil(t, entries)
		assert.Equal(t, int64(0), totalBitsWagered)
		assert.Contains(t, err.Error(), "failed to get all users")
		assert.Contains(t, err.Error(), "database connection failed")

		mockUserRepo.AssertExpectations(t)
	})
}