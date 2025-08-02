package entities

import (
	"fmt"
	"strings"
	"time"
)

// Summoner represents a League of Legends summoner with tag line information
type Summoner struct {
	ID           int64     `db:"id"`
	SummonerName string    `db:"summoner_name"`
	TagLine      string    `db:"tag_line"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// GetFullName returns the full summoner name in "SummonerName#TagLine" format
func (s *Summoner) GetFullName() string {
	return fmt.Sprintf("%s#%s", s.SummonerName, s.TagLine)
}

// IsValid checks if the summoner has valid name and tag line
func (s *Summoner) IsValid() bool {
	return strings.TrimSpace(s.SummonerName) != "" && strings.TrimSpace(s.TagLine) != ""
}

// Matches checks if this summoner matches the given name and tag line
func (s *Summoner) Matches(summonerName, tagLine string) bool {
	return strings.EqualFold(s.SummonerName, summonerName) && strings.EqualFold(s.TagLine, tagLine)
}