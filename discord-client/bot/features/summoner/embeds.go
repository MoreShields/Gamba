package summoner

import (
	"fmt"
	"time"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/models"
	summoner_pb "gambler/discord-client/proto/services"

	"github.com/bwmarrin/discordgo"
)

// createSuccessEmbed creates a success embed for a newly tracked summoner
func createSuccessEmbed(watchDetail *models.SummonerWatchDetail, summonerDetails *summoner_pb.SummonerDetails) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "✅ Summoner Tracking Started",
		Color: common.ColorSuccess,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Summoner",
				Value:  fmt.Sprintf("%s#%s", watchDetail.SummonerName, watchDetail.TagLine),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "You will receive notifications about this summoner's activity",
	}

	return embed
}

// createErrorEmbed creates an error embed for validation failures
func createErrorEmbed(summonerName, tagLine, errorMessage string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "❌ Failed to Track Summoner",
		Description: fmt.Sprintf("Could not start tracking **%s #%s**", summonerName, tagLine),
		Color:       common.ColorError,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Error",
				Value: errorMessage,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Please check the summoner name and tag, then try again",
		},
	}
}

// createAlreadyWatchingEmbed creates an embed for when a summoner is already being watched
func createAlreadyWatchingEmbed(summonerName, tagLine string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "ℹ️ Already Tracking Summoner",
		Description: fmt.Sprintf("**%s #%s** is already being tracked for this server.", summonerName, tagLine),
		Color:       common.ColorInfo,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "No action needed - tracking continues",
		},
	}
}

// createUnwatchSuccessEmbed creates a success embed for a summoner that was successfully unwatched
func createUnwatchSuccessEmbed(summonerName, tagLine string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "✅ Summoner Tracking Stopped",
		Description: fmt.Sprintf("No longer tracking **%s#%s** for this server.", summonerName, tagLine),
		Color:       common.ColorSuccess,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Summoner",
				Value:  summonerName,
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "You will no longer receive notifications about this summoner",
		},
	}
}

// createNotWatchingEmbed creates an embed for when a summoner is not being tracked
func createNotWatchingEmbed(summonerName, tagLine string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "ℹ️ Not Tracking Summoner",
		Description: fmt.Sprintf("**%s#%s** is not being tracked for this server.", summonerName, tagLine),
		Color:       common.ColorInfo,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /summoner watch to start tracking",
		},
	}
}
