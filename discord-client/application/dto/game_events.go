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
	GameID            string
	SummonerName      string
	TagLine           string
	Win               bool
	Loss              bool
	DurationSeconds   int32
	QueueType         string
	ChampionPlayed    string
	EventTime         time.Time
}