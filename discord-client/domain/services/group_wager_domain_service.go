package services

import (
	"errors"
	"time"

	"gambler/discord-client/domain/entities"
)

// GroupWagerService contains pure business logic for group wager operations
type GroupWagerService struct{}

// NewGroupWagerDomainService creates a new GroupWagerService
func NewGroupWagerDomainService() *GroupWagerService {
	return &GroupWagerService{}
}

// GroupWagerCreationParams contains parameters for creating a group wager
type GroupWagerCreationParams struct {
	Condition           string
	Options             []string
	VotingPeriodMinutes int
	MinParticipants     int
	WagerType           entities.GroupWagerType
}

// GroupWagerResolutionResult contains the result of resolving a group wager
type GroupWagerResolutionResult struct {
	WinningOption *entities.GroupWagerOption
	Winners       []*entities.GroupWagerParticipant
	Losers        []*entities.GroupWagerParticipant
	PayoutDetails map[int64]int64 // Discord ID -> payout amount
	TotalPot      int64
}

// ValidateGroupWagerCreation validates group wager creation parameters
func (s *GroupWagerService) ValidateGroupWagerCreation(params GroupWagerCreationParams) error {
	if params.Condition == "" {
		return errors.New("wager condition cannot be empty")
	}
	
	if len(params.Condition) > 500 {
		return errors.New("wager condition too long")
	}
	
	if len(params.Options) < 2 {
		return errors.New("must have at least 2 options")
	}
	
	if len(params.Options) > 10 {
		return errors.New("cannot have more than 10 options")
	}
	
	if params.VotingPeriodMinutes <= 0 {
		return errors.New("voting period must be positive")
	}
	
	if params.MinParticipants < 1 {
		return errors.New("minimum participants must be at least 1")
	}
	
	// Validate option texts
	for i, option := range params.Options {
		if option == "" {
			return errors.New("option text cannot be empty")
		}
		
		if len(option) > 100 {
			return errors.New("option text too long")
		}
		
		// Check for duplicates
		for j := i + 1; j < len(params.Options); j++ {
			if params.Options[j] == option {
				return errors.New("duplicate option text")
			}
		}
	}
	
	return nil
}

// CanUserPlaceBet validates if a user can place a bet on a group wager
func (s *GroupWagerService) CanUserPlaceBet(wager *entities.GroupWager, user *entities.User, amount int64, optionID int64) error {
	if wager.State != entities.GroupWagerStateActive {
		return errors.New("wager is not active")
	}
	
	if !wager.CanAcceptBets() {
		return errors.New("wager is no longer accepting bets")
	}
	
	if amount <= 0 {
		return errors.New("bet amount must be positive")
	}
	
	if !user.CanAfford(amount) {
		return errors.New("insufficient available balance")
	}
	
	// Validate option exists (would need options list to validate)
	// This would typically be validated by checking against actual options
	
	return nil
}

// CanGroupWagerBeResolved checks if a group wager can be resolved
func (s *GroupWagerService) CanGroupWagerBeResolved(wager *entities.GroupWager, resolverID *int64) error {
	if wager.State != entities.GroupWagerStatePendingResolution && wager.State != entities.GroupWagerStateActive {
		return errors.New("wager cannot be resolved in current state")
	}
	
	// If wager has expired, it should be transitioned to pending resolution first
	if wager.State == entities.GroupWagerStateActive && wager.IsVotingPeriodExpired() {
		return errors.New("wager has expired but not yet transitioned to pending resolution")
	}
	
	// Validation for who can resolve would go here
	// This might depend on business rules (admins, creators, etc.)
	
	return nil
}

