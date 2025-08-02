package wagers

import (
	"fmt"
	"gambler/discord-client/bot/common"
	"time"

	"gambler/discord-client/domain/entities"

	"github.com/bwmarrin/discordgo"
)

// BuildWagerProposedEmbed creates an embed for a proposed wager
func BuildWagerProposedEmbed(wager *entities.Wager, proposerName, targetName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "⚔️ Wager Proposed",
		Description: fmt.Sprintf("**%s** challenges %s!", common.GetUserMention(wager.ProposerDiscordID), common.GetUserMention(wager.TargetDiscordID)),
		Color:       common.ColorWarning,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount",
				Value:  common.FormatBalance(wager.Amount),
				Inline: true,
			},
			{
				Name:   "📜 Condition",
				Value:  wager.Condition,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("___________________________________________________\nID: %d • Only %s can respond", wager.ID, targetName),
		},
		Timestamp: wager.CreatedAt.Format(time.RFC3339),
	}

	return embed
}

// BuildWagerDeclinedEmbed creates an embed for a declined wager
func BuildWagerDeclinedEmbed(wager *entities.Wager, proposerName, targetName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Wager Declined",
		Description: fmt.Sprintf("**%s** declined the wager from **%s**", targetName, proposerName),
		Color:       common.ColorDanger,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount",
				Value:  common.FormatBalance(wager.Amount),
				Inline: true,
			},
			{
				Name:   "📜 Condition",
				Value:  wager.Condition,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Wager ID: %d", wager.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed
}

// BuildWagerVotingEmbed creates an embed for a wager in voting state
func BuildWagerVotingEmbed(wager *entities.Wager, proposerName, targetName string, voteCounts *entities.VoteCount) *discordgo.MessageEmbed {
	// Determine voting status for each participant
	proposerStatus := "⏳ Pending"
	targetStatus := "⏳ Pending"

	if voteCounts.ProposerVoted {
		if voteCounts.ProposerVotes > 0 {
			proposerStatus = fmt.Sprintf("✅ Voted for %s", proposerName)
		} else {
			proposerStatus = fmt.Sprintf("✅ Voted for %s", targetName)
		}
	}

	if voteCounts.TargetVoted {
		if voteCounts.TargetVotes > 0 {
			targetStatus = fmt.Sprintf("✅ Voted for %s", targetName)
		} else {
			targetStatus = fmt.Sprintf("✅ Voted for %s", proposerName)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🗳️ Wager Voting",
		Description: fmt.Sprintf("**%s** vs **%s** \n💰 %s bits\n", proposerName, targetName, common.FormatBalance(wager.Amount)),
		Color:       common.ColorSuccess,
		Fields: []*discordgo.MessageEmbedField{

			{
				Name:   "📜 Condition",
				Value:  wager.Condition,
				Inline: false,
			},
			{
				Name:   fmt.Sprintf("👤 %s", proposerName),
				Value:  proposerStatus,
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("👤 %s", targetName),
				Value:  targetStatus,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Wager ID: %d • Both participants must agree", wager.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add agreement indicator if both participants agree
	if voteCounts.BothParticipantsAgree() {
		winnerName := proposerName
		if voteCounts.TargetVotes > voteCounts.ProposerVotes {
			winnerName = targetName
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🏆 Both Participants Agree",
			Value:  fmt.Sprintf("**%s** wins!", winnerName),
			Inline: false,
		})
	} else if voteCounts.BothParticipantsVoted() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚖️ Participants Disagree",
			Value:  "Both participants have voted but disagree on the winner. Wager remains open.",
			Inline: false,
		})
	}

	return embed
}

// BuildWagerResolvedEmbed creates an embed for a resolved wager
func BuildWagerResolvedEmbed(wager *entities.Wager, proposerName, targetName, winnerName, loserName string, finalVotes *entities.VoteCount) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "🏆 Wager Resolved",
		Description: fmt.Sprintf("**%s** wins the wager against **%s**!", winnerName, loserName),
		Color:       common.ColorPrimary,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount Won",
				Value:  common.FormatBalance(wager.Amount),
				Inline: true,
			},
			{
				Name:   "📜 Condition",
				Value:  wager.Condition,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Wager ID: %d", wager.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed
}

// BuildWagerListEmbed creates an embed showing a user's active wagers
func BuildWagerListEmbed(wagers []*entities.Wager, userID int64, userName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📋 Active Wagers for %s", userName),
		Color:       common.ColorPrimary,
		Description: fmt.Sprintf("You have %d active wager(s)", len(wagers)),
		Fields:      []*discordgo.MessageEmbedField{},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use buttons to interact with wagers",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if len(wagers) == 0 {
		embed.Description = "You have no active wagers"
		return embed
	}

	for i, wager := range wagers {
		if i >= 10 { // Limit to 10 wagers in embed
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "...",
				Value:  fmt.Sprintf("And %d more wager(s)", len(wagers)-10),
				Inline: false,
			})
			break
		}

		role := "Proposer"
		if wager.TargetDiscordID == userID {
			role = "Target"
		}

		status := string(wager.State)
		if wager.State == entities.WagerStateProposed {
			status = "⏳ Awaiting Response"
		} else if wager.State == entities.WagerStateVoting {
			status = "🗳️ Voting Active"
		}

		fieldValue := fmt.Sprintf("**Role:** %s\n**Amount:** %s\n**Status:** %s\n**Condition:** %.50s...",
			role, common.FormatBalance(wager.Amount), status, wager.Condition)

		if len(wager.Condition) <= 50 {
			fieldValue = fmt.Sprintf("**Role:** %s\n**Amount:** %s\n**Status:** %s\n**Condition:** %s",
				role, common.FormatBalance(wager.Amount), status, wager.Condition)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Wager #%d", wager.ID),
			Value:  fieldValue,
			Inline: false,
		})
	}

	return embed
}
