package groupwagers
import (
	"gambler/bot/common"
	"fmt"
	"gambler/models"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// createGroupWagerEmbed creates an embed for a group wager
func createGroupWagerEmbed(detail *models.GroupWagerDetail) *discordgo.MessageEmbed {
	// Build description with pot info and voting period
	description := fmt.Sprintf("**Total Pot: %s bits**", common.FormatBalance(detail.Wager.TotalPot))
	
	// Add voting period information for active wagers
	if detail.Wager.State == models.GroupWagerStateActive && detail.Wager.VotingEndsAt != nil {
		if detail.Wager.IsVotingPeriodActive() {
			description += fmt.Sprintf("\n**Voting ends: <t:%d:R>**", detail.Wager.VotingEndsAt.Unix())
		} else {
			description += fmt.Sprintf("\n**Voting ended: <t:%d:R>**", detail.Wager.VotingEndsAt.Unix())
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       detail.Wager.Condition,
		Description: description,
		Color:       common.ColorWarning,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Group Wager ID: %d", detail.Wager.ID),
		},
		Timestamp: detail.Wager.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Group participants by option
	participantsByOption := detail.GetParticipantsByOption()

	// Add fields for each option
	for _, option := range detail.Options {
		participants := participantsByOption[option.ID]

		// Build participant list
		var participantList []string
		for _, p := range participants {
			participantList = append(participantList, fmt.Sprintf("<@%d>: %s bits", p.DiscordID, common.FormatBalance(p.Amount)))
		}

		// Calculate multiplier
		multiplier := option.CalculateMultiplier(detail.Wager.TotalPot)

		fieldValue := fmt.Sprintf("**Total: %s bits** (%.2fx multiplier)\n", common.FormatBalance(option.TotalAmount), multiplier)
		if len(participantList) > 0 {
			fieldValue += strings.Join(participantList, "\n")
		} else {
			fieldValue += "*No participants yet*"
		}

		// Truncate if too long
		if len(fieldValue) > 1024 {
			fieldValue = fieldValue[:1021] + "..."
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Option %d: %s", option.OptionOrder+1, option.OptionText),
			Value:  fieldValue,
			Inline: false,
		})
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
func createGroupWagerComponents(detail *models.GroupWagerDetail) []discordgo.MessageComponent {
	// Only show components for active wagers that haven't expired
	if detail.Wager.IsActive() && detail.Wager.IsVotingPeriodActive() {
		return createActiveWagerComponents(detail)
	}
	
	// No components for resolved, cancelled, or expired wagers
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
