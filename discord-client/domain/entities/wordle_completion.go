package entities

import (
	"fmt"
	"time"
)

// WordleCompletion represents a completed Wordle puzzle
type WordleCompletion struct {
	ID          int64       `db:"id"`
	DiscordID   int64       `db:"discord_id"`
	GuildID     int64       `db:"guild_id"`
	Score       WordleScore `db:"score"`
	CompletedAt time.Time   `db:"completed_at"`
	CreatedAt   time.Time   `db:"created_at"`
}

// NewWordleCompletion creates a new WordleCompletion with validation
func NewWordleCompletion(discordID, guildID int64, score WordleScore, completedAt time.Time) (*WordleCompletion, error) {
	if discordID <= 0 {
		return nil, fmt.Errorf("discordID must be greater than 0, got %d", discordID)
	}
	if guildID <= 0 {
		return nil, fmt.Errorf("guildID must be greater than 0, got %d", guildID)
	}
	if completedAt.IsZero() {
		return nil, fmt.Errorf("completedAt cannot be zero time")
	}

	return &WordleCompletion{
		DiscordID:   discordID,
		GuildID:     guildID,
		Score:       score,
		CompletedAt: completedAt,
		CreatedAt:   time.Now(),
	}, nil
}

// CalculateReward calculates the reward points for this completion
func (wc *WordleCompletion) CalculateReward(baseReward int) int {
	return wc.Score.BasePoints(baseReward)
}

// IsToday checks if the completion was made today
func (wc *WordleCompletion) IsToday() bool {
	now := time.Now()
	return wc.CompletedAt.Year() == now.Year() &&
		wc.CompletedAt.YearDay() == now.YearDay()
}