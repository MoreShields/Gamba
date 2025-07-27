package bot

import (
	"context"
	"fmt"

	"gambler/discord-client/proto/events"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)


// publishDiscordMessage converts a Discord message and publishes it via the message publisher
func (b *Bot) publishDiscordMessage(ctx context.Context, message *discordgo.MessageCreate) error {
	// Convert Discord message to protobuf format
	discordMsg := b.convertDiscordMessage(message)

	// Determine the subject
	subject := fmt.Sprintf("discord.messages.%s.%s", message.GuildID, message.ChannelID)

	// Create the event wrapper
	event := &events.DiscordMessageEvent{
		Subject:     subject,
		Message:     discordMsg,
		PublishedAt: timestamppb.Now(),
	}

	// Serialize to protobuf
	data, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message event: %w", err)
	}

	// Publish to message bus
	if err := b.messagePublisher.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.WithFields(log.Fields{
		"subject":    subject,
		"message_id": message.ID,
		"author":     message.Author.Username,
		"size":       len(data),
	}).Debug("Published Discord message to message bus")

	return nil
}

// convertDiscordMessage converts a discordgo.MessageCreate to protobuf DiscordMessage format
func (b *Bot) convertDiscordMessage(message *discordgo.MessageCreate) *events.DiscordMessage {
	discordMsg := &events.DiscordMessage{
		Id:          message.ID,
		ChannelId:   message.ChannelID,
		GuildId:     message.GuildID,
		Content:     message.Content,
		Timestamp:   timestamppb.New(message.Timestamp),
		MessageType: int32(message.Type),
		Flags:       int32(message.Flags),
	}

	// Convert author
	if message.Author != nil {
		discordMsg.Author = &events.DiscordUser{
			Id:            message.Author.ID,
			Username:      message.Author.Username,
			Discriminator: message.Author.Discriminator,
			Bot:           message.Author.Bot,
		}
		if message.Author.Avatar != "" {
			discordMsg.Author.Avatar = &message.Author.Avatar
		}
	}

	// Convert edited timestamp
	if message.EditedTimestamp != nil {
		discordMsg.EditedTimestamp = timestamppb.New(*message.EditedTimestamp)
	}

	// Convert attachments
	for _, attachment := range message.Attachments {
		pbAttachment := &events.DiscordAttachment{
			Id:       attachment.ID,
			Filename: attachment.Filename,
			Size:     int32(attachment.Size),
			Url:      attachment.URL,
			ProxyUrl: attachment.ProxyURL,
		}
		if attachment.ContentType != "" {
			pbAttachment.ContentType = &attachment.ContentType
		}
		if attachment.Width != 0 {
			width := int32(attachment.Width)
			pbAttachment.Width = &width
		}
		if attachment.Height != 0 {
			height := int32(attachment.Height)
			pbAttachment.Height = &height
		}
		discordMsg.Attachments = append(discordMsg.Attachments, pbAttachment)
	}

	// Convert embeds (simplified)
	for _, embed := range message.Embeds {
		pbEmbed := &events.DiscordEmbed{}
		if embed.Type != "" {
			embedType := string(embed.Type)
			pbEmbed.Type = &embedType
		}
		if embed.Title != "" {
			pbEmbed.Title = &embed.Title
		}
		if embed.Description != "" {
			pbEmbed.Description = &embed.Description
		}
		if embed.URL != "" {
			pbEmbed.Url = &embed.URL
		}
		if embed.Color != 0 {
			color := int32(embed.Color)
			pbEmbed.Color = &color
		}
		discordMsg.Embeds = append(discordMsg.Embeds, pbEmbed)
	}

	// Convert message reference (replies)
	if message.MessageReference != nil {
		discordMsg.ReferencedMessage = &events.DiscordMessageReference{}
		if message.MessageReference.MessageID != "" {
			discordMsg.ReferencedMessage.MessageId = &message.MessageReference.MessageID
		}
		if message.MessageReference.ChannelID != "" {
			discordMsg.ReferencedMessage.ChannelId = &message.MessageReference.ChannelID
		}
		if message.MessageReference.GuildID != "" {
			discordMsg.ReferencedMessage.GuildId = &message.MessageReference.GuildID
		}
	}

	return discordMsg
}