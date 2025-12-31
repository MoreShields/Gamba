package lottery

import (
	"fmt"

	"gambler/discord-client/domain/entities"

	"github.com/bwmarrin/discordgo"
)

// CreateLotteryComponents creates the button components for the lottery embed
func CreateLotteryComponents(draw *entities.LotteryDraw) []discordgo.MessageComponent {
	// Only show buy button if draw is still open
	if !draw.CanPurchaseTickets() {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Lottery Closed",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("lotto_closed_%d", draw.ID),
						Disabled: true,
					},
				},
			},
		}
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Buy Tickets",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("lotto_buy_%d", draw.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "🎟️",
					},
				},
			},
		},
	}
}

// CreateBuyTicketsModal creates the modal for purchasing tickets
func CreateBuyTicketsModal(drawID, ticketCost, userBalance int64) *discordgo.InteractionResponseData {
	maxAffordable := userBalance / ticketCost
	placeholderText := "1"
	if maxAffordable > 1 {
		placeholderText = fmt.Sprintf("1 (max: %d)", maxAffordable)
	}

	return &discordgo.InteractionResponseData{
		CustomID: fmt.Sprintf("lotto_buy_modal_%d", drawID),
		Title:    "Purchase Lottery Tickets",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "quantity",
						Label:       fmt.Sprintf("Number of Tickets (%d bits each)", ticketCost),
						Style:       discordgo.TextInputShort,
						Placeholder: placeholderText,
						Required:    true,
						MinLength:   1,
						MaxLength:   5,
					},
				},
			},
		},
	}
}

// CreateCompletedLotteryComponents creates disabled components for a completed draw
func CreateCompletedLotteryComponents(draw *entities.LotteryDraw) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Draw Complete",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("lotto_complete_%d", draw.ID),
					Disabled: true,
				},
			},
		},
	}
}
