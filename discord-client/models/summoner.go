package models

import (
	"time"
)

// Summoner represents a League of Legends summoner with tag line information
type Summoner struct {
	ID           int64     `db:"id"`
	SummonerName string    `db:"game_name"`
	TagLine      string    `db:"tag_line"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
