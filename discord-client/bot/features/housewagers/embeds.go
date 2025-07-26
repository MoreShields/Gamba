package housewagers

import (
	"fmt"
	"time"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
)

// CreateHouseWagerEmbed creates an embed for a house wager
func CreateHouseWagerEmbed(houseWager dto.HouseWagerPostDTO) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       houseWager.Title,
		Description: houseWager.Description,
		Color:       common.ColorWarning, // Orange for house wagers to distinguish from group wagers
		Timestamp:   time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("House Wager ID: %d", houseWager.WagerID),
		},
	}

	// Add summoner information field with more emphasis
	summonerValue := fmt.Sprintf("**%s#%s**\n🎮 %s\n🆔 %s",
		houseWager.SummonerInfo.GameName,
		houseWager.SummonerInfo.TagLine,
		houseWager.SummonerInfo.QueueType,
		houseWager.SummonerInfo.GameID)

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "📊 Game Info",
		Value:  summonerValue,
		Inline: true, // Position alongside betting options
	})

	// Add betting options with fixed odds
	if len(houseWager.Options) > 0 {
		optionsText := ""
		for i, option := range houseWager.Options {
			emoji := getOptionEmoji(i + 1)
			optionsText += fmt.Sprintf("%s **%s** - %.2fx odds\n",
				emoji,
				option.Text,
				option.Multiplier)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🎯 Betting Options",
			Value:  optionsText,
			Inline: true, // Position alongside game info
		})
	}

	// Add voting period information if available
	if houseWager.VotingEndsAt != nil {
		if houseWager.VotingEndsAt.After(time.Now()) {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟢 Betting Open",
				Value:  fmt.Sprintf("**Ends <t:%d:R>**", houseWager.VotingEndsAt.Unix()),
				Inline: true,
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🟠 Betting Closed",
				Value:  fmt.Sprintf("**Ended <t:%d:R>**", houseWager.VotingEndsAt.Unix()),
				Inline: true,
			})
		}
	}

	return embed
}

// getOptionEmoji returns appropriate emoji for betting options
func getOptionEmoji(optionNumber int) string {
	switch optionNumber {
	case 1:
		return "🟢" // Green for Win
	case 2:
		return "🔴" // Red for Loss
	default:
		return "⚪" // White for additional options
	}
}

// formatCompactAmount formats large amounts in a compact way (e.g., 1.2M, 500K)
func formatCompactAmount(amount int64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.0fK", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}

// CreateHouseWagerResolvedEmbed creates an embed for a resolved house wager
func CreateHouseWagerResolvedEmbed(houseWager dto.HouseWagerPostDTO, winningOption string, totalPayout int64) *discordgo.MessageEmbed {
	embed := CreateHouseWagerEmbed(houseWager)
	
	// Update for resolved state
	embed.Color = common.ColorPrimary // Blue for resolved
	embed.Title = "🏆 " + embed.Title + " - RESOLVED"
	
	// Add resolution information
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🎉 Result",
		Value:  fmt.Sprintf("**%s** won!\nTotal payout: **%s bits**", winningOption, formatCompactAmount(totalPayout)),
		Inline: false,
	})

	embed.Footer.Text += " | RESOLVED"

	return embed
}