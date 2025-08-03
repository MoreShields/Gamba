package dto

import "time"

// GameStartedDTO represents a game that has started
type GameStartedDTO struct {
	GameID       string
	SummonerName string
	TagLine      string
	QueueType    string
	EventTime    time.Time
}

// GameEndedDTO represents a game that has ended
type GameEndedDTO struct {
	GameID          string
	SummonerName    string
	TagLine         string
	Won             bool
	DurationSeconds int32
	QueueType       string
	ChampionPlayed  string
	EventTime       time.Time
}

// TFTGameStartedDTO represents a TFT game that has started
type TFTGameStartedDTO struct {
	GameID       string
	SummonerName string
	TagLine      string
	QueueType    string
	EventTime    time.Time
}

// TFTGameEndedDTO represents a TFT game that has ended
type TFTGameEndedDTO struct {
	GameID          string
	SummonerName    string
	TagLine         string
	Placement       int32 // 1-8 placement
	DurationSeconds int32
	QueueType       string
	EventTime       time.Time
}
