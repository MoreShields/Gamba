package infrastructure

import (
	"fmt"
	"gambler/discord-client/application/dto"
	events "gambler/discord-client/proto/events"
)

// ProtobufToLoLAdapter converts protobuf messages to domain DTOs
// This adapter isolates protobuf dependencies from the application layer
type ProtobufToLoLAdapter struct{}

// NewProtobufToLoLAdapter creates a new protobuf to LoL adapter
func NewProtobufToLoLAdapter() *ProtobufToLoLAdapter {
	return &ProtobufToLoLAdapter{}
}

// ConvertGameStateChanged converts a protobuf LoLGameStateChanged event to domain DTOs
// Returns either a GameStartedDTO or GameEndedDTO based on the state transition
func (a *ProtobufToLoLAdapter) ConvertGameStateChanged(event *events.LoLGameStateChanged) (interface{}, error) {
	switch {
	case a.isGameStart(event):
		return a.convertToGameStarted(event), nil
	case a.isGameEnd(event):
		return a.convertToGameEnded(event)
	default:
		return nil, fmt.Errorf("unhandled state transition: %s -> %s", 
			event.PreviousStatus, event.CurrentStatus)
	}
}

// isGameStart checks if the event represents a game starting
func (a *ProtobufToLoLAdapter) isGameStart(event *events.LoLGameStateChanged) bool {
	return event.PreviousStatus == events.GameStatus_GAME_STATUS_NOT_IN_GAME &&
		event.CurrentStatus == events.GameStatus_GAME_STATUS_IN_GAME
}

// isGameEnd checks if the event represents a game ending
func (a *ProtobufToLoLAdapter) isGameEnd(event *events.LoLGameStateChanged) bool {
	return event.PreviousStatus == events.GameStatus_GAME_STATUS_IN_GAME &&
		event.CurrentStatus == events.GameStatus_GAME_STATUS_NOT_IN_GAME
}

// convertToGameStarted converts protobuf event to GameStartedDTO
func (a *ProtobufToLoLAdapter) convertToGameStarted(event *events.LoLGameStateChanged) dto.GameStartedDTO {
	return dto.GameStartedDTO{
		GameID:       event.GetGameId(),
		SummonerName: event.GameName,
		TagLine:      event.TagLine,
		QueueType:    event.GetQueueType(),
		EventTime:    event.EventTime.AsTime(),
	}
}

// convertToGameEnded converts protobuf event to GameEndedDTO
func (a *ProtobufToLoLAdapter) convertToGameEnded(event *events.LoLGameStateChanged) (dto.GameEndedDTO, error) {
	if event.GameResult == nil {
		return dto.GameEndedDTO{}, fmt.Errorf("game ended without result data for summoner %s#%s", 
			event.GameName, event.TagLine)
	}

	return dto.GameEndedDTO{
		GameID:          event.GetGameId(),
		SummonerName:    event.GameName,
		TagLine:         event.TagLine,
		Won:             event.GameResult.Won,
		DurationSeconds: event.GameResult.DurationSeconds,
		QueueType:       event.GameResult.QueueType,
		ChampionPlayed:  event.GameResult.ChampionPlayed,
		EventTime:       event.EventTime.AsTime(),
	}, nil
}