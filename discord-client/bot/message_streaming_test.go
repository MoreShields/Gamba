package bot

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/proto/events"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// MockMessagePublisher implements infrastructure.MessagePublisher for testing
type MockMessagePublisher struct {
	PublishedMessages []PublishedMessage
}

type PublishedMessage struct {
	Subject string
	Data    []byte
}

func (m *MockMessagePublisher) Publish(ctx context.Context, subject string, data []byte) error {
	m.PublishedMessages = append(m.PublishedMessages, PublishedMessage{
		Subject: subject,
		Data:    data,
	})
	return nil
}

func TestPublishDiscordMessage(t *testing.T) {
	tests := []struct {
		name            string
		discordMessage  *discordgo.MessageCreate
		expectedSubject string
		validateEvent   func(t *testing.T, event *events.DiscordMessageEvent)
	}{
		{
			name: "basic message",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg123",
					ChannelID: "channel456",
					GuildID:   "guild789",
					Content:   "Hello, world!",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Type:      discordgo.MessageTypeDefault,
					Flags:     64,
					Author: &discordgo.User{
						ID:            "user123",
						Username:      "testuser",
						Discriminator: "1234",
						Avatar:        "avatar_hash",
						Bot:           false,
					},
				},
			},
			expectedSubject: "discord.messages.guild789.channel456",
			validateEvent: func(t *testing.T, event *events.DiscordMessageEvent) {
				msg := event.Message
				assert.Equal(t, "msg123", msg.Id)
				assert.Equal(t, "channel456", msg.ChannelId)
				assert.Equal(t, "guild789", msg.GuildId)
				assert.Equal(t, "Hello, world!", msg.Content)
				assert.Equal(t, int32(discordgo.MessageTypeDefault), msg.MessageType)
				assert.Equal(t, int32(64), msg.Flags)

				// Verify timestamp
				assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), msg.Timestamp.AsTime())
				assert.Nil(t, msg.EditedTimestamp)

				// Verify author
				require.NotNil(t, msg.Author)
				assert.Equal(t, "user123", msg.Author.Id)
				assert.Equal(t, "testuser", msg.Author.Username)
				assert.Equal(t, "1234", msg.Author.Discriminator)
				assert.Equal(t, "avatar_hash", *msg.Author.Avatar)
				assert.False(t, msg.Author.Bot)

				// Verify no attachments/embeds/references
				assert.Empty(t, msg.Attachments)
				assert.Empty(t, msg.Embeds)
				assert.Nil(t, msg.ReferencedMessage)
			},
		},
		{
			name: "message with attachments and embeds",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:              "msg456",
					ChannelID:       "channel789",
					GuildID:         "guild123",
					Content:         "Check this out!",
					Timestamp:       time.Date(2023, 1, 2, 15, 30, 0, 0, time.UTC),
					EditedTimestamp: func() *time.Time { t := time.Date(2023, 1, 2, 15, 35, 0, 0, time.UTC); return &t }(),
					Author: &discordgo.User{
						ID:       "user456",
						Username: "author",
						Bot:      false,
					},
					Attachments: []*discordgo.MessageAttachment{
						{
							ID:          "attach1",
							Filename:    "image.png",
							Size:        1024,
							URL:         "https://cdn.discord.com/image.png",
							ProxyURL:    "https://media.discord.net/image.png",
							ContentType: "image/png",
							Width:       800,
							Height:      600,
						},
					},
					Embeds: []*discordgo.MessageEmbed{
						{
							Type:        discordgo.EmbedTypeRich,
							Title:       "Cool Embed",
							Description: "This is awesome",
							URL:         "https://example.com",
							Color:       0xFF0000,
						},
					},
				},
			},
			expectedSubject: "discord.messages.guild123.channel789",
			validateEvent: func(t *testing.T, event *events.DiscordMessageEvent) {
				msg := event.Message
				assert.Equal(t, "msg456", msg.Id)
				assert.Equal(t, "Check this out!", msg.Content)

				// Verify edited timestamp
				assert.Equal(t, time.Date(2023, 1, 2, 15, 35, 0, 0, time.UTC), msg.EditedTimestamp.AsTime())

				// Verify attachment
				require.Len(t, msg.Attachments, 1)
				attachment := msg.Attachments[0]
				assert.Equal(t, "attach1", attachment.Id)
				assert.Equal(t, "image.png", attachment.Filename)
				assert.Equal(t, int32(1024), attachment.Size)
				assert.Equal(t, "https://cdn.discord.com/image.png", attachment.Url)
				assert.Equal(t, "https://media.discord.net/image.png", attachment.ProxyUrl)
				assert.Equal(t, "image/png", *attachment.ContentType)
				assert.Equal(t, int32(800), *attachment.Width)
				assert.Equal(t, int32(600), *attachment.Height)

				// Verify embed
				require.Len(t, msg.Embeds, 1)
				embed := msg.Embeds[0]
				assert.Equal(t, "rich", *embed.Type)
				assert.Equal(t, "Cool Embed", *embed.Title)
				assert.Equal(t, "This is awesome", *embed.Description)
				assert.Equal(t, "https://example.com", *embed.Url)
				assert.Equal(t, int32(0xFF0000), *embed.Color)
			},
		},
		{
			name: "message reply with reference",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "reply123",
					ChannelID: "channel999",
					GuildID:   "guild888",
					Content:   "Great point!",
					Timestamp: time.Date(2023, 1, 3, 9, 15, 0, 0, time.UTC),
					Author: &discordgo.User{
						ID:       "user789",
						Username: "replier",
						Bot:      false,
					},
					MessageReference: &discordgo.MessageReference{
						MessageID: "original123",
						ChannelID: "channel999",
						GuildID:   "guild888",
					},
				},
			},
			expectedSubject: "discord.messages.guild888.channel999",
			validateEvent: func(t *testing.T, event *events.DiscordMessageEvent) {
				msg := event.Message
				assert.Equal(t, "reply123", msg.Id)
				assert.Equal(t, "Great point!", msg.Content)

				// Verify message reference
				require.NotNil(t, msg.ReferencedMessage)
				assert.Equal(t, "original123", *msg.ReferencedMessage.MessageId)
				assert.Equal(t, "channel999", *msg.ReferencedMessage.ChannelId)
				assert.Equal(t, "guild888", *msg.ReferencedMessage.GuildId)
			},
		},
		{
			name: "bot message with minimal fields",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "bot123",
					ChannelID: "channel111",
					GuildID:   "guild222",
					Content:   "",
					Timestamp: time.Date(2023, 1, 4, 10, 0, 0, 0, time.UTC),
					Author: &discordgo.User{
						ID:       "bot456",
						Username: "coolbot",
						Avatar:   "", // Empty avatar
						Bot:      true,
					},
				},
			},
			expectedSubject: "discord.messages.guild222.channel111",
			validateEvent: func(t *testing.T, event *events.DiscordMessageEvent) {
				msg := event.Message
				assert.Equal(t, "bot123", msg.Id)
				assert.Equal(t, "", msg.Content)

				// Verify bot author with minimal fields
				require.NotNil(t, msg.Author)
				assert.Equal(t, "bot456", msg.Author.Id)
				assert.Equal(t, "coolbot", msg.Author.Username)
				assert.True(t, msg.Author.Bot)
				assert.Nil(t, msg.Author.Avatar) // Empty string becomes nil

				// Verify no optional content
				assert.Nil(t, msg.EditedTimestamp)
				assert.Empty(t, msg.Attachments)
				assert.Empty(t, msg.Embeds)
				assert.Nil(t, msg.ReferencedMessage)
			},
		},
		{
			name: "attachment without optional fields",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "file123",
					ChannelID: "channel333",
					GuildID:   "guild444",
					Content:   "Here's a file",
					Timestamp: time.Date(2023, 1, 5, 14, 20, 0, 0, time.UTC),
					Author: &discordgo.User{
						ID:       "user999",
						Username: "uploader",
						Bot:      false,
					},
					Attachments: []*discordgo.MessageAttachment{
						{
							ID:       "file1",
							Filename: "document.txt",
							Size:     512,
							URL:      "https://cdn.discord.com/document.txt",
							ProxyURL: "https://media.discord.net/document.txt",
							// No ContentType, Width, Height
						},
					},
				},
			},
			expectedSubject: "discord.messages.guild444.channel333",
			validateEvent: func(t *testing.T, event *events.DiscordMessageEvent) {
				msg := event.Message

				// Verify attachment with missing optional fields
				require.Len(t, msg.Attachments, 1)
				attachment := msg.Attachments[0]
				assert.Equal(t, "file1", attachment.Id)
				assert.Equal(t, "document.txt", attachment.Filename)
				assert.Equal(t, int32(512), attachment.Size)
				assert.Nil(t, attachment.ContentType)
				assert.Nil(t, attachment.Width)
				assert.Nil(t, attachment.Height)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPublisher := &MockMessagePublisher{}
			bot := &Bot{
				messagePublisher: mockPublisher,
			}

			// Publish the message
			ctx := context.Background()
			err := bot.publishDiscordMessage(ctx, tt.discordMessage)
			require.NoError(t, err)

			// Verify the message was published
			require.Len(t, mockPublisher.PublishedMessages, 1)
			published := mockPublisher.PublishedMessages[0]

			// Verify subject
			assert.Equal(t, tt.expectedSubject, published.Subject)

			// Deserialize and verify the protobuf message
			var event events.DiscordMessageEvent
			err = proto.Unmarshal(published.Data, &event)
			require.NoError(t, err)

			// Verify event metadata
			assert.Equal(t, tt.expectedSubject, event.Subject)
			assert.NotNil(t, event.PublishedAt)
			assert.True(t, time.Since(event.PublishedAt.AsTime()) < time.Minute)
			require.NotNil(t, event.Message)

			// Run test-specific validations
			tt.validateEvent(t, &event)
		})
	}
}
