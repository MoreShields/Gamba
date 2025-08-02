package services

import (
	"errors"
	"time"

	"gambler/discord-client/domain/entities"
)

// WagerResolutionService contains pure business logic for wager resolution
type WagerResolutionService struct{}

// NewWagerResolutionService creates a new WagerResolutionService
func NewWagerResolutionService() *WagerResolutionService {
	return &WagerResolutionService{}
}

// WagerResolutionResult represents the outcome of resolving a wager
type WagerResolutionResult struct {
	WinnerID      int64
	LoserID       int64
	WinnerPayout  int64
	LoserDeduction int64
	TotalAmount   int64
}

// CanWagerBeProposed validates if a wager can be proposed
func (s *WagerResolutionService) CanWagerBeProposed(proposer *entities.User, target *entities.User, amount int64) error {
	if proposer.DiscordID == target.DiscordID {
		return errors.New("cannot wager against yourself")
	}
	
	if amount <= 0 {
		return errors.New("wager amount must be positive")
	}
	
	if !proposer.CanAfford(amount) {
		return errors.New("proposer has insufficient available balance")
	}
	
	if !target.CanAfford(amount) {
		return errors.New("target has insufficient available balance")
	}
	
	return nil
}

// CanWagerBeAccepted checks if a wager can be accepted by the target
func (s *WagerResolutionService) CanWagerBeAccepted(wager *entities.Wager, acceptorID int64) error {
	if wager.State != entities.WagerStateProposed {
		return errors.New("wager is not in proposed state")
	}
	
	if wager.TargetDiscordID != acceptorID {
		return errors.New("only the target can accept the wager")
	}
	
	return nil
}

// CanWagerBeCancelled checks if a wager can be cancelled by the proposer
func (s *WagerResolutionService) CanWagerBeCancelled(wager *entities.Wager, cancellerID int64) error {
	if wager.State != entities.WagerStateProposed {
		return errors.New("can only cancel proposed wagers")
	}
	
	if wager.ProposerDiscordID != cancellerID {
		return errors.New("only the proposer can cancel the wager")
	}
	
	return nil
}

// CanWagerBeResolved checks if a wager can be resolved
func (s *WagerResolutionService) CanWagerBeResolved(wager *entities.Wager, resolverID int64) error {
	if wager.State != entities.WagerStateVoting {
		return errors.New("wager must be in voting state to be resolved")
	}
	
	if !wager.IsParticipant(resolverID) {
		return errors.New("only participants can resolve the wager")
	}
	
	if !s.IsWinnerValid(wager, resolverID) {
		return errors.New("invalid winner for this wager")
	}
	
	return nil
}

// IsWinnerValid checks if the proposed winner is a valid participant
func (s *WagerResolutionService) IsWinnerValid(wager *entities.Wager, winnerID int64) bool {
	return wager.IsParticipant(winnerID)
}

// CalculateWagerPayout calculates the payout for a resolved wager
func (s *WagerResolutionService) CalculateWagerPayout(wager *entities.Wager, winnerID int64) *WagerResolutionResult {
	loserID := wager.GetOpponent(winnerID)
	
	return &WagerResolutionResult{
		WinnerID:       winnerID,
		LoserID:        loserID,
		WinnerPayout:   wager.Amount * 2, // Winner gets both amounts
		LoserDeduction: wager.Amount,     // Loser loses their amount
		TotalAmount:    wager.Amount * 2,
	}
}

// CheckVotingConsensus determines if there's consensus on who won
func (s *WagerResolutionService) CheckVotingConsensus(wager *entities.Wager, voteCount *entities.VoteCount) (winner int64, hasConsensus bool) {
	// Both participants must agree for consensus
	if !voteCount.BothParticipantsAgree() {
		return 0, false
	}
	
	return voteCount.GetAgreedWinnerID(wager.ProposerDiscordID, wager.TargetDiscordID), true
}

// IsVotingPeriodActive checks if voting is still active for a wager
func (s *WagerResolutionService) IsVotingPeriodActive(wager *entities.Wager, votingDurationMinutes int) bool {
	if wager.AcceptedAt == nil {
		return false
	}
	
	votingEndTime := wager.AcceptedAt.Add(time.Duration(votingDurationMinutes) * time.Minute)
	return time.Now().Before(votingEndTime)
}

// CalculateVotingDeadline calculates when voting ends for a wager
func (s *WagerResolutionService) CalculateVotingDeadline(acceptedAt time.Time, votingDurationMinutes int) time.Time {
	return acceptedAt.Add(time.Duration(votingDurationMinutes) * time.Minute)
}

// ValidateWagerCondition ensures the wager condition meets requirements
func (s *WagerResolutionService) ValidateWagerCondition(condition string, minLength, maxLength int) error {
	if len(condition) < minLength {
		return errors.New("wager condition too short")
	}
	
	if len(condition) > maxLength {
		return errors.New("wager condition too long")
	}
	
	if condition == "" {
		return errors.New("wager condition cannot be empty")
	}
	
	return nil
}

// DetermineWagerExpiration checks if a wager should be expired
func (s *WagerResolutionService) DetermineWagerExpiration(wager *entities.Wager, expirationHours int) bool {
	if wager.State != entities.WagerStateProposed {
		return false
	}
	
	expirationTime := wager.CreatedAt.Add(time.Duration(expirationHours) * time.Hour)
	return time.Now().After(expirationTime)
}