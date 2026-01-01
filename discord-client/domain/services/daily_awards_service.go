package services

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
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
	Streak     int
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

func (w WordleDailyAward) GetStreak() int {
	return w.Streak
}

// DailyAwardsSummary represents all daily awards for a guild
type DailyAwardsSummary struct {
	GuildID         int64
	Date            time.Time
	Awards          []DailyAward
	TotalPayout     int64
	TotalServerBits int64
}

// DailyAwardsService handles aggregating and formatting daily awards
type DailyAwardsService struct {
	wordleRepo     interfaces.WordleCompletionRepository
	userRepo       interfaces.UserRepository
	wagerRepo      interfaces.WagerRepository
	betRepo        interfaces.BetRepository
	groupWagerRepo interfaces.GroupWagerRepository
}

// NewDailyAwardsService creates a new DailyAwardsService
func NewDailyAwardsService(
	wordleRepo interfaces.WordleCompletionRepository,
	userRepo interfaces.UserRepository,
	wagerRepo interfaces.WagerRepository,
	betRepo interfaces.BetRepository,
	groupWagerRepo interfaces.GroupWagerRepository,
) *DailyAwardsService {
	return &DailyAwardsService{
		wordleRepo:     wordleRepo,
		userRepo:       userRepo,
		wagerRepo:      wagerRepo,
		betRepo:        betRepo,
		groupWagerRepo: groupWagerRepo,
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

	// Calculate total server bits by summing all user balances
	totalBits, err := s.calculateTotalServerBits(ctx)
	if err != nil {
		// Log error but don't fail the whole operation
		// Total server bits is nice to have but not critical
		totalBits = 0
	}
	summary.TotalServerBits = totalBits

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
		reward, err := s.CalculateWordleReward(ctx, s.wordleRepo, completion.DiscordID, guildID, completion.Score)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate reward for user %d: %w", completion.DiscordID, err)
		}

		// Get streak for this user
		streak, err := s.CountConsecutiveDays(ctx, s.wordleRepo, completion.DiscordID, guildID)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate streak for user %d: %w", completion.DiscordID, err)
		}

		awards = append(awards, WordleDailyAward{
			DiscordID:  completion.DiscordID,
			GuessCount: completion.Score.Guesses,
			Reward:     reward,
			Streak:     streak,
		})
	}

	return awards, nil
}

// calculateTotalServerBits sums up all user balances in the guild
func (s *DailyAwardsService) calculateTotalServerBits(ctx context.Context) (int64, error) {
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get all users: %w", err)
	}

	var totalBits int64
	for _, user := range users {
		totalBits += user.Balance
	}

	return totalBits, nil
}

// CalculateWordleReward calculates the total reward for a Wordle completion, including streak bonus
func (s *DailyAwardsService) CalculateWordleReward(ctx context.Context, repo interfaces.WordleCompletionRepository, discordID, guildID int64, score entities.WordleScore) (int64, error) {
	// Calculate streak
	streakDays, err := s.CountConsecutiveDays(ctx, repo, discordID, guildID)
	if err != nil {
		return 0, err
	}

	// Special case: single guess always pays 5 bits regardless of streak
	if score.Guesses == 1 {
		return 5, nil
	}

	// Calculate base reward based on guesses
	var baseReward int64
	switch score.Guesses {
	case 2, 3, 4:
		baseReward = 2
	case 5, 6:
		baseReward = 1
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

// CountConsecutiveDays returns the number of consecutive days including today
func (s *DailyAwardsService) CountConsecutiveDays(ctx context.Context, repo interfaces.WordleCompletionRepository, discordID, guildID int64) (int, error) {
	// Get all completions to count the true streak length
	// While the reward multiplier caps at 5x, we want to show the actual consecutive days
	// for user achievement tracking. The counting stops at the first gap anyway.
	// Note: Database constraint ensures only one completion per user per guild per day
	completions, err := repo.GetRecentCompletions(ctx, discordID, guildID, 0) // 0 = no limit
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