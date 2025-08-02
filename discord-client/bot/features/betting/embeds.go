package betting

import (
	"fmt"
	"gambler/discord-client/bot/common"

	"gambler/discord-client/domain/entities"

	"github.com/bwmarrin/discordgo"
)

// buildInitialBetEmbed creates the initial betting interface embed
func buildInitialBetEmbed(balance int64, remaining int64) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🎰 **Place Your Bet** 🎰",
		Description: fmt.Sprintf("Current Balance: **%s bits**\nRemaining Daily limit: **%s**", common.FormatBalance(balance), common.FormatBalance(remaining)),
		Color:       common.ColorPrimary,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Select your win probability",
		},
	}
}

// buildWinEmbed creates the embed for a winning bet
func buildWinEmbed(result *entities.BetResult, odds float64, session *BetSession, userID int64) *discordgo.MessageEmbed {
	percentage := int(odds * 100)

	fields := []*discordgo.MessageEmbedField{
		{
			Name: "Bet Details",
			Value: fmt.Sprintf("• Bet: **%s bits** at %d%% odds\n",
				common.FormatBalance(result.BetAmount),
				percentage,
			),
			Inline: false,
		},
	}

	// Add session PnL indicator
	pnlPercent := float64(session.SessionPnL) / float64(session.StartingBalance) * 100
	var pnlDisplay string

	if session.SessionPnL > 0 {
		pnlDisplay = fmt.Sprintf("+%s (+%.1f%% return, %d bets)", common.FormatBalance(session.SessionPnL), pnlPercent, session.BetCount)
	} else if session.SessionPnL < 0 {
		pnlDisplay = fmt.Sprintf("-%s (%.1f%% return, %d bets)", common.FormatBalance(-session.SessionPnL), pnlPercent, session.BetCount)
	} else {
		pnlDisplay = fmt.Sprintf("0 (0.0%% return, %d bets)", session.BetCount)
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Session PnL",
		Value:  pnlDisplay,
		Inline: true,
	})

	return &discordgo.MessageEmbed{
		Description: fmt.Sprintf("🎉 **<@%d>** 🎉\nBalance: **%s bits**", userID, common.FormatBalance(session.CurrentBalance)),
		Color:       common.ColorSuccess,
		Fields:      fields,
	}
}

// buildLossEmbed creates the embed for a losing bet
func buildLossEmbed(result *entities.BetResult, odds float64, session *BetSession, userID int64) *discordgo.MessageEmbed {
	percentage := int(odds * 100)

	fields := []*discordgo.MessageEmbedField{
		{
			Name: "",
			Value: fmt.Sprintf("• Bet: **%s** at %d%% odds",
				common.FormatBalance(result.BetAmount),
				percentage,
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
		Description: fmt.Sprintf("🤡 **<@%d>** 🤡\nBalance: %s", userID, common.FormatBalance(session.CurrentBalance)),
		Color:       common.ColorDanger,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
	}
}
