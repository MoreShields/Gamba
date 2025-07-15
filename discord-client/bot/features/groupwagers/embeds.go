package groupwagers

import (
	"fmt"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/models"
	"sort"
	"strings"

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

// formatCompactAmount formats large amounts in a compact way (e.g., 1.2M, 500K)
func formatCompactAmount(amount int64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.0fK", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}

// CreateGroupWagerEmbed creates an embed for a group wager
func CreateGroupWagerEmbed(detail *models.GroupWagerDetail) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: detail.Wager.Condition,
		Color: common.ColorWarning,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Group Wager ID: %d", detail.Wager.ID),
		},
		Timestamp: detail.Wager.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add inline fields for pot and voting info
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Total Pot",
		Value:  fmt.Sprintf("**%s bits**", common.FormatBalance(detail.Wager.TotalPot)),
		Inline: true,
	})

	// Add voting period information for active wagers
	if detail.Wager.State == models.GroupWagerStateActive && detail.Wager.VotingEndsAt != nil {
		if detail.Wager.IsVotingPeriodActive() {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟢 Voting Open",
				Value:  fmt.Sprintf("**Ends <t:%d:R>**", detail.Wager.VotingEndsAt.Unix()),
				Inline: true,
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟠 Voting Closed",
				Value:  fmt.Sprintf("**Ended <t:%d:R>**", detail.Wager.VotingEndsAt.Unix()),
				Inline: true,
			})
		}
	}

	// Group participants by option
	participantsByOption := detail.GetParticipantsByOption()

	// Sort options by order for consistent display
	sortedOptions := make([]*models.GroupWagerOption, len(detail.Options))
	copy(sortedOptions, detail.Options)
	sort.Slice(sortedOptions, func(i, j int) bool {
		return sortedOptions[i].OptionOrder < sortedOptions[j].OptionOrder
	})

	// Add fields for each option with visual progress bars
	for _, option := range sortedOptions {
		participants := participantsByOption[option.ID]

		// Calculate percentage of total pot
		percentage := float64(0)
		if detail.Wager.TotalPot > 0 {
			percentage = float64(option.TotalAmount) * 100 / float64(detail.Wager.TotalPot)
		}

		// Calculate multiplier
		multiplier := option.CalculateMultiplier(detail.Wager.TotalPot)

		// Build the visual bar graph line
		multiplierEmoji := getMultiplierEmoji(multiplier)
		progressBar := createProgressBar(percentage, 25)

		// Format the main stats line.
		statsLine := fmt.Sprintf("%s `%s` • %-7s • %5.2fx",
			multiplierEmoji,
			progressBar,
			formatCompactAmount(option.TotalAmount)+" bits",
			multiplier)

		// Build participant info
		var participantInfo string
		if len(participants) > 0 {
			// Sort participants by amount (highest first)
			sortedParticipants := make([]*models.GroupWagerParticipant, len(participants))
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
			if len(participants) == 1 {
				participantInfo = participantTags[0]
			} else {
				// Join with bullet points for clean separation
				participantInfo = strings.Join(participantTags, " • ")
			}
		}

		// Combine all parts into field value
		fieldValue := statsLine + "\n" + participantInfo

		// Truncate if too long
		if len(fieldValue) > 1024 {
			fieldValue = fieldValue[:1021] + "..."
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s", option.OptionText),
			Value:  fieldValue,
			Inline: false,
		})
	}

	// Add summary statistics field
	if len(detail.Participants) > 0 && detail.Wager.TotalPot > 0 {
		// Count options with bets
		optionsWithBets := 0
		for _, option := range detail.Options {
			if option.TotalAmount > 0 {
				optionsWithBets++
			}
		}
	}

	// Update color based on state
	switch detail.Wager.State {
	case models.GroupWagerStateResolved:
		embed.Color = common.ColorPrimary
		embed.Description += "\n**RESOLVED**"
		if detail.Wager.WinningOptionID != nil {
			for _, option := range detail.Options {
				if option.ID == *detail.Wager.WinningOptionID {
					embed.Description += fmt.Sprintf("\nWinner: **%s**", option.OptionText)
					break
				}
			}
		}
	case models.GroupWagerStateCancelled:
		embed.Color = common.ColorDanger
		embed.Description += "\n**CANCELLED**"
	case models.GroupWagerStatePendingResolution:
		embed.Color = common.ColorPrimary
		embed.Description += "\n**⏳ AWAITING RESOLUTION**"
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  "Voting has ended. This wager is ready to be resolved.",
			Inline: false,
		})
	case models.GroupWagerStateActive:
		// For active wagers, change color based on voting period status
		if detail.Wager.IsVotingPeriodExpired() {
			embed.Color = common.ColorPrimary // voting ended, waiting for resolution
		}
	}

	// Add minimum participants info if not met
	participantCount := len(detail.Participants)
	if participantCount < detail.Wager.MinParticipants && detail.Wager.IsActive() {
		embed.Footer.Text += fmt.Sprintf(" | Need %d more participants", detail.Wager.MinParticipants-participantCount)
	} else {
		embed.Footer.Text += fmt.Sprintf(" | %d participants", len(detail.Participants))
	}

	return embed
}

// truncateButtonLabel safely truncates text to fit Discord's button label limit
func truncateButtonLabel(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Leave room for ellipsis
	truncateAt := maxLength - 3

	// Try to find a word boundary to truncate at
	for i := truncateAt; i > truncateAt-10 && i > 0; i-- {
		if text[i] == ' ' {
			return text[:i] + "..."
		}
	}

	// If no word boundary found, just truncate at the limit
	return text[:truncateAt] + "..."
}

// createGroupWagerComponents creates the button components for a group wager
func CreateGroupWagerComponents(detail *models.GroupWagerDetail) []discordgo.MessageComponent {
	// Only show components for active wagers that haven't expired
	if detail.Wager.IsActive() && detail.Wager.IsVotingPeriodActive() {
		return createActiveWagerComponents(detail)
	}

	// No components for resolved, cancelled, expired, or pending resolution wagers
	return []discordgo.MessageComponent{}
}

// createActiveWagerComponents creates betting option buttons for active wagers
func createActiveWagerComponents(detail *models.GroupWagerDetail) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	// Sort options by order
	options := make([]*models.GroupWagerOption, len(detail.Options))
	copy(options, detail.Options)
	sort.Slice(options, func(i, j int) bool {
		return options[i].OptionOrder < options[j].OptionOrder
	})

	// Create buttons for each option
	for i, option := range options {
		button := discordgo.Button{
			Label:    truncateButtonLabel(option.OptionText, 80),
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("group_wager_option_%d_%d", detail.Wager.ID, option.ID),
			Emoji: &discordgo.ComponentEmoji{
				Name: getNumberEmoji(option.OptionOrder + 1),
			},
		}

		currentRow = append(currentRow, button)

		// Max 5 buttons per row
		if len(currentRow) == 5 || i == len(options)-1 {
			rows = append(rows, discordgo.ActionsRow{
				Components: currentRow,
			})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	return rows
}

// getNumberEmoji returns the emoji for a number (1-10)
func getNumberEmoji(num int16) string {
	emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	if num >= 1 && num <= 10 {
		return emojis[num-1]
	}
	return "🔢"
}