// CalculatePoolWagerPayouts calculates payouts for a pool-style group wager
func (s *GroupWagerService) CalculatePoolWagerPayouts(
	wager *entities.GroupWager,
	winningOption *entities.GroupWagerOption,
	participants []*entities.GroupWagerParticipant,
) *GroupWagerResolutionResult {
	result := &GroupWagerResolutionResult{
		WinningOption: winningOption,
		PayoutDetails: make(map[int64]int64),
		TotalPot:      wager.TotalPot,
	}
	
	var winners []*entities.GroupWagerParticipant
	var losers []*entities.GroupWagerParticipant
	
	// Separate winners and losers
	for _, participant := range participants {
		if participant.OptionID == winningOption.ID {
			winners = append(winners, participant)
		} else {
			losers = append(losers, participant)
		}
	}
	
	result.Winners = winners
	result.Losers = losers
	
	// Calculate payouts for winners
	if len(winners) > 0 && winningOption.TotalAmount > 0 {
		for _, winner := range winners {
			// Payout = (participant's contribution / total winning amount) * total pot
			payout := (winner.Amount * wager.TotalPot) / winningOption.TotalAmount
			result.PayoutDetails[winner.DiscordID] = payout
		}
	}
	
	return result
}

// CalculateHouseWagerPayouts calculates payouts for a house-style group wager
func (s *GroupWagerService) CalculateHouseWagerPayouts(
	wager *entities.GroupWager,
	winningOption *entities.GroupWagerOption,
	participants []*entities.GroupWagerParticipant,
) *GroupWagerResolutionResult {
	result := &GroupWagerResolutionResult{
		WinningOption: winningOption,
		PayoutDetails: make(map[int64]int64),
		TotalPot:      wager.TotalPot,
	}
	
	var winners []*entities.GroupWagerParticipant
	var losers []*entities.GroupWagerParticipant
	
	// Separate winners and losers
	for _, participant := range participants {
		if participant.OptionID == winningOption.ID {
			winners = append(winners, participant)
		} else {
			losers = append(losers, participant)
		}
	}
	
	result.Winners = winners
	result.Losers = losers
	
	// Calculate payouts using fixed odds
	for _, winner := range winners {
		payout := int64(float64(winner.Amount) * winningOption.OddsMultiplier)
		result.PayoutDetails[winner.DiscordID] = payout
	}
	
	return result
}

// CalculateOptionOdds calculates odds for all options in a group wager
func (s *GroupWagerService) CalculateOptionOdds(
	options []*entities.GroupWagerOption,
	totalPot int64,
) map[int64]float64 {
	odds := make(map[int64]float64)
	
	for _, option := range options {
		if option.TotalAmount > 0 {
			odds[option.ID] = float64(totalPot) / float64(option.TotalAmount)
		} else {
			odds[option.ID] = 1.0 // Default odds if no bets placed
		}
	}
	
	return odds
}

// ShouldTransitionToPendingResolution checks if a wager should be transitioned
func (s *GroupWagerService) ShouldTransitionToPendingResolution(wager *entities.GroupWager) bool {
	return wager.State == entities.GroupWagerStateActive && wager.IsVotingPeriodExpired()
}

// ValidateMinimumParticipation checks if minimum participation requirements are met
func (s *GroupWagerService) ValidateMinimumParticipation(
	wager *entities.GroupWager,
	participantCount int,
) error {
	if !wager.HasMinimumParticipants(participantCount) {
		return errors.New("minimum participation requirements not met")
	}
	return nil
}

// ValidateMultipleOptionsWithParticipants ensures at least 2 options have participants
func (s *GroupWagerService) ValidateMultipleOptionsWithParticipants(
	participants []*entities.GroupWagerParticipant,
) error {
	optionsWithParticipants := make(map[int64]bool)
	
	for _, participant := range participants {
		optionsWithParticipants[participant.OptionID] = true
	}
	
	if len(optionsWithParticipants) < 2 {
		return errors.New("at least 2 options must have participants")
	}
	
	return nil
}

// CalculateVotingEndTime calculates when voting ends for a group wager
func (s *GroupWagerService) CalculateVotingEndTime(startTime time.Time, periodMinutes int) time.Time {
	return startTime.Add(time.Duration(periodMinutes) * time.Minute)
}

// IsWagerExpired checks if a group wager has expired
func (s *GroupWagerService) IsWagerExpired(wager *entities.GroupWager) bool {
	return wager.IsVotingPeriodExpired()
}