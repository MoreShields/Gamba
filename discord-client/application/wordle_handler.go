package application

import (
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/events"

	log "github.com/sirupsen/logrus"
)

// WordleHandler processes Discord messages from the Wordle bot
type WordleHandler interface {
	HandleDiscordMessage(ctx context.Context, event interface{}) error
}

// wordleHandler implements the WordleHandler interface
type wordleHandler struct {
	uowFactory   UnitOfWorkFactory
	wordleBotID  string
	rewardAmount int64
}

// NewWordleHandler creates a new WordleHandler
func NewWordleHandler(uowFactory UnitOfWorkFactory, wordleBotID string, rewardAmount int64) WordleHandler {
	return &wordleHandler{
		uowFactory:   uowFactory,
		wordleBotID:  wordleBotID,
		rewardAmount: rewardAmount,
	}
}

// HandleDiscordMessage processes Discord messages and awards bits for Wordle completions
func (h *wordleHandler) HandleDiscordMessage(ctx context.Context, event interface{}) error {
	m, err := AssertEventType[events.DiscordMessageEvent](event, "DiscordMessageEvent")
	if err != nil {
		return err
	}

	// Check if message is from the Wordle bot
	if m.UserID != h.wordleBotID {
		return nil
	}

	log.WithFields(log.Fields{
		"message_id": m.MessageID,
		"channel_id": m.ChannelID,
		"guild_id":   m.GuildID,
		"content":    m.Content,
	}).Debug("Processing Wordle bot message")

	// Parse guild ID
	_, err = strconv.ParseInt(m.GuildID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse guild ID: %w", err)
	}

	return nil
}
