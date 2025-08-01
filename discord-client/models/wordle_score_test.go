package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWordleScore(t *testing.T) {
	tests := []struct {
		name        string
		guesses     int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid score with 3 guesses",
			guesses:     3,
			expectError: false,
		},
		{
			name:        "valid perfect score",
			guesses:     1,
			expectError: false,
		},
		{
			name:        "valid last guess",
			guesses:     6,
			expectError: false,
		},
		{
			name:        "invalid guesses too low",
			guesses:     0,
			expectError: true,
			errorMsg:    "guesses must be between 1 and 6, got 0",
		},
		{
			name:        "invalid guesses too high",
			guesses:     7,
			expectError: true,
			errorMsg:    "guesses must be between 1 and 6, got 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewWordleScore(tt.guesses)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Equal(t, WordleScore{}, score)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.guesses, score.Guesses)
			}
		})
	}
}

func TestWordleScore_BasePoints(t *testing.T) {
	baseReward := 100

	tests := []struct {
		name           string
		guesses        int
		expectedPoints int
	}{
		{
			name:           "1 guess = 6x reward",
			guesses:        1,
			expectedPoints: 600,
		},
		{
			name:           "2 guesses = 4x reward",
			guesses:        2,
			expectedPoints: 400,
		},
		{
			name:           "3 guesses = 3x reward",
			guesses:        3,
			expectedPoints: 300,
		},
		{
			name:           "4 guesses = 2x reward",
			guesses:        4,
			expectedPoints: 200,
		},
		{
			name:           "5 guesses = 1x reward",
			guesses:        5,
			expectedPoints: 100,
		},
		{
			name:           "6 guesses = 0.5x reward",
			guesses:        6,
			expectedPoints: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewWordleScore(tt.guesses)
			assert.NoError(t, err)

			points := score.BasePoints(baseReward)
			assert.Equal(t, tt.expectedPoints, points)
		})
	}
}

func TestWordleScore_BasePoints_EdgeCases(t *testing.T) {
	// Test with different base rewards
	score, err := NewWordleScore(1)
	assert.NoError(t, err)

	assert.Equal(t, 0, score.BasePoints(0))
	assert.Equal(t, 6, score.BasePoints(1))
	assert.Equal(t, 600, score.BasePoints(100))
	assert.Equal(t, 6000, score.BasePoints(1000))

	// Test division for 6 guesses
	score6, err := NewWordleScore(6)
	assert.NoError(t, err)

	assert.Equal(t, 0, score6.BasePoints(1))    // 1/2 = 0 in integer division
	assert.Equal(t, 1, score6.BasePoints(2))    // 2/2 = 1
	assert.Equal(t, 50, score6.BasePoints(100)) // 100/2 = 50
}

func TestWordleScore_IsPerfect(t *testing.T) {
	tests := []struct {
		name      string
		guesses   int
		isPerfect bool
	}{
		{
			name:      "1 guess is perfect",
			guesses:   1,
			isPerfect: true,
		},
		{
			name:      "2 guesses is not perfect",
			guesses:   2,
			isPerfect: false,
		},
		{
			name:      "6 guesses is not perfect",
			guesses:   6,
			isPerfect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewWordleScore(tt.guesses)
			assert.NoError(t, err)

			assert.Equal(t, tt.isPerfect, score.IsPerfect())
		})
	}
}
