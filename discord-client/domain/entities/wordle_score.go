package entities

import (
	"fmt"
)

// WordleScore represents a value object for Wordle game scoring
type WordleScore struct {
	Guesses int `db:"guesses"`
}

// NewWordleScore creates a new WordleScore with validation
func NewWordleScore(guesses int) (WordleScore, error) {
	if guesses < 1 || guesses > 6 {
		return WordleScore{}, fmt.Errorf("guesses must be between 1 and 6, got %d", guesses)
	}

	return WordleScore{
		Guesses: guesses,
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

// IsExcellent returns true if the Wordle was solved in 1-2 guesses
func (ws WordleScore) IsExcellent() bool {
	return ws.Guesses <= 2
}

// IsGood returns true if the Wordle was solved in 3-4 guesses
func (ws WordleScore) IsGood() bool {
	return ws.Guesses >= 3 && ws.Guesses <= 4
}

// IsAverage returns true if the Wordle was solved in 5-6 guesses
func (ws WordleScore) IsAverage() bool {
	return ws.Guesses >= 5
}