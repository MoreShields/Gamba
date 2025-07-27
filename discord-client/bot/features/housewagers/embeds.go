package housewagers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
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
	// Build footer text
	footerText := fmt.Sprintf("House Wager ID: %d", houseWager.WagerID)

	embed := &discordgo.MessageEmbed{
		Title:       houseWager.Title,
		Description: houseWager.Description,
		Color:       common.ColorWarning, // Orange for house wagers to distinguish from group wagers
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
	if houseWager.VotingEndsAt != nil {
		if houseWager.VotingEndsAt.After(time.Now()) {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟢 Betting Open",
				Value:  fmt.Sprintf("**Closes <t:%d:R>**", houseWager.VotingEndsAt.Unix()),
				Inline: true,
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟠 Betting Closed",
				Value:  "",
				Inline: true,
			})
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
			Name:   fmt.Sprintf("%s", option.Text),
			Value:  fieldValue,
			Inline: false,
		})
	}

	// Add participant count to footer
	participantCount := len(houseWager.Participants)
	if participantCount > 0 {
		embed.Footer.Text += fmt.Sprintf(" | %d participants", participantCount)
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
	embed := CreateHouseWagerEmbed(houseWager)

	// Update for resolved state
	embed.Color = common.ColorPrimary // Blue for resolved
	embed.Title = "🏆 " + embed.Title + " - RESOLVED"

	// Add resolution information
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🎉 Result",
		Value:  fmt.Sprintf("**%s** won!\nTotal payout: **%s bits**", winningOption, formatCompactAmount(totalPayout)),
		Inline: false,
	})

	embed.Footer.Text += " | RESOLVED"

	return embed
}
