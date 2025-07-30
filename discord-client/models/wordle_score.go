package models

import (
	"fmt"
)

// WordleScore represents a value object for Wordle game scoring
type WordleScore struct {
	Guesses    int
	MaxGuesses int
}

// NewWordleScore creates a new WordleScore with validation
func NewWordleScore(guesses, maxGuesses int) (WordleScore, error) {
	if maxGuesses < 1 || maxGuesses > 6 {
		return WordleScore{}, fmt.Errorf("maxGuesses must be between 1 and 6, got %d", maxGuesses)
	}
	if guesses < 1 || guesses > maxGuesses {
		return WordleScore{}, fmt.Errorf("guesses must be between 1 and %d, got %d", maxGuesses, guesses)
	}

	return WordleScore{
		Guesses:    guesses,
		MaxGuesses: maxGuesses,
	}, nil
}

// BasePoints calculates the base points earned based on the number of guesses
func (ws WordleScore) BasePoints(baseReward int) int {
	switch ws.Guesses {
	case 1:
		return baseReward * 6
	case 2:
		return baseReward * 4
	case 3:
		return baseReward * 3
	case 4:
		return baseReward * 2
	case 5:
		return baseReward * 1
	case 6:
		return baseReward / 2
	default:
		return 0
	}
}

// IsPerfect returns true if the Wordle was solved in one guess
func (ws WordleScore) IsPerfect() bool {
	return ws.Guesses == 1
}
