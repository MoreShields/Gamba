package models

import (
	"fmt"
	"time"
)

// WordleCompletion represents a completed Wordle puzzle
type WordleCompletion struct {
	ID          int64
	DiscordID   int64
	GuildID     int64
	Score       WordleScore
	CompletedAt time.Time
	CreatedAt   time.Time
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
