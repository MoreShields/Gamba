package betting

import (
	"fmt"

	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
)

// buildBetAmountModal creates the modal for entering bet amount
func buildBetAmountModal(odds float64, balance int64) *discordgo.InteractionResponseData {
	percentage := int(odds * 100)
	label := fmt.Sprintf("Bet Amount (Balance: %s bits)", common.FormatBalance(balance))

	return &discordgo.InteractionResponseData{
		CustomID: "bet_amount_modal",
		Title:    fmt.Sprintf("Place Bet at %d%% Odds", percentage),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "bet_amount_input",
						Label:       label,
						Style:       discordgo.TextInputShort,
						Placeholder: "Enter amount to bet",
						Required:    true,
						MaxLength:   20,
					},
				},
			},
		},
	}
}
