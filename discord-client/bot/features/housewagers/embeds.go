package housewagers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// createProgressBar generates a visual progress bar using Unicode block characters
func createProgressBar(percentage float64, length int) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(length) * percentage / 100)
	if filled > length {
		filled = length
	}

	bar := strings.Repeat("█", filled)
	bar += strings.Repeat("░", length-filled)

	return bar
}

// getMultiplierEmoji returns an emoji indicator based on the multiplier value
func getMultiplierEmoji(multiplier float64) string {
	if multiplier < 1.5 {
		return "🟩" // Green - favorite
	} else if multiplier < 3.0 {
		return "🟨" // Yellow - moderate
	}
	return "🟥" // Red - underdog
}

// CreateHouseWagerEmbed creates an embed for a house wager
func CreateHouseWagerEmbed(houseWager dto.HouseWagerPostDTO) *discordgo.MessageEmbed {
	// Check if this is a resolved wager and delegate to resolved embed
	if houseWager.State == "resolved" {
		// Find the winning option and calculate total payout
		var winningOption string
		var totalPayout int64

		// Find the winning option using WinningOptionID
		if houseWager.WinningOptionID != nil {
			for _, option := range houseWager.Options {
				if option.ID == *houseWager.WinningOptionID {
					winningOption = option.Text

					// Calculate total payout by looking at winners (participants who bet on winning option)
					for _, participant := range houseWager.Participants {
						if participant.OptionID == *houseWager.WinningOptionID {
							// For house wagers, payout = bet amount * multiplier
							totalPayout += int64(float64(participant.Amount) * option.Multiplier)
						}
					}
					break
				}
			}
		}

		// This should never happen if validation worked correctly
		if winningOption == "" {
			log.WithFields(log.Fields{
				"wagerID":         houseWager.WagerID,
				"winningOptionID": houseWager.WinningOptionID,
				"optionCount":     len(houseWager.Options),
				"state":           houseWager.State,
			}).Error("Resolved house wager missing valid winning option - this indicates a data integrity issue")

			// Return an error embed instead of continuing with invalid data
			return &discordgo.MessageEmbed{
				Title:       "⚠️ Wager Resolution Error",
				Description: "This wager has been marked as resolved but the winning option could not be determined.",
				Color:       common.ColorDanger,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("House Wager ID: %d | ERROR", houseWager.WagerID),
				},
			}
		}

		return CreateHouseWagerResolvedEmbed(houseWager, winningOption, totalPayout)
	}

	// Check if this is a cancelled wager
	if houseWager.State == "cancelled" {
		return CreateHouseWagerCancelledEmbed(houseWager)
	}

	// For active or non-resolved wagers, use the base embed
	return createBaseHouseWagerEmbed(houseWager)
}

// createBaseHouseWagerEmbed creates the base embed for a house wager (used by both active and resolved embeds)
func createBaseHouseWagerEmbed(houseWager dto.HouseWagerPostDTO) *discordgo.MessageEmbed {
	// Build footer text
	footerText := fmt.Sprintf("House Wager ID: %d", houseWager.WagerID)

	embed := &discordgo.MessageEmbed{
		Title:       houseWager.Title,       // Title from the DTO
		Description: houseWager.Description, // Description from the DTO
		Color:       common.ColorWarning,    // Orange for house wagers to distinguish from group wagers
		Timestamp:   time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
	}

	// Add total pot information if there are participants
	if len(houseWager.Participants) > 0 && houseWager.TotalPot > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Total Pot",
			Value:  fmt.Sprintf("**%s bits**", common.FormatBalance(houseWager.TotalPot)),
			Inline: true,
		})
	}

	// Add betting countdown field inline with summoner info
	// Skip this field for cancelled wagers
	if houseWager.VotingEndsAt != nil && houseWager.State != "cancelled" {
		if houseWager.VotingEndsAt.After(time.Now()) {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟢 Betting Open",
				Value:  fmt.Sprintf("**Closes <t:%d:R>**", houseWager.VotingEndsAt.Unix()),
				Inline: true,
			})
		} else {
			if houseWager.State != "resolved" {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "🟠 Betting Closed",
					Value:  "",
					Inline: true,
				})
			}
		}
	}

	// Group participants by option
	participantsByOption := make(map[int64][]dto.ParticipantDTO)
	for _, participant := range houseWager.Participants {
		participantsByOption[participant.OptionID] = append(participantsByOption[participant.OptionID], participant)
	}

	// Sort options by order for consistent display
	sortedOptions := make([]dto.WagerOptionDTO, len(houseWager.Options))
	copy(sortedOptions, houseWager.Options)
	sort.Slice(sortedOptions, func(i, j int) bool {
		return sortedOptions[i].Order < sortedOptions[j].Order
	})

	// Add fields for each option with visual progress bars and participants
	for _, option := range sortedOptions {
		participants := participantsByOption[option.ID]

		// Calculate percentage of total pot
		percentage := float64(0)
		if houseWager.TotalPot > 0 {
			percentage = float64(option.TotalAmount) * 100 / float64(houseWager.TotalPot)
		}

		// Use stored multiplier for consistent display
		multiplier := option.Multiplier

		// Build participant info if there are any
		var fieldValue string
		if len(participants) > 0 {
			// Build the visual bar graph line
			multiplierEmoji := getMultiplierEmoji(multiplier)
			progressBar := createProgressBar(percentage, 25)

			// Format the main stats line
			statsLine := fmt.Sprintf("%s `%s` • %-7s • %5.2fx",
				multiplierEmoji,
				progressBar,
				formatCompactAmount(option.TotalAmount)+" bits",
				multiplier)

			// Sort participants by amount (highest first)
			sortedParticipants := make([]dto.ParticipantDTO, len(participants))
			copy(sortedParticipants, participants)
			sort.Slice(sortedParticipants, func(i, j int) bool {
				return sortedParticipants[i].Amount > sortedParticipants[j].Amount
			})

			// Build participant list with amounts
			participantTags := make([]string, 0, len(sortedParticipants))
			for _, p := range sortedParticipants {
				participantTags = append(participantTags, fmt.Sprintf("<@%d> - %s", p.DiscordID, formatCompactAmount(p.Amount)))
			}

			// Format participant line with clean delineation
			var participantInfo string
			if len(participants) == 1 {
				participantInfo = participantTags[0]
			} else {
				// Join with bullet points for clean separation
				participantInfo = strings.Join(participantTags, " • ")
			}

			// Combine all parts into field value
			fieldValue = statsLine + "\n" + participantInfo
		} else {
			// Show betting options with fixed odds when no participants
			emoji := getOptionEmoji(int(option.Order) + 1)
			fieldValue = fmt.Sprintf("%s **%.2fx odds**",
				emoji,
				option.Multiplier)
		}

		// Truncate if too long
		if len(fieldValue) > 1024 {
			fieldValue = fieldValue[:1021] + "..."
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   option.Text,
			Value:  fieldValue,
			Inline: false,
		})
	}

	// Add participant count to footer
	participantCount := len(houseWager.Participants)
	if participantCount > 0 {
		embed.Footer.Text += fmt.Sprintf(" • %d participants", participantCount)
	}

	return embed
}

