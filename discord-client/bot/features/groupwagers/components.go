package groupwagers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// CreateBetButtons creates the betting buttons for a group wager
func CreateBetButtons(groupWagerID int64, options []string) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	for i, option := range options {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("💰 Bet on %s", option),
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("groupwager_bet_%d_%d", groupWagerID, i+1),
		})
	}

	// Create rows of buttons (max 5 per row)
	var rows []discordgo.MessageComponent
	for i := 0; i < len(buttons); i += 5 {
		end := i + 5
		if end > len(buttons) {
			end = len(buttons)
		}

		rows = append(rows, discordgo.ActionsRow{
			Components: buttons[i:end],
		})
	}

	return rows
}

// CreateResolveButtons creates admin buttons for resolving a group wager
func CreateResolveButtons(groupWagerID int64, options []string) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	for i, option := range options {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("✅ %s Wins", option),
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("groupwager_resolve_%d_%d", groupWagerID, i+1),
		})
	}

	// Add cancel button
	buttons = append(buttons, discordgo.Button{
		Label:    "❌ Cancel Wager",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("groupwager_cancel_%d", groupWagerID),
	})

	// Create rows of buttons
	var rows []discordgo.MessageComponent
	for i := 0; i < len(buttons); i += 5 {
		end := i + 5
		if end > len(buttons) {
			end = len(buttons)
		}

		rows = append(rows, discordgo.ActionsRow{
			Components: buttons[i:end],
		})
	}

	return rows
}
