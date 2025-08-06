package infrastructure

import (
	"fmt"
	"gambler/discord-client/application/dto"
	events "gambler/discord-client/proto/events"
)

// ProtobufToTFTAdapter converts protobuf messages to TFT domain DTOs
// This adapter isolates protobuf dependencies from the application layer
type ProtobufToTFTAdapter struct{}

// NewProtobufToTFTAdapter creates a new protobuf to TFT adapter
func NewProtobufToTFTAdapter() *ProtobufToTFTAdapter {
	return &ProtobufToTFTAdapter{}
}

// ConvertGameStateChanged converts a protobuf TFTGameStateChanged event to domain DTOs
// Returns either a TFTGameStartedDTO or TFTGameEndedDTO based on the state transition
func (a *ProtobufToTFTAdapter) ConvertGameStateChanged(event *events.TFTGameStateChanged) (interface{}, error) {
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
func (a *ProtobufToTFTAdapter) isGameStart(event *events.TFTGameStateChanged) bool {
	return event.PreviousStatus == events.TFTGameStatus_TFT_GAME_STATUS_NOT_IN_GAME &&
		event.CurrentStatus == events.TFTGameStatus_TFT_GAME_STATUS_IN_GAME
}

// isGameEnd checks if the event represents a game ending
func (a *ProtobufToTFTAdapter) isGameEnd(event *events.TFTGameStateChanged) bool {
	return event.PreviousStatus == events.TFTGameStatus_TFT_GAME_STATUS_IN_GAME &&
		event.CurrentStatus == events.TFTGameStatus_TFT_GAME_STATUS_NOT_IN_GAME
}

// convertToGameStarted converts protobuf event to TFTGameStartedDTO
func (a *ProtobufToTFTAdapter) convertToGameStarted(event *events.TFTGameStateChanged) dto.TFTGameStartedDTO {
	return dto.TFTGameStartedDTO{
		GameID:       event.GameId,
		SummonerName: event.GameName,
		TagLine:      event.TagLine,
		QueueType:    event.QueueType,
		EventTime:    event.EventTime.AsTime(),
	}
}

// convertToGameEnded converts protobuf event to TFTGameEndedDTO
func (a *ProtobufToTFTAdapter) convertToGameEnded(event *events.TFTGameStateChanged) (dto.TFTGameEndedDTO, error) {
	if event.GameResult == nil {
		return dto.TFTGameEndedDTO{}, fmt.Errorf("game ended without result data for summoner %s#%s",
			event.GameName, event.TagLine)
	}

	return dto.TFTGameEndedDTO{
		GameID:          event.GameId,
		SummonerName:    event.GameName,
		TagLine:         event.TagLine,
		Placement:       event.GameResult.Placement,
		DurationSeconds: event.GameResult.DurationSeconds,
		QueueType:       event.QueueType,
		EventTime:       event.EventTime.AsTime(),
	}, nil
}