// getOptionEmoji returns appropriate emoji for betting options
func getOptionEmoji(optionNumber int) string {
	switch optionNumber {
	case 1:
		return "🟢" // Green for Win
	case 2:
		return "🔴" // Red for Loss
	default:
		return "⚪" // White for additional options
	}
}

// formatCompactAmount formats large amounts in a compact way (e.g., 1.2M, 500K)
func formatCompactAmount(amount int64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.0fK", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}

// CreateHouseWagerResolvedEmbed creates an embed for a resolved house wager
func CreateHouseWagerResolvedEmbed(houseWager dto.HouseWagerPostDTO, winningOption string, totalPayout int64) *discordgo.MessageEmbed {
	// Create base embed without calling CreateHouseWagerEmbed to avoid recursion
	embed := createBaseHouseWagerEmbed(houseWager)

	// Update for resolved state
	embed.Color = common.ColorPrimary // Blue for resolved

	// Only append RESOLVED to title if there is a title
	if embed.Title != "" {
		embed.Title = embed.Title + " - RESOLVED"
	} else {
		// If no title, prepend RESOLVED to description
		embed.Description = "**RESOLVED**\n\n" + embed.Description
	}

	// Find winners and show detailed results
	var winnerInfo strings.Builder
	var winnerCount int

	if houseWager.WinningOptionID != nil {
		// Find winning option details
		var winningOptionObj *dto.WagerOptionDTO
		for _, option := range houseWager.Options {
			if option.ID == *houseWager.WinningOptionID {
				winningOptionObj = &option
				break
			}
		}

		// Count winners and build winner list
		for _, participant := range houseWager.Participants {
			if participant.OptionID == *houseWager.WinningOptionID {
				if winnerCount > 0 {
					winnerInfo.WriteString(" • ")
				}

				payout := int64(float64(participant.Amount) * winningOptionObj.Multiplier)
				winnerInfo.WriteString(fmt.Sprintf("<@%d> (+%s)", participant.DiscordID, formatCompactAmount(payout)))
				winnerCount++
			}
		}
	}

	// Build result field
	resultValue := fmt.Sprintf("**%s**", winningOption)
	if totalPayout > 0 {
		resultValue += fmt.Sprintf("\nTotal payout: **%s bits**", formatCompactAmount(totalPayout))
	}
	if winnerCount > 0 {
		if winnerCount == 1 {
			resultValue += fmt.Sprintf("\n**Winner:** %s", winnerInfo.String())
		} else {
			resultValue += fmt.Sprintf("\n**Winners (%d):** %s", winnerCount, winnerInfo.String())
		}
	} else {
		resultValue += "\n*No winners - House keeps all bets*"
	}

	// Add resolution information
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🎉 Result",
		Value:  resultValue,
		Inline: false,
	})

	return embed
}

// CreateHouseWagerCancelledEmbed creates an embed for a cancelled house wager
func CreateHouseWagerCancelledEmbed(houseWager dto.HouseWagerPostDTO) *discordgo.MessageEmbed {
	// Create base embed
	embed := createBaseHouseWagerEmbed(houseWager)

	// Update for cancelled state
	embed.Color = common.ColorDanger // Red for cancelled

	// Update title to show CANCELLED status
	if embed.Title != "" {
		embed.Title = embed.Title + " - CANCELLED"
	} else {
		// If no title, prepend CANCELLED to description
		embed.Description = "**CANCELLED**\n\n" + embed.Description
	}

	// Remove the Match Details link from the description since the game was forfeit/remake
	// The description typically contains "[Match Details](url)" which we want to remove
	if strings.Contains(embed.Description, "[Match Details]") {
		// Find and remove the entire line containing the Match Details link
		lines := strings.Split(embed.Description, "\n")
		var filteredLines []string
		for _, line := range lines {
			if !strings.Contains(line, "[Match Details]") {
				filteredLines = append(filteredLines, line)
			}
		}
		embed.Description = strings.Join(filteredLines, "\n")
	}

	// Add cancellation notice field
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "❌ Wager Cancelled",
		Value:  "Game ended in less than 10 minutes (forfeit/remake).\nAll bets have been refunded.",
		Inline: false,
	})

	return embed
}
