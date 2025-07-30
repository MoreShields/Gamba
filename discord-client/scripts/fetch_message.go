//go:build fetch
// +build fetch

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func main() {
	var (
		token     = flag.String("token", "", "Discord bot token (or set DISCORD_TOKEN env var)")
		messageID = flag.String("message", "", "Message ID to fetch")
		channelID = flag.String("channel", "", "Channel ID where the message is located")
		output    = flag.String("output", "json", "Output format: json, go, or messageCreate")
	)
	flag.Parse()

	// Get token from env if not provided
	if *token == "" {
		*token = os.Getenv("DISCORD_TOKEN")
	}

	if *token == "" {
		log.Fatal("Discord token is required. Use -token flag or set DISCORD_TOKEN env var")
	}

	if *messageID == "" || *channelID == "" {
		log.Fatal("Both -message and -channel flags are required")
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	// Fetch the message
	message, err := dg.ChannelMessage(*channelID, *messageID)
	if err != nil {
		log.Fatalf("Error fetching message: %v", err)
	}

	switch *output {
	case "json":
		outputJSON(message)
	case "go":
		outputGoStruct(message)
	case "messageCreate":
		outputMessageCreate(message)
	default:
		log.Fatalf("Unknown output format: %s", *output)
	}
}

func outputJSON(message *discordgo.Message) {
	data, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling message to JSON: %v", err)
	}
	fmt.Println(string(data))
}

func outputGoStruct(message *discordgo.Message) {
	fmt.Printf("&discordgo.Message{\n")
	fmt.Printf("\tID:        %q,\n", message.ID)
	fmt.Printf("\tChannelID: %q,\n", message.ChannelID)
	fmt.Printf("\tGuildID:   %q,\n", message.GuildID)
	fmt.Printf("\tContent:   %q,\n", message.Content)
	fmt.Printf("\tTimestamp: %q,\n", message.Timestamp)

	if message.Author != nil {
		fmt.Printf("\tAuthor: &discordgo.User{\n")
		fmt.Printf("\t\tID:            %q,\n", message.Author.ID)
		fmt.Printf("\t\tUsername:      %q,\n", message.Author.Username)
		fmt.Printf("\t\tDiscriminator: %q,\n", message.Author.Discriminator)
		fmt.Printf("\t\tBot:           %v,\n", message.Author.Bot)
		fmt.Printf("\t},\n")
	}

	if len(message.Attachments) > 0 {
		fmt.Printf("\tAttachments: []*discordgo.MessageAttachment{\n")
		for _, att := range message.Attachments {
			fmt.Printf("\t\t{\n")
			fmt.Printf("\t\t\tID:       %q,\n", att.ID)
			fmt.Printf("\t\t\tURL:      %q,\n", att.URL)
			fmt.Printf("\t\t\tProxyURL: %q,\n", att.ProxyURL)
			fmt.Printf("\t\t\tFilename: %q,\n", att.Filename)
			fmt.Printf("\t\t\tSize:     %d,\n", att.Size)
			fmt.Printf("\t\t},\n")
		}
		fmt.Printf("\t},\n")
	}

	fmt.Printf("}\n")
}

func outputMessageCreate(message *discordgo.Message) {
	fmt.Println("// Reconstructed MessageCreate event")
	fmt.Printf("messageCreate := &discordgo.MessageCreate{\n")
	fmt.Printf("\tMessage: &discordgo.Message{\n")
	fmt.Printf("\t\tID:        %q,\n", message.ID)
	fmt.Printf("\t\tChannelID: %q,\n", message.ChannelID)
	fmt.Printf("\t\tGuildID:   %q,\n", message.GuildID)
	fmt.Printf("\t\tContent:   %q,\n", message.Content)
	fmt.Printf("\t\tTimestamp: %q,\n", message.Timestamp)

	if message.Author != nil {
		fmt.Printf("\t\tAuthor: &discordgo.User{\n")
		fmt.Printf("\t\t\tID:            %q,\n", message.Author.ID)
		fmt.Printf("\t\t\tUsername:      %q,\n", message.Author.Username)
		fmt.Printf("\t\t\tDiscriminator: %q,\n", message.Author.Discriminator)
		fmt.Printf("\t\t\tBot:           %v,\n", message.Author.Bot)
		fmt.Printf("\t\t},\n")
	}

	if message.Member != nil {
		fmt.Printf("\t\tMember: &discordgo.Member{\n")
		fmt.Printf("\t\t\tUser: &discordgo.User{\n")
		fmt.Printf("\t\t\t\tID:            %q,\n", message.Author.ID)
		fmt.Printf("\t\t\t\tUsername:      %q,\n", message.Author.Username)
		fmt.Printf("\t\t\t\tDiscriminator: %q,\n", message.Author.Discriminator)
		fmt.Printf("\t\t\t\tBot:           %v,\n", message.Author.Bot)
		fmt.Printf("\t\t\t},\n")
		if message.Member.Nick != "" {
			fmt.Printf("\t\t\tNick: %q,\n", message.Member.Nick)
		}
		if len(message.Member.Roles) > 0 {
			fmt.Printf("\t\t\tRoles: []string{%s},\n", formatStringSlice(message.Member.Roles))
		}
		fmt.Printf("\t\t},\n")
	}

	if len(message.Attachments) > 0 {
		fmt.Printf("\t\tAttachments: []*discordgo.MessageAttachment{\n")
		for _, att := range message.Attachments {
			fmt.Printf("\t\t\t{\n")
			fmt.Printf("\t\t\t\tID:       %q,\n", att.ID)
			fmt.Printf("\t\t\t\tURL:      %q,\n", att.URL)
			fmt.Printf("\t\t\t\tProxyURL: %q,\n", att.ProxyURL)
			fmt.Printf("\t\t\t\tFilename: %q,\n", att.Filename)
			fmt.Printf("\t\t\t\tSize:     %d,\n", att.Size)
			fmt.Printf("\t\t\t},\n")
		}
		fmt.Printf("\t\t},\n")
	}

	fmt.Printf("\t},\n")
	fmt.Printf("}\n")
}

func formatStringSlice(slice []string) string {
	quoted := make([]string, len(slice))
	for i, s := range slice {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}
