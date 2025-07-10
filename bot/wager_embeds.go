package bot

import (
	"fmt"
	"time"

	"gambler/models"

	"github.com/bwmarrin/discordgo"
)

// BuildWagerProposedEmbed creates an embed for a proposed wager
func BuildWagerProposedEmbed(wager *models.Wager, proposerName, targetName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "⚔️ Wager Proposed",
		Description: fmt.Sprintf("**%s** challenges **%s**!", proposerName, targetName),
		Color:       0xFFD700, // Gold
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount",
				Value:  FormatBalance(wager.Amount),
				Inline: true,
			},
			{
				Name:   "📜 Condition",
				Value:  wager.Condition,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("___________________________________________________\nOnly %s can respond • %d", targetName, wager.ID),
		},
		Timestamp: wager.CreatedAt.Format(time.RFC3339),
	}

	return embed
}

// BuildWagerDeclinedEmbed creates an embed for a declined wager
func BuildWagerDeclinedEmbed(wager *models.Wager, proposerName, targetName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Wager Declined",
		Description: fmt.Sprintf("**%s** declined the wager from **%s**", targetName, proposerName),
		Color:       0xFF0000, // Red
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount",
				Value:  FormatBalance(wager.Amount),
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
func BuildWagerVotingEmbed(wager *models.Wager, proposerName, targetName string, voteCounts *models.VoteCount) *discordgo.MessageEmbed {
	// Calculate vote percentages
	proposerPercent := 0
	targetPercent := 0
	if voteCounts.TotalVotes > 0 {
		proposerPercent = (voteCounts.ProposerVotes * 100) / voteCounts.TotalVotes
		targetPercent = (voteCounts.TargetVotes * 100) / voteCounts.TotalVotes
	}

	// Create vote bars
	proposerBar := createVoteBar(proposerPercent)
	targetBar := createVoteBar(targetPercent)

	embed := &discordgo.MessageEmbed{
		Title:       "🗳️ Wager Voting",
		Description: fmt.Sprintf("**%s** vs **%s**\nCommunity voting is now open!", proposerName, targetName),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount",
				Value:  FormatBalance(wager.Amount),
				Inline: true,
			},
			{
				Name:   "📊 Total Votes",
				Value:  fmt.Sprintf("%d", voteCounts.TotalVotes),
				Inline: true,
			},
			{
				Name:   "📜 Condition",
				Value:  wager.Condition,
				Inline: false,
			},
			{
				Name:   fmt.Sprintf("👤 %s", proposerName),
				Value:  fmt.Sprintf("%s\n%d votes (%d%%)", proposerBar, voteCounts.ProposerVotes, proposerPercent),
				Inline: false,
			},
			{
				Name:   fmt.Sprintf("👤 %s", targetName),
				Value:  fmt.Sprintf("%s\n%d votes (%d%%)", targetBar, voteCounts.TargetVotes, targetPercent),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Wager ID: %d • Majority wins", wager.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add majority indicator if applicable
	if voteCounts.HasMajority() {
		winnerName := proposerName
		if voteCounts.TargetVotes > voteCounts.ProposerVotes {
			winnerName = targetName
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🏆 Majority Reached",
			Value:  fmt.Sprintf("**%s** has majority! Resolving wager...", winnerName),
			Inline: false,
		})
	}

	return embed
}

// BuildWagerResolvedEmbed creates an embed for a resolved wager
func BuildWagerResolvedEmbed(wager *models.Wager, proposerName, targetName, winnerName, loserName string, finalVotes *models.VoteCount) *discordgo.MessageEmbed {
	winnerVotes := finalVotes.ProposerVotes
	loserVotes := finalVotes.TargetVotes
	if wager.WinnerDiscordID != nil && *wager.WinnerDiscordID == wager.TargetDiscordID {
		winnerVotes = finalVotes.TargetVotes
		loserVotes = finalVotes.ProposerVotes
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏆 Wager Resolved",
		Description: fmt.Sprintf("**%s** wins the wager against **%s**!", winnerName, loserName),
		Color:       0x9B59B6, // Purple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Amount Won",
				Value:  FormatBalance(wager.Amount),
				Inline: true,
			},
			{
				Name:   "🗳️ Final Vote",
				Value:  fmt.Sprintf("%d - %d", winnerVotes, loserVotes),
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

// createVoteBar creates a visual representation of vote percentage
func createVoteBar(percentage int) string {
	const barLength = 10
	filledBars := (percentage * barLength) / 100
	emptyBars := barLength - filledBars

	bar := ""
	for i := 0; i < filledBars; i++ {
		bar += "█"
	}
	for i := 0; i < emptyBars; i++ {
		bar += "░"
	}

	return bar
}

// BuildWagerListEmbed creates an embed showing a user's active wagers
func BuildWagerListEmbed(wagers []*models.Wager, userID int64, userName string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📋 Active Wagers for %s", userName),
		Color:       0x3498DB, // Blue
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
		if wager.State == models.WagerStateProposed {
			status = "⏳ Awaiting Response"
		} else if wager.State == models.WagerStateVoting {
			status = "🗳️ Voting Active"
		}

		fieldValue := fmt.Sprintf("**Role:** %s\n**Amount:** %s\n**Status:** %s\n**Condition:** %.50s...",
			role, FormatBalance(wager.Amount), status, wager.Condition)

		if len(wager.Condition) <= 50 {
			fieldValue = fmt.Sprintf("**Role:** %s\n**Amount:** %s\n**Status:** %s\n**Condition:** %s",
				role, FormatBalance(wager.Amount), status, wager.Condition)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Wager #%d", wager.ID),
			Value:  fieldValue,
			Inline: false,
		})
	}

	return embed
}
