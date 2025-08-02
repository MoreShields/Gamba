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

		// Mock users
		users := []*entities.User{
			{DiscordID: 100, Username: "user1", Balance: 5000, AvailableBalance: 4000},
			{DiscordID: 200, Username: "user2", Balance: 3000, AvailableBalance: 3000},
			{DiscordID: 300, Username: "user3", Balance: 0, AvailableBalance: 0}, // Should be filtered
		}
		mockUserRepo.On("GetAll", ctx).Return(users, nil)

		// Mock active wagers
		activeWagers1 := []*entities.Wager{{ID: 1}, {ID: 2}}
		activeWagers2 := []*entities.Wager{{ID: 3}}
		mockWagerRepo.On("GetActiveByUser", ctx, int64(100)).Return(activeWagers1, nil)
		mockWagerRepo.On("GetActiveByUser", ctx, int64(200)).Return(activeWagers2, nil)
		// Note: user3 with 0 balance is skipped by GetScoreboard, so no calls expected

		// Mock wager stats
		wagerStats1 := &entities.WagerStats{TotalResolved: 10, TotalWon: 6}
		wagerStats2 := &entities.WagerStats{TotalResolved: 5, TotalWon: 2}
		mockWagerRepo.On("GetStats", ctx, int64(100)).Return(wagerStats1, nil)
		mockWagerRepo.On("GetStats", ctx, int64(200)).Return(wagerStats2, nil)
		// Note: user3 with 0 balance is skipped by GetScoreboard, so no calls expected

		// Mock bet stats
		betStats1 := &entities.BetStats{TotalBets: 20, TotalWins: 15}
		betStats2 := &entities.BetStats{TotalBets: 10, TotalWins: 3}
		mockBetRepo.On("GetStats", ctx, int64(100)).Return(betStats1, nil)
		mockBetRepo.On("GetStats", ctx, int64(200)).Return(betStats2, nil)
		// Note: user3 with 0 balance is skipped by GetScoreboard, so no calls expected

		// Mock volume for each user
		mockBalanceHistoryRepo.On("GetTotalVolumeByUser", ctx, int64(100)).Return(int64(10000), nil)
		mockBalanceHistoryRepo.On("GetTotalVolumeByUser", ctx, int64(200)).Return(int64(5000), nil)

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

		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, int64(200), entries[1].DiscordID)
		assert.Equal(t, int64(3000), entries[1].TotalBalance)
		assert.Equal(t, 1, entries[1].ActiveWagerCount)
		assert.Equal(t, float64(40), entries[1].WagerWinRate)
		assert.Equal(t, float64(30), entries[1].BetWinRate)
		assert.Equal(t, int64(5000), entries[1].TotalVolume)

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

		// Mock many users
		users := make([]*entities.User, 5)
		for i := 0; i < 5; i++ {
			users[i] = &entities.User{
				DiscordID:        int64(100 + i),
				Username:         fmt.Sprintf("user%d", i+1),
				Balance:          int64(5000 - i*1000),
				AvailableBalance: int64(5000 - i*1000),
			}
		}
		mockUserRepo.On("GetAll", ctx).Return(users, nil)

		// Mock empty stats for all users
		for i := 0; i < 5; i++ {
			mockWagerRepo.On("GetActiveByUser", ctx, int64(100+i)).Return([]*entities.Wager{}, nil)
			mockWagerRepo.On("GetStats", ctx, int64(100+i)).Return(&entities.WagerStats{}, nil)
			mockBetRepo.On("GetStats", ctx, int64(100+i)).Return(&entities.BetStats{}, nil)
			mockBalanceHistoryRepo.On("GetTotalVolumeByUser", ctx, int64(100+i)).Return(int64(0), nil)
		}

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