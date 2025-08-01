package service

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWordleRewardService_CalculateReward(t *testing.T) {
	t.Parallel()

	baseReward := int64(1000)
	ctx := context.Background()

	tests := []struct {
		name           string
		score          models.WordleScore
		setupMock      func(repo *MockWordleCompletionRepository)
		expectedReward int64
		wantErr        bool
	}{
		{
			name:  "single guess always 50k",
			score: mustCreateScore(t, 1),
			setupMock: func(repo *MockWordleCompletionRepository) {
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return([]*models.WordleCompletion{}, nil)
			},
			expectedReward: 50000, // Always 50k for single guess
		},
		{
			name:  "single guess with 30 day streak still 50k",
			score: mustCreateScore(t, 1),
			setupMock: func(repo *MockWordleCompletionRepository) {
				today := time.Now().UTC().Truncate(24 * time.Hour)
				var completions []*models.WordleCompletion
				for i := 0; i < 30; i++ {
					completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
				}
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(completions, nil)
			},
			expectedReward: 50000, // Still 50k regardless of streak
		},
		{
			name:  "2 guesses no streak",
			score: mustCreateScore(t, 2),
			setupMock: func(repo *MockWordleCompletionRepository) {
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return([]*models.WordleCompletion{}, nil)
			},
			expectedReward: 10000, // 10k * 1
		},
		{
			name:  "2 guesses 7 day streak capped at 5x",
			score: mustCreateScore(t, 2),
			setupMock: func(repo *MockWordleCompletionRepository) {
				today := time.Now().UTC().Truncate(24 * time.Hour)
				var completions []*models.WordleCompletion
				for i := 0; i < 7; i++ {
					completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
				}
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(completions, nil)
			},
			expectedReward: 50000, // 10k * 5 (capped)
		},
		{
			name:  "3 guesses 3 day streak",
			score: mustCreateScore(t, 3),
			setupMock: func(repo *MockWordleCompletionRepository) {
				today := time.Now().UTC().Truncate(24 * time.Hour)
				completions := []*models.WordleCompletion{
					createCompletionForDate(t, today),
					createCompletionForDate(t, today.AddDate(0, 0, -1)),
					createCompletionForDate(t, today.AddDate(0, 0, -2)),
				}
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(completions, nil)
			},
			expectedReward: 21000, // 7k * 3
		},
		{
			name:  "4 guesses 14 day streak capped at 5x",
			score: mustCreateScore(t, 4),
			setupMock: func(repo *MockWordleCompletionRepository) {
				today := time.Now().UTC().Truncate(24 * time.Hour)
				var completions []*models.WordleCompletion
				// Repository would only return up to 7 completions
				for i := 0; i < 7; i++ {
					completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
				}
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(completions, nil)
			},
			expectedReward: 35000, // 7k * 5 (capped)
		},
		{
			name:  "5 guesses 30 day streak capped at 5x",
			score: mustCreateScore(t, 5),
			setupMock: func(repo *MockWordleCompletionRepository) {
				today := time.Now().UTC().Truncate(24 * time.Hour)
				var completions []*models.WordleCompletion
				// Repository would only return up to 7 completions
				for i := 0; i < 7; i++ {
					completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
				}
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(completions, nil)
			},
			expectedReward: 25000, // 5k * 5 (capped)
		},
		{
			name:  "6 guesses no streak",
			score: mustCreateScore(t, 6),
			setupMock: func(repo *MockWordleCompletionRepository) {
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return([]*models.WordleCompletion{}, nil)
			},
			expectedReward: 5000, // 5k * 1
		},
		{
			name:  "streak broken yesterday",
			score: mustCreateScore(t, 3),
			setupMock: func(repo *MockWordleCompletionRepository) {
				today := time.Now().UTC().Truncate(24 * time.Hour)
				completions := []*models.WordleCompletion{
					createCompletionForDate(t, today),
					// Gap here - no completion yesterday
					createCompletionForDate(t, today.AddDate(0, 0, -2)),
					createCompletionForDate(t, today.AddDate(0, 0, -3)),
				}
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(completions, nil)
			},
			expectedReward: 7000, // 7k * 1 (only 1 day streak)
		},
		{
			name:  "repository error",
			score: mustCreateScore(t, 3),
			setupMock: func(repo *MockWordleCompletionRepository) {
				repo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
					Return(nil, assert.AnError)
			},
			expectedReward: 0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockWordleCompletionRepository)
			tt.setupMock(mockRepo)

			service := NewWordleRewardService(mockRepo, baseReward)
			reward, err := service.CalculateReward(ctx, 123, 456, tt.score)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedReward, reward)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestWordleRewardService_StreakEdgeCases(t *testing.T) {
	t.Parallel()

	baseReward := int64(100)
	ctx := context.Background()

	t.Run("completions not in consecutive order", func(t *testing.T) {
		mockRepo := new(MockWordleCompletionRepository)
		today := time.Now().UTC().Truncate(24 * time.Hour)

		// Completions returned in wrong order
		completions := []*models.WordleCompletion{
			createCompletionForDate(t, today.AddDate(0, 0, -2)),
			createCompletionForDate(t, today),
			createCompletionForDate(t, today.AddDate(0, 0, -1)),
		}

		mockRepo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
			Return(completions, nil)

		service := NewWordleRewardService(mockRepo, baseReward)
		score, _ := models.NewWordleScore(3)
		reward, err := service.CalculateReward(ctx, 123, 456, score)

		require.NoError(t, err)
		// Should only count today (streak of 1)
		assert.Equal(t, int64(7000), reward) // 7k * 1
	})

t.Run("future dated completion", func(t *testing.T) {
		mockRepo := new(MockWordleCompletionRepository)
		today := time.Now().UTC().Truncate(24 * time.Hour)

		completions := []*models.WordleCompletion{
			createCompletionForDate(t, today.AddDate(0, 0, 1)), // Tomorrow
			createCompletionForDate(t, today),
		}

		mockRepo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
			Return(completions, nil)

		service := NewWordleRewardService(mockRepo, baseReward)
		score, _ := models.NewWordleScore(3)
		reward, err := service.CalculateReward(ctx, 123, 456, score)

		require.NoError(t, err)
		// Should ignore future completion and count only today (streak of 1)
		assert.Equal(t, int64(7000), reward) // 7k * 1
	})
}

func TestWordleRewardService_SpecialCases(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("single guess always returns 50k", func(t *testing.T) {
		mockRepo := new(MockWordleCompletionRepository)
		
		// Even with a long streak (which would be capped at 5x for other scores)
		today := time.Now().UTC().Truncate(24 * time.Hour)
		var completions []*models.WordleCompletion
		for i := 0; i < 7; i++ {
			completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
		}
		
		mockRepo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
			Return(completions, nil)

		// Note: baseReward parameter is ignored for single guess
		service := NewWordleRewardService(mockRepo, 999999)
		score, _ := models.NewWordleScore(1)
		reward, err := service.CalculateReward(ctx, 123, 456, score)

		require.NoError(t, err)
		assert.Equal(t, int64(50000), reward)
	})

	t.Run("zero streak counts as 1", func(t *testing.T) {
		mockRepo := new(MockWordleCompletionRepository)
		
		// No completions in history
		mockRepo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
			Return([]*models.WordleCompletion{}, nil)

		service := NewWordleRewardService(mockRepo, 1000)
		score, _ := models.NewWordleScore(2)
		reward, err := service.CalculateReward(ctx, 123, 456, score)

		require.NoError(t, err)
		assert.Equal(t, int64(10000), reward) // 10k * 1
	})

	t.Run("exactly 5 day streak", func(t *testing.T) {
		mockRepo := new(MockWordleCompletionRepository)
		
		today := time.Now().UTC().Truncate(24 * time.Hour)
		var completions []*models.WordleCompletion
		for i := 0; i < 5; i++ {
			completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
		}
		
		mockRepo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
			Return(completions, nil)

		service := NewWordleRewardService(mockRepo, 1000)
		score, _ := models.NewWordleScore(3)
		reward, err := service.CalculateReward(ctx, 123, 456, score)

		require.NoError(t, err)
		assert.Equal(t, int64(35000), reward) // 7k * 5
	})

	t.Run("6 day streak still capped at 5x", func(t *testing.T) {
		mockRepo := new(MockWordleCompletionRepository)
		
		today := time.Now().UTC().Truncate(24 * time.Hour)
		var completions []*models.WordleCompletion
		for i := 0; i < 6; i++ {
			completions = append(completions, createCompletionForDate(t, today.AddDate(0, 0, -i)))
		}
		
		mockRepo.On("GetRecentCompletions", ctx, int64(123), int64(456), 7).
			Return(completions, nil)

		service := NewWordleRewardService(mockRepo, 1000)
		score, _ := models.NewWordleScore(3)
		reward, err := service.CalculateReward(ctx, 123, 456, score)

		require.NoError(t, err)
		assert.Equal(t, int64(35000), reward) // 7k * 5 (capped)
	})
}

// Helper functions

func mustCreateScore(t *testing.T, guesses int) models.WordleScore {
	score, err := models.NewWordleScore(guesses)
	require.NoError(t, err)
	return score
}

func createCompletionForDate(t *testing.T, date time.Time) *models.WordleCompletion {
	score, _ := models.NewWordleScore(3)
	completion, err := models.NewWordleCompletion(123, 456, score, date)
	require.NoError(t, err)
	return completion
}