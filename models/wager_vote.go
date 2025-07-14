package models

import (
	"time"
)

// WagerVote represents a participant vote on a wager
type WagerVote struct {
	ID               int64     `db:"id"`
	WagerID          int64     `db:"wager_id"`
	GuildID          int64     `db:"guild_id"`
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
	ProposerVoted bool
	TargetVoted   bool
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

// BothParticipantsVoted checks if both participants have cast votes
func (vc *VoteCount) BothParticipantsVoted() bool {
	return vc.ProposerVoted && vc.TargetVoted
}

// BothParticipantsAgree checks if both participants voted for the same person
func (vc *VoteCount) BothParticipantsAgree() bool {
	if !vc.BothParticipantsVoted() {
		return false
	}
	// Both participants agree if one has 2 votes and the other has 0
	return (vc.ProposerVotes == 2 && vc.TargetVotes == 0) || (vc.ProposerVotes == 0 && vc.TargetVotes == 2)
}

// GetAgreedWinnerID returns the winner ID if both participants agree, 0 otherwise
func (vc *VoteCount) GetAgreedWinnerID(proposerID, targetID int64) int64 {
	if !vc.BothParticipantsAgree() {
		return 0
	}
	return vc.GetWinnerID(proposerID, targetID)
}