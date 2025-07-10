package models

import (
	"time"
)

// WagerVote represents a community vote on a wager
type WagerVote struct {
	ID               int64     `db:"id"`
	WagerID          int64     `db:"wager_id"`
	VoterDiscordID   int64     `db:"voter_discord_id"`
	VoteForDiscordID int64     `db:"vote_for_discord_id"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// VoteCount represents the vote tally for a wager
type VoteCount struct {
	ProposerVotes int
	TargetVotes   int
	TotalVotes    int
}

// GetWinnerID returns the discord ID of the current winner based on votes
// Returns 0 if tied
func (vc *VoteCount) GetWinnerID(proposerID, targetID int64) int64 {
	if vc.ProposerVotes > vc.TargetVotes {
		return proposerID
	}
	if vc.TargetVotes > vc.ProposerVotes {
		return targetID
	}
	return 0 // Tied
}

// HasMajority checks if either participant has a majority of votes
func (vc *VoteCount) HasMajority() bool {
	if vc.TotalVotes == 0 {
		return false
	}
	majorityThreshold := vc.TotalVotes/2 + 1
	return vc.ProposerVotes >= majorityThreshold || vc.TargetVotes >= majorityThreshold
}

// GetMajorityWinnerID returns the winner ID if they have a majority, 0 otherwise
func (vc *VoteCount) GetMajorityWinnerID(proposerID, targetID int64) int64 {
	if !vc.HasMajority() {
		return 0
	}
	return vc.GetWinnerID(proposerID, targetID)
}