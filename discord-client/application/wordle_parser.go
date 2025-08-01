package application

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// WordleResult represents a parsed Wordle score for a user
type WordleResult struct {
	UserID     string
	GuessCount int
	MaxGuesses int
}

// parseWordleResults extracts Wordle results from the Discord message content
// If userResolver is provided, it will also resolve @nickname mentions to user IDs
func parseWordleResults(ctx context.Context, content string, guildID int64, userResolver UserResolver) ([]WordleResult, error) {
	// Split content into lines
	lines := strings.Split(content, "\n")

	// Regex patterns
	userPattern := regexp.MustCompile(`<@(\d+)>`)
	nickPattern := regexp.MustCompile(`@([\w\s]+?)(?:\s|$|<)`)
	scorePattern := regexp.MustCompile(`(\d+)/(\d+)`)

	var results []WordleResult

	for _, line := range lines {
		// Find score pattern in the line (should be at the beginning)
		scoreMatch := scorePattern.FindStringSubmatch(line)
		if scoreMatch == nil {
			continue
		}

		guessCount, err := strconv.Atoi(scoreMatch[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse guess count: %w", err)
		}

		maxGuesses, err := strconv.Atoi(scoreMatch[2])
		if err != nil {
			return nil, fmt.Errorf("failed to parse max guesses: %w", err)
		}

		// Find all user mentions in the line
		userMatches := userPattern.FindAllStringSubmatch(line, -1)

		// Create a result for each user ID mention
		for _, userMatch := range userMatches {
			results = append(results, WordleResult{
				UserID:     userMatch[1],
				GuessCount: guessCount,
				MaxGuesses: maxGuesses,
			})
		}

		// If we have a userResolver, also look for @nickname mentions
		if userResolver != nil {
			nickMatches := nickPattern.FindAllStringSubmatch(line, -1)
			for _, nickMatch := range nickMatches {
				nickname := strings.TrimSpace(nickMatch[1])

				// Resolve nickname to user IDs
				userIDs, err := userResolver.ResolveUsersByNick(ctx, guildID, nickname)
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"nickname": nickname,
						"guild_id": guildID,
					}).Error("Failed to resolve nickname to user ID")
					continue
				}

				// Create a result for each resolved user
				for _, userID := range userIDs {
					results = append(results, WordleResult{
						UserID:     strconv.FormatInt(userID, 10),
						GuessCount: guessCount,
						MaxGuesses: maxGuesses,
					})
				}
			}
		}
	}

	return results, nil
}

