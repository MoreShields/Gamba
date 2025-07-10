package betting
import (
	"gambler/bot/common"
	"fmt"

	"gambler/models"

	"github.com/bwmarrin/discordgo"
)


// buildInitialBetEmbed creates the initial betting interface embed
func buildInitialBetEmbed(balance int64) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🎰 **Place Your Bet** 🎰",
		Description: fmt.Sprintf("Current Balance: **%s bits**\n", common.FormatBalance(balance)),
		Color:       common.ColorPrimary,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Select your win probability",
		},
	}
}

// buildWinEmbed creates the embed for a winning bet
func buildWinEmbed(result *models.BetResult, odds float64, session *BetSession) *discordgo.MessageEmbed {
	percentage := int(odds * 100)

	fields := []*discordgo.MessageEmbedField{
		{
			Name: "Bet Details",
			Value: fmt.Sprintf("• Bet: **%s bits** at %d%% odds\n• Won: **%s bits**",
				common.FormatBalance(result.BetAmount),
				percentage,
				common.FormatBalance(result.WinAmount),
			),
			Inline: false,
		},
	}

	// Add session PnL indicator
	pnlPercent := float64(session.SessionPnL) / float64(session.StartingBalance) * 100
	var pnlDisplay string

	if session.SessionPnL > 0 {
		pnlDisplay = fmt.Sprintf("+%s bits (+%.1f%% return, %d bets)", common.FormatBalance(session.SessionPnL), pnlPercent, session.BetCount)
	} else if session.SessionPnL < 0 {
		pnlDisplay = fmt.Sprintf("-%s bits (%.1f%% return, %d bets)", common.FormatBalance(-session.SessionPnL), pnlPercent, session.BetCount)
	} else {
		pnlDisplay = fmt.Sprintf("0 bits (0.0%% return, %d bets)", session.BetCount)
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Session PnL",
		Value:  pnlDisplay,
		Inline: true,
	})

	return &discordgo.MessageEmbed{
		Description: fmt.Sprintf("🎉 **WINNER!** 🎉\nBalance: **%s bits**", common.FormatBalance(result.NewBalance)),
		Color:       common.ColorSuccess,
		Fields:      fields,
	}
}

// buildLossEmbed creates the embed for a losing bet
func buildLossEmbed(result *models.BetResult, odds float64, session *BetSession) *discordgo.MessageEmbed {
	percentage := int(odds * 100)

	fields := []*discordgo.MessageEmbedField{
		{
			Name: "Bet Details",
			Value: fmt.Sprintf("• Bet: **%s bits** at %d%% odds\n• Lost: **%s bits**",
				common.FormatBalance(result.BetAmount),
				percentage,
				common.FormatBalance(result.BetAmount),
			),
			Inline: false,
		},
	}
	// Add session PnL indicator
	pnlPercent := float64(session.SessionPnL) / float64(session.StartingBalance) * 100
	var pnlDisplay string

	if session.SessionPnL > 0 {
		pnlDisplay = fmt.Sprintf("+%s bits (+%.1f%% return, %d bets)", common.FormatBalance(session.SessionPnL), pnlPercent, session.BetCount)
	} else if session.SessionPnL < 0 {
		pnlDisplay = fmt.Sprintf("-%s bits (%.1f%% return, %d bets)", common.FormatBalance(-session.SessionPnL), pnlPercent, session.BetCount)
	} else {
		pnlDisplay = fmt.Sprintf("0 bits (0.0%% return, %d bets)", session.BetCount)
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Session PnL",
		Value:  pnlDisplay,
		Inline: true,
	})

	// Add encouraging message based on remaining balance
	var footerText string
	if result.NewBalance > result.BetAmount {
		footerText = ""
	} else if result.NewBalance > 0 {
		footerText = "Don't give up you filthy degen!"
	} else {
		footerText = "Yikes."
	}

	return &discordgo.MessageEmbed{
		Description: fmt.Sprintf("**LOSE**\nBalance: %s", common.FormatBalance(result.NewBalance)),
		Color:       common.ColorDanger,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
	}
}
