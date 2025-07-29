package bot

import (
	"context"
	"fmt"

	"gambler/discord-client/events"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// publishDiscordMessage publishes a Discord message as a domain event
func (b *Bot) publishDiscordMessage(ctx context.Context, message *discordgo.MessageCreate) error {
	// Create the domain event
	event := events.DiscordMessageEvent{
		MessageID: message.ID,
		ChannelID: message.ChannelID,
		GuildID:   message.GuildID,
		UserID:    message.Author.ID,
		Username:  message.Author.Username,
		Content:   message.Content,
		Timestamp: message.Timestamp.Unix(),
	}

	// Publish the event through the event publisher
	// The infrastructure layer will handle local handlers and NATS publishing
	if err := b.eventPublisher.Publish(event); err != nil {
		return fmt.Errorf("failed to publish Discord message event: %w", err)
	}

	log.WithFields(log.Fields{
		"message_id": message.ID,
		"guild_id":   message.GuildID,
		"channel_id": message.ChannelID,
		"author":     message.Author.Username,
	}).Debug("Published Discord message event")

	return nil
}

