package lottery

import (
	"fmt"
	"strings"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"

	"github.com/bwmarrin/discordgo"
)

// CreateLotteryEmbed creates the main lottery embed for an in-progress draw
func CreateLotteryEmbed(drawInfo *interfaces.LotteryDrawInfo) *discordgo.MessageEmbed {
	draw := drawInfo.Draw

	// Build participant list (top 5)
	participantStr := "No participants yet"
	if len(drawInfo.Participants) > 0 {
		lines := make([]string, 0)
		maxShow := 5
		if len(drawInfo.Participants) < maxShow {
			maxShow = len(drawInfo.Participants)
		}
		for i := 0; i < maxShow; i++ {
			p := drawInfo.Participants[i]
			lines = append(lines, fmt.Sprintf("<@%d>: %d tickets", p.DiscordID, p.TicketCount))
		}
		if len(drawInfo.Participants) > maxShow {
			lines = append(lines, fmt.Sprintf("...and %d more", len(drawInfo.Participants)-maxShow))
		}
		participantStr = strings.Join(lines, "\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Lotto #%d - %s bits", draw.ID, common.FormatBalance(draw.TotalPot)),
		Color:       common.ColorInfo,
		Description: fmt.Sprintf("Draws <t:%d:d> <t:%d:t>", draw.DrawTime.Unix(), draw.DrawTime.Unix()),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Ticket Cost",
				Value:  common.FormatBalance(drawInfo.TicketCost),
				Inline: true,
			},
			{
				Name:   "Odds",
				Value:  fmt.Sprintf("1 in %d", draw.GetTotalNumbers()),
				Inline: true,
			},
			{
				Name:   "Participants",
				Value:  participantStr,
				Inline: false,
			},
		},
	}

	return embed
}

// CreatePurchaseConfirmationEmbed creates an ephemeral embed for purchase confirmation
func CreatePurchaseConfirmationEmbed(result *interfaces.LotteryPurchaseResult) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Tickets Purchased!",
		Color:       common.ColorSuccess,
		Description: fmt.Sprintf("You bought %d ticket(s) for %s", len(result.Tickets), common.FormatBalance(result.TotalCost)),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "New Balance",
				Value:  common.FormatBalance(result.NewBalance),
				Inline: true,
			},
		},
	}
}

// CreateDrawResultEmbed creates an embed for a completed draw
func CreateDrawResultEmbed(result *interfaces.LotteryDrawResult, draw *entities.LotteryDraw, participants []*entities.LotteryParticipantInfo) *discordgo.MessageEmbed {
	var color int
	if result.RolledOver {
		color = common.ColorWarning
	} else {
		color = common.ColorSuccess
	}

	// Build participant list (top 5)
	participantStr := "No participants"
	if len(participants) > 0 {
		lines := make([]string, 0)
		maxShow := 5
		if len(participants) < maxShow {
			maxShow = len(participants)
		}
		for i := 0; i < maxShow; i++ {
			p := participants[i]
			lines = append(lines, fmt.Sprintf("<@%d>: %d tickets", p.DiscordID, p.TicketCount))
		}
		if len(participants) > maxShow {
			lines = append(lines, fmt.Sprintf("...and %d more", len(participants)-maxShow))
		}
		participantStr = strings.Join(lines, "\n")
	}

	// Build winner section
	var winnerStr string
	if result.RolledOver {
		winnerStr = "No winner"
	} else {
		winnerMentions := make([]string, 0, len(result.Winners))
		for _, winner := range result.Winners {
			winnerMentions = append(winnerMentions, fmt.Sprintf("<@%d>", winner.DiscordID))
		}
		winningsPerWinner := result.PotAmount / int64(len(result.Winners))
		winnerStr = fmt.Sprintf("%s - %s", strings.Join(winnerMentions, ", "), common.FormatBalance(winningsPerWinner))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Lotto #%d - %s bits", draw.ID, common.FormatBalance(result.PotAmount)),
		Color:       color,
		Description: fmt.Sprintf("Drew <t:%d:d> <t:%d:t>", draw.DrawTime.Unix(), draw.DrawTime.Unix()),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Odds",
				Value:  fmt.Sprintf("1 in %d", draw.GetTotalNumbers()),
				Inline: true,
			},
			{
				Name:   "Participants",
				Value:  participantStr,
				Inline: false,
			},
			{
				Name:   "Winner",
				Value:  winnerStr,
				Inline: false,
			},
		},
	}

	return embed
}
