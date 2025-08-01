package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWordleCompletion(t *testing.T) {
	validScore, _ := NewWordleScore(3)
	validTime := time.Now()

	tests := []struct {
		name        string
		discordID   int64
		guildID     int64
		score       WordleScore
		completedAt time.Time
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid completion",
			discordID:   123456,
			guildID:     789012,
			score:       validScore,
			completedAt: validTime,
			expectError: false,
		},
		{
			name:        "invalid discordID zero",
			discordID:   0,
			guildID:     789012,
			score:       validScore,
			completedAt: validTime,
			expectError: true,
			errorMsg:    "discordID must be greater than 0, got 0",
		},
		{
			name:        "invalid discordID negative",
			discordID:   -1,
			guildID:     789012,
			score:       validScore,
			completedAt: validTime,
			expectError: true,
			errorMsg:    "discordID must be greater than 0, got -1",
		},
		{
			name:        "invalid guildID zero",
			discordID:   123456,
			guildID:     0,
			score:       validScore,
			completedAt: validTime,
			expectError: true,
			errorMsg:    "guildID must be greater than 0, got 0",
		},
		{
			name:        "invalid guildID negative",
			discordID:   123456,
			guildID:     -1,
			score:       validScore,
			completedAt: validTime,
			expectError: true,
			errorMsg:    "guildID must be greater than 0, got -1",
		},
		{
			name:        "invalid completedAt zero time",
			discordID:   123456,
			guildID:     789012,
			score:       validScore,
			completedAt: time.Time{},
			expectError: true,
			errorMsg:    "completedAt cannot be zero time",
		},
		{
			name:        "valid with perfect score",
			discordID:   999999,
			guildID:     111111,
			score:       WordleScore{Guesses: 1},
			completedAt: validTime,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeCreate := time.Now()
			completion, err := NewWordleCompletion(tt.discordID, tt.guildID, tt.score, tt.completedAt)
			afterCreate := time.Now()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Nil(t, completion)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, completion)
				assert.Equal(t, tt.discordID, completion.DiscordID)
				assert.Equal(t, tt.guildID, completion.GuildID)
				assert.Equal(t, tt.score, completion.Score)
				assert.Equal(t, tt.completedAt, completion.CompletedAt)

				// CreatedAt should be set to approximately now
				assert.True(t, completion.CreatedAt.After(beforeCreate) || completion.CreatedAt.Equal(beforeCreate))
				assert.True(t, completion.CreatedAt.Before(afterCreate) || completion.CreatedAt.Equal(afterCreate))

				// ID should not be set by constructor
				assert.Equal(t, int64(0), completion.ID)
			}
		})
	}
}

func TestWordleCompletion_FullScenario(t *testing.T) {
	// Create a score
	score, err := NewWordleScore(3)
	assert.NoError(t, err)

	// Create a completion
	completedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	completion, err := NewWordleCompletion(123456, 789012, score, completedAt)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, int64(123456), completion.DiscordID)
	assert.Equal(t, int64(789012), completion.GuildID)
	assert.Equal(t, 3, completion.Score.Guesses)
	// MaxGuesses is no longer configurable, it's always 6
	assert.Equal(t, completedAt, completion.CompletedAt)
	assert.False(t, completion.Score.IsPerfect())

	// Calculate points
	baseReward := 1000
	points := completion.Score.BasePoints(baseReward)
	assert.Equal(t, 3000, points) // 3 guesses = 3x reward
}

func TestWordleCompletion_PerfectScoreScenario(t *testing.T) {
	// Create a perfect score
	score, err := NewWordleScore(1)
	assert.NoError(t, err)
	assert.True(t, score.IsPerfect())

	// Create a completion
	completedAt := time.Now()
	completion, err := NewWordleCompletion(999999, 111111, score, completedAt)
	assert.NoError(t, err)

	// Verify perfect score
	assert.True(t, completion.Score.IsPerfect())

	// Calculate points for perfect score
	baseReward := 500
	points := completion.Score.BasePoints(baseReward)
	assert.Equal(t, 3000, points) // 1 guess = 6x reward
}
