package service

import (
	"context"
	"fmt"
	"time"
)

// DailyAwardType represents different types of daily awards
type DailyAwardType string

const (
	DailyAwardTypeWordle DailyAwardType = "wordle"
	// Future award types can be added here
	// DailyAwardTypeStreak DailyAwardType = "streak"
	// DailyAwardTypeChallenge DailyAwardType = "challenge"
)

// DailyAward represents a generic daily award
type DailyAward interface {
	GetType() DailyAwardType
	GetDiscordID() int64
	GetReward() int64
	GetDetails() string
}

// WordleDailyAward represents a wordle completion award
type WordleDailyAward struct {
	DiscordID  int64
	GuessCount int
	Reward     int64
}

func (w WordleDailyAward) GetType() DailyAwardType {
	return DailyAwardTypeWordle
}

func (w WordleDailyAward) GetDiscordID() int64 {
	return w.DiscordID
}

func (w WordleDailyAward) GetReward() int64 {
	return w.Reward
}

func (w WordleDailyAward) GetDetails() string {
	return fmt.Sprintf("%d/6", w.GuessCount)
}

// DailyAwardsSummary represents all daily awards for a guild
type DailyAwardsSummary struct {
	GuildID    int64
	Date       time.Time
	Awards     []DailyAward
	TotalPayout int64
}

// DailyAwardsService handles aggregating and formatting daily awards
type DailyAwardsService struct {
	wordleRepo     WordleCompletionRepository
	userRepo       UserRepository
	rewardService  *WordleRewardService
}

// NewDailyAwardsService creates a new DailyAwardsService
func NewDailyAwardsService(
	wordleRepo WordleCompletionRepository,
	userRepo UserRepository,
	rewardService *WordleRewardService,
) *DailyAwardsService {
	return &DailyAwardsService{
		wordleRepo:    wordleRepo,
		userRepo:      userRepo,
		rewardService: rewardService,
	}
}

// GetDailyAwardsSummary retrieves all daily awards for a guild
func (s *DailyAwardsService) GetDailyAwardsSummary(ctx context.Context, guildID int64) (*DailyAwardsSummary, error) {
	summary := &DailyAwardsSummary{
		GuildID: guildID,
		Date:    time.Now().UTC().Truncate(24 * time.Hour),
		Awards:  []DailyAward{},
	}

	// Get wordle awards
	wordleAwards, err := s.getWordleAwards(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wordle awards: %w", err)
	}

	// Add wordle awards to summary
	for _, award := range wordleAwards {
		summary.Awards = append(summary.Awards, award)
		summary.TotalPayout += award.GetReward()
	}

	// Future: Add other daily award types here
	// streakAwards, err := s.getStreakAwards(ctx, guildID)
	// ...

	return summary, nil
}

// getWordleAwards retrieves today's wordle awards for a guild
func (s *DailyAwardsService) getWordleAwards(ctx context.Context, guildID int64) ([]DailyAward, error) {
	// Get all completions for today
	completions, err := s.wordleRepo.GetTodaysCompletions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's completions: %w", err)
	}

	// Build awards with reward calculations
	awards := make([]DailyAward, 0, len(completions))
	for _, completion := range completions {
		// Calculate reward including streak bonus
		reward, err := s.rewardService.CalculateReward(ctx, completion.DiscordID, guildID, completion.Score)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate reward for user %d: %w", completion.DiscordID, err)
		}

		awards = append(awards, WordleDailyAward{
			DiscordID:  completion.DiscordID,
			GuessCount: completion.Score.Guesses,
			Reward:     reward,
		})
	}

	return awards, nil
}