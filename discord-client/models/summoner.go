package models

import (
	"time"
)

// Summoner represents a League of Legends summoner with region information
type Summoner struct {
	ID           int64     `db:"id"`
	SummonerName string    `db:"summoner_name"`
	Region       string    `db:"region"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}