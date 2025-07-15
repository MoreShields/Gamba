package betting

import (
	"fmt"

	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
)

// buildBetAmountModal creates the modal for entering bet amount
func buildBetAmountModal(odds float64, balance int64, remainingLimit int64) *discordgo.InteractionResponseData {
	percentage := int(odds * 100)

	// Build label with balance and daily limit info
	label := fmt.Sprintf("Bet Amount (Balance: %s bits)", common.FormatBalance(balance))
	if remainingLimit > 0 {
		label = fmt.Sprintf("Bet Amount (Balance: %s)",
			common.FormatBalance(balance),
			//common.FormatBalance(remainingLimit))
		)
	}

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
