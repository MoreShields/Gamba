package summoner

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/models"
	summoner_pb "gambler/api/gen/go/services"
)

// createSuccessEmbed creates a success embed for a newly tracked summoner
func createSuccessEmbed(watchDetail *models.SummonerWatchDetail, summonerDetails *summoner_pb.SummonerDetails) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "✅ Summoner Tracking Started",
		Color: common.ColorSuccess,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Summoner",
				Value:  watchDetail.SummonerName,
				Inline: true,
			},
			{
				Name:   "Region", 
				Value:  watchDetail.Region,
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	
	// Add summoner details if available from validation
	if summonerDetails != nil {
		embed.Description = fmt.Sprintf("Now tracking **%s** in **%s** for this server.", 
			summonerDetails.SummonerName, summonerDetails.Region)
		
		// Add summoner level if available
		if summonerDetails.SummonerLevel > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Level",
				Value:  fmt.Sprintf("%d", summonerDetails.SummonerLevel),
				Inline: true,
			})
		}
	} else {
		embed.Description = fmt.Sprintf("Now tracking **%s** in **%s** for this server.", 
			watchDetail.SummonerName, watchDetail.Region)
	}
	
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "You will receive notifications about this summoner's activity",
	}
	
	return embed
}

// createErrorEmbed creates an error embed for validation failures
func createErrorEmbed(summonerName, region, errorMessage string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "❌ Failed to Track Summoner",
		Description: fmt.Sprintf("Could not start tracking **%s** in **%s**", summonerName, region),
		Color:       common.ColorError,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Error",
				Value: errorMessage,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Please check the summoner name and region, then try again",
		},
	}
}

// createAlreadyWatchingEmbed creates an embed for when a summoner is already being watched
func createAlreadyWatchingEmbed(summonerName, region string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "ℹ️ Already Tracking Summoner", 
		Description: fmt.Sprintf("**%s** in **%s** is already being tracked for this server.", summonerName, region),
		Color:       common.ColorInfo,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "No action needed - tracking continues",
		},
	}
}