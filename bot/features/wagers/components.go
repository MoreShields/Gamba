package wagers

import (
	"fmt"

	"gambler/models"

	"github.com/bwmarrin/discordgo"
)

// BuildWagerProposalComponents creates the accept/decline buttons for a proposed wager
func BuildWagerProposalComponents(wagerID int64) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "✅ Accept",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("wager_accept_%d", wagerID),
				},
				discordgo.Button{
					Label:    "❌ Decline",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("wager_decline_%d", wagerID),
				},
			},
		},
	}
}

// BuildWagerVotingComponents creates voting buttons for a wager
func BuildWagerVotingComponents(wager *models.Wager, proposerName, targetName string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    fmt.Sprintf("Vote for %s", proposerName),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("wager_vote_%d_%d", wager.ID, wager.ProposerDiscordID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "👤",
					},
				},
				discordgo.Button{
					Label:    fmt.Sprintf("Vote for %s", targetName),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("wager_vote_%d_%d", wager.ID, wager.TargetDiscordID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "👤",
					},
				},
			},
		},
	}
}

// BuildWagerConditionModal creates a modal for entering wager condition
func BuildWagerConditionModal(proposerID, targetID, amount int64) discordgo.InteractionResponseData {
	customID := fmt.Sprintf("wager_condition_modal_%d_%d_%d", proposerID, targetID, amount)

	return discordgo.InteractionResponseData{
		CustomID: customID,
		Title:    "Create Wager",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "wager_condition_input",
						Label:       "Wager Condition",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "Enter the condition for this wager (e.g., 'I will beat you in the next game')",
						Required:    true,
						MinLength:   10,
						MaxLength:   500,
					},
				},
			},
		},
	}
}

// DisableComponents disables all interactive components in a message
func DisableComponents(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	disabled := make([]discordgo.MessageComponent, len(components))

	for i, component := range components {
		if row, ok := component.(discordgo.ActionsRow); ok {
			newRow := discordgo.ActionsRow{
				Components: make([]discordgo.MessageComponent, len(row.Components)),
			}

			for j, comp := range row.Components {
				if button, ok := comp.(discordgo.Button); ok {
					button.Disabled = true
					newRow.Components[j] = button
				} else {
					newRow.Components[j] = comp
				}
			}

			disabled[i] = newRow
		} else {
			disabled[i] = component
		}
	}

	return disabled
}
