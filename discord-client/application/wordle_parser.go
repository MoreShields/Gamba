package application

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// WordleResult represents a parsed Wordle score for a user
type WordleResult struct {
	UserID     string
	GuessCount int
	MaxGuesses int
}

// parseWordleResults extracts Wordle results from the Discord message content
func parseWordleResults(content string) ([]WordleResult, error) {
	// Split content into lines
	lines := strings.Split(content, "\n")
	
	// Regex patterns
	userPattern := regexp.MustCompile(`<@(\d+)>`)
	scorePattern := regexp.MustCompile(`(\d+)/(\d+)`)
	
	var results []WordleResult
	
	for _, line := range lines {
		// Find all user mentions in the line
		userMatches := userPattern.FindAllStringSubmatch(line, -1)
		
		// Find score pattern in the line (should be at the beginning)
		scoreMatch := scorePattern.FindStringSubmatch(line)
		
		// If we have both a score and at least one user on this line
		if scoreMatch != nil && len(userMatches) > 0 {
			guessCount, err := strconv.Atoi(scoreMatch[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse guess count: %w", err)
			}
			
			maxGuesses, err := strconv.Atoi(scoreMatch[2])
			if err != nil {
				return nil, fmt.Errorf("failed to parse max guesses: %w", err)
			}
			
			// Create a result for each user on this line
			for _, userMatch := range userMatches {
				results = append(results, WordleResult{
					UserID:     userMatch[1],
					GuessCount: guessCount,
					MaxGuesses: maxGuesses,
				})
			}
		}
	}
	
	return results, nil
}