package infrastructure

import (
	"context"
	"fmt"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	events "gambler/discord-client/proto/events"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// LoLEventListener handles LoL-specific NATS events and converts them to application DTOs
type LoLEventListener struct {
	lolHandler application.LoLHandler
}

// NewLoLEventListener creates a new LoL event listener
func NewLoLEventListener(lolHandler application.LoLHandler) *LoLEventListener {
	return &LoLEventListener{
		lolHandler: lolHandler,
	}
}

// HandleLoLGameStateChange processes LoL game state change events from NATS
func (l *LoLEventListener) HandleLoLGameStateChange(ctx context.Context, data []byte) error {
	// Deserialize the protobuf message
	event := &events.LoLGameStateChanged{}
	if err := proto.Unmarshal(data, event); err != nil {
		return fmt.Errorf("failed to unmarshal LoLGameStateChanged: %w", err)
	}

	log.WithFields(log.Fields{
		"summoner":       fmt.Sprintf("%s#%s", event.GameName, event.TagLine),
		"previousStatus": event.PreviousStatus,
		"currentStatus":  event.CurrentStatus,
		"gameId":         event.GameId,
	}).Debug("Processing LoL game state change")

	// Route based on state transition
	switch {
	case isGameStart(event):
		return l.handleGameStart(ctx, event)
	case isGameEnd(event):
		return l.handleGameEnd(ctx, event)
	default:
		// Ignore other transitions
		log.WithFields(log.Fields{
			"previousStatus": event.PreviousStatus,
			"currentStatus":  event.CurrentStatus,
		}).Debug("Ignoring non-relevant state transition")
		return nil
	}
}

// isGameStart checks if the event represents a game starting
func isGameStart(event *events.LoLGameStateChanged) bool {
	return event.PreviousStatus == events.GameStatus_GAME_STATUS_NOT_IN_GAME &&
		event.CurrentStatus == events.GameStatus_GAME_STATUS_IN_GAME
}

// isGameEnd checks if the event represents a game ending
func isGameEnd(event *events.LoLGameStateChanged) bool {
	return event.PreviousStatus == events.GameStatus_GAME_STATUS_IN_GAME &&
		event.CurrentStatus == events.GameStatus_GAME_STATUS_NOT_IN_GAME
}

// handleGameStart converts protobuf event to DTO and calls application layer
func (l *LoLEventListener) handleGameStart(ctx context.Context, event *events.LoLGameStateChanged) error {
	gameStartedDTO := dto.GameStartedDTO{
		GameID:       event.GetGameId(),
		SummonerName: event.GameName,
		TagLine:      event.TagLine,
		QueueType:    event.GetQueueType(),
		EventTime:    event.EventTime.AsTime(),
	}

	return l.lolHandler.HandleGameStarted(ctx, gameStartedDTO)
}

// handleGameEnd converts protobuf event to DTO and calls application layer
func (l *LoLEventListener) handleGameEnd(ctx context.Context, event *events.LoLGameStateChanged) error {
	if event.GameResult == nil {
		log.WithFields(log.Fields{
			"summoner": fmt.Sprintf("%s#%s", event.GameName, event.TagLine),
			"gameId":   event.GameId,
		}).Warn("Game ended without result data")
		return nil
	}

	gameEndedDTO := dto.GameEndedDTO{
		GameID:          event.GetGameId(),
		SummonerName:    event.GameName,
		TagLine:         event.TagLine,
		Win:             event.GameResult.Win,
		Loss:            event.GameResult.Loss,
		DurationSeconds: event.GameResult.DurationSeconds,
		QueueType:       event.GameResult.QueueType,
		ChampionPlayed:  event.GameResult.ChampionPlayed,
		EventTime:       event.EventTime.AsTime(),
	}

	return l.lolHandler.HandleGameEnded(ctx, gameEndedDTO)
}
