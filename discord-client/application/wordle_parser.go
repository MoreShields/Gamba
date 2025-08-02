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

		// Track users already processed for this line to avoid duplicates
		processedUsers := make(map[string]bool)

		// Find all user mentions in the line
		userMatches := userPattern.FindAllStringSubmatch(line, -1)

		// Create a result for each user ID mention
		for _, userMatch := range userMatches {
			userID := userMatch[1]
			if !processedUsers[userID] {
				results = append(results, WordleResult{
					UserID:     userID,
					GuessCount: guessCount,
					MaxGuesses: maxGuesses,
				})
				processedUsers[userID] = true
			}
		}

		// If we have a userResolver, also look for @nickname mentions
		if userResolver != nil {
			// Find all @ positions that are not part of <@userid> pattern
			atPositions := []int{}
			for i := 0; i < len(line); i++ {
				if line[i] == '@' && (i == 0 || line[i-1] != '<') {
					atPositions = append(atPositions, i)
				}
			}

			// Extract nickname from each @ position
			for _, pos := range atPositions {
				// Find the end of the nickname
				end := pos + 1
				for end < len(line) {
					ch := line[end]
					if ch == '@' || ch == '<' || ch == '\n' {
						break
					}
					end++
				}

				nickname := strings.TrimSpace(line[pos+1 : end])
				if nickname == "" {
					continue
				}

				// Debug logging
				log.WithFields(log.Fields{
					"nickname": nickname,
					"line":     line,
					"position": pos,
					"raw":      line[pos:end],
				}).Debug("Found nickname mention")

				// Resolve nickname to user IDs
				userIDs, err := userResolver.ResolveUsersByNick(ctx, guildID, nickname)
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"nickname": nickname,
						"guild_id": guildID,
					}).Error("Failed to resolve nickname to user ID")
					continue
				}

				// Log resolution result
				if len(userIDs) == 0 {
					log.WithFields(log.Fields{
						"nickname": nickname,
						"guild_id": guildID,
					}).Warn("No users found for nickname")
				} else {
					log.WithFields(log.Fields{
						"nickname":   nickname,
						"guild_id":   guildID,
						"user_count": len(userIDs),
						"user_ids":   userIDs,
					}).Debug("Resolved nickname to users")
				}

				// Create a result for each resolved user (skip if already processed)
				for _, userID := range userIDs {
					userIDStr := strconv.FormatInt(userID, 10)
					if !processedUsers[userIDStr] {
						results = append(results, WordleResult{
							UserID:     userIDStr,
							GuessCount: guessCount,
							MaxGuesses: maxGuesses,
						})
						processedUsers[userIDStr] = true
					}
				}
			}
		}
	}

	// Log parsing summary
	if len(results) > 0 {
		log.WithFields(log.Fields{
			"total_results": len(results),
			"guild_id":      guildID,
		}).Info("Wordle parsing complete")
	}

	return results, nil
}
