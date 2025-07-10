package betting

import (
	"fmt"

	"gambler/bot/common"

	"github.com/bwmarrin/discordgo"
)

// Predefined odds options
var oddsOptions = []struct {
	percentage int
	label      string
}{
	{10, "10%"},
	{25, "25%"},
	{33, "33%"},
	{50, "50%"},
	{66, "66%"},
	{75, "75%"},
	{80, "80%"},
	{90, "90%"},
	{95, "95%"},
}

// CreateInitialComponents creates the initial betting interface components
func CreateInitialComponents() []discordgo.MessageComponent {
	return buildOddsButtons()
}

// buildOddsButtons creates the odds selection button grid
func buildOddsButtons() []discordgo.MessageComponent {
	components := []discordgo.MessageComponent{}
	
	// Create 3 rows of 3 buttons each
	for row := 0; row < 3; row++ {
		buttons := []discordgo.MessageComponent{}
		
		for col := 0; col < 3; col++ {
			idx := row*3 + col
			if idx < len(oddsOptions) {
				opt := oddsOptions[idx]
				payout := calculatePayoutRatio(float64(opt.percentage) / 100.0)
				
				buttons = append(buttons, discordgo.Button{
					Label:    fmt.Sprintf("🎯 %s | Pays %s", opt.label, payout),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("bet_odds_%d", opt.percentage),
				})
			}
		}
		
		if len(buttons) > 0 {
			components = append(components, discordgo.ActionsRow{
				Components: buttons,
			})
		}
	}
	
	return components
}

// buildBetAmountModal creates the modal for entering bet amount
// Note: This is duplicated in modal.go, remove from here
func buildBetAmountModalOld(odds float64, balance int64) discordgo.InteractionResponseData {
	percentage := int(odds * 100)
	payout := calculatePayoutRatio(odds)
	
	// Suggest 10% of balance or 100, whichever is larger
	suggestedAmount := balance / 10
	if suggestedAmount < 100 {
		suggestedAmount = 100
	}
	if suggestedAmount > balance {
		suggestedAmount = balance
	}
	
	return discordgo.InteractionResponseData{
		CustomID: "bet_amount_modal",
		Title:    fmt.Sprintf("Bet at %d%% odds (Pays %s)", percentage, payout),
		Components: []discordgo.MessageComponent{
			// Bet amount input
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "bet_amount_input",
						Label:       fmt.Sprintf("Bet Amount (Balance: %s bits)", common.FormatBalance(balance)),
						Style:       discordgo.TextInputShort,
						Placeholder: fmt.Sprintf("%d", suggestedAmount),
						Required:    true,
						MinLength:   1,
						MaxLength:   20,
					},
				},
			},
		},
	}
}

// CreateActionButtons creates the post-bet action buttons
func CreateActionButtons(lastAmount, balance int64) []discordgo.MessageComponent {
	return buildActionButtons(lastAmount, balance)
}

// buildActionButtons creates the post-bet action buttons
func buildActionButtons(lastAmount, balance int64) []discordgo.MessageComponent {
	// Calculate double and halve amounts
	doubleAmount := lastAmount * 2
	halveAmount := lastAmount / 2
	if halveAmount < 1 {
		halveAmount = 1
	}
	
	// Create buttons
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "🎲 New Bet",
			Style:    discordgo.PrimaryButton,
			CustomID: "bet_new",
		},
	}
	
	// Add repeat same button if balance allows
	if lastAmount <= balance {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("🔄 Repeat (%s)", common.FormatBalance(lastAmount)),
			Style:    discordgo.PrimaryButton,
			CustomID: "bet_repeat",
		})
	} else {
		buttons = append(buttons, discordgo.Button{
			Label:    "🔄 Repeat (Insufficient Balance)",
			Style:    discordgo.SecondaryButton,
			CustomID: "bet_repeat",
			Disabled: true,
		})
	}
	
	// Add double button if balance allows
	if doubleAmount <= balance {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("⬆️ Double (%s)", common.FormatBalance(doubleAmount)),
			Style:    discordgo.SuccessButton,
			CustomID: "bet_double",
		})
	} else {
		buttons = append(buttons, discordgo.Button{
			Label:    "⬆️ Double (Insufficient Balance)",
			Style:    discordgo.SecondaryButton,
			CustomID: "bet_double",
			Disabled: true,
		})
	}
	
	// Add halve button
	buttons = append(buttons, discordgo.Button{
		Label:    fmt.Sprintf("⬇️ Halve (%s)", common.FormatBalance(halveAmount)),
		Style:    discordgo.SecondaryButton,
		CustomID: "bet_halve",
	})
	
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: buttons,
		},
	}
}

// calculatePayoutRatio returns a user-friendly payout ratio string
func calculatePayoutRatio(winProbability float64) string {
	if winProbability <= 0 || winProbability >= 1 {
		return "Invalid"
	}
	
	// Calculate the payout multiplier
	payoutMultiplier := (1 - winProbability) / winProbability
	
	// Format based on whether payout is greater or less than 1:1
	if payoutMultiplier >= 1 {
		// Express as X:1 for payouts >= 1
		return fmt.Sprintf("%.0f:1", payoutMultiplier)
	} else {
		// Express as 1:X for payouts < 1
		return fmt.Sprintf("1:%.0f", 1/payoutMultiplier)
	}
}