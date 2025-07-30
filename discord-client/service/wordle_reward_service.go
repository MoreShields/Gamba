package service

import (
	"context"
	"time"

	"gambler/discord-client/models"
)

// WordleRewardService calculates rewards for Wordle completions including streak bonuses
type WordleRewardService struct {
	repo       WordleCompletionRepository
	baseReward int64
}

// NewWordleRewardService creates a new WordleRewardService
func NewWordleRewardService(repo WordleCompletionRepository, baseReward int64) *WordleRewardService {
	return &WordleRewardService{
		repo:       repo,
		baseReward: baseReward,
	}
}

// CalculateReward calculates the total reward for a Wordle completion, including streak bonus
func (s *WordleRewardService) CalculateReward(ctx context.Context, discordID, guildID int64, score models.WordleScore) (int64, error) {
	// Calculate streak
	streakDays, err := s.countConsecutiveDays(ctx, discordID, guildID)
	if err != nil {
		return 0, err
	}

	// Special case: single guess always pays 50k regardless of streak
	if score.Guesses == 1 {
		return 50000, nil
	}

	// Calculate base reward based on guesses
	var baseReward int64
	switch score.Guesses {
	case 2:
		baseReward = 10000
	case 3, 4:
		baseReward = 7000
	case 5, 6:
		baseReward = 5000
	default:
		return 0, nil // Should not happen with valid WordleScore
	}

	// Apply streak multiplier (minimum 1, maximum 5)
	streakMultiplier := streakDays
	if streakMultiplier < 1 {
		streakMultiplier = 1
	} else if streakMultiplier > 5 {
		streakMultiplier = 5
	}

	totalReward := baseReward * int64(streakMultiplier)

	return totalReward, nil
}

// countConsecutiveDays returns the number of consecutive days including today
func (s *WordleRewardService) countConsecutiveDays(ctx context.Context, discordID, guildID int64) (int, error) {
	// Get recent completions - only need to check enough days to reach the cap
	// Since we cap at 5x multiplier, we only need to check up to 5 days
	// Adding a small buffer for edge cases
	// Note: Database constraint ensures only one completion per user per guild per day
	completions, err := s.repo.GetRecentCompletions(ctx, discordID, guildID, 7)
	if err != nil {
		return 0, err
	}

	if len(completions) == 0 {
		return 0, nil
	}

	// Start counting from today
	streak := 0
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i, completion := range completions {
		expectedDate := today.AddDate(0, 0, -i)
		completionDate := completion.CompletedAt.UTC().Truncate(24 * time.Hour)

		if completionDate.Equal(expectedDate) {
			streak++
		} else {
			// Gap found, stop counting
			break
		}
	}

	return streak, nil
}
