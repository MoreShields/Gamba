package bot

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/domain/events"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEventPublisher implements services.EventPublisher for testing
type MockEventPublisher struct {
	PublishedEvents []events.Event
}

func (m *MockEventPublisher) Publish(event events.Event) error {
	m.PublishedEvents = append(m.PublishedEvents, event)
	return nil
}

func TestPublishDiscordMessage(t *testing.T) {
	tests := []struct {
		name           string
		discordMessage *discordgo.MessageCreate
		validateEvent  func(t *testing.T, event events.DiscordMessageEvent)
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
					Author: &discordgo.User{
						ID:       "user123",
						Username: "testuser",
					},
				},
			},
			validateEvent: func(t *testing.T, event events.DiscordMessageEvent) {
				assert.Equal(t, "msg123", event.MessageID)
				assert.Equal(t, "channel456", event.ChannelID)
				assert.Equal(t, "guild789", event.GuildID)
				assert.Equal(t, "Hello, world!", event.Content)
				assert.Equal(t, "user123", event.UserID)
				assert.Equal(t, "testuser", event.Username)
				assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC).Unix(), event.Timestamp)
			},
		},
		{
			name: "message with empty content",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg456",
					ChannelID: "channel789",
					GuildID:   "guild123",
					Content:   "",
					Timestamp: time.Date(2023, 1, 2, 15, 30, 0, 0, time.UTC),
					Author: &discordgo.User{
						ID:       "user456",
						Username: "emptyuser",
					},
				},
			},
			validateEvent: func(t *testing.T, event events.DiscordMessageEvent) {
				assert.Equal(t, "msg456", event.MessageID)
				assert.Equal(t, "", event.Content)
				assert.Equal(t, "emptyuser", event.Username)
			},
		},
		{
			name: "bot message",
			discordMessage: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "bot123",
					ChannelID: "channel111",
					GuildID:   "guild222",
					Content:   "I am a bot",
					Timestamp: time.Date(2023, 1, 3, 10, 0, 0, 0, time.UTC),
					Author: &discordgo.User{
						ID:       "bot456",
						Username: "coolbot",
						Bot:      true,
					},
				},
			},
			validateEvent: func(t *testing.T, event events.DiscordMessageEvent) {
				assert.Equal(t, "bot123", event.MessageID)
				assert.Equal(t, "coolbot", event.Username)
				assert.Equal(t, "bot456", event.UserID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPublisher := &MockEventPublisher{}
			bot := &Bot{
				eventPublisher: mockPublisher,
			}

			// Publish the message
			ctx := context.Background()
			err := bot.publishDiscordMessage(ctx, tt.discordMessage)
			require.NoError(t, err)

			// Verify the event was published
			require.Len(t, mockPublisher.PublishedEvents, 1)
			publishedEvent := mockPublisher.PublishedEvents[0]

			// Verify it's a DiscordMessageEvent
			discordEvent, ok := publishedEvent.(events.DiscordMessageEvent)
			require.True(t, ok, "Expected DiscordMessageEvent")

			// Run test-specific validations
			tt.validateEvent(t, discordEvent)
		})
	}
}