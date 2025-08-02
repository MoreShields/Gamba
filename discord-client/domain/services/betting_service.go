package services

import (
	"errors"
	"math/rand"

	"gambler/discord-client/domain/entities"
)

// BettingService contains pure business logic for betting operations
type BettingService struct{}

// NewBettingService creates a new BettingService
func NewBettingService() *BettingService {
	return &BettingService{}
}

// BetParameters contains the parameters for placing a bet
type BetParameters struct {
	Amount        int64
	Probability   float64
	CurrentBalance int64
	AvailableBalance int64
}

// BetResult contains the result of a bet calculation
type BetResult struct {
	Won           bool
	WinAmount     int64
	ChangeAmount  int64
	NewBalance    int64
	TransactionType entities.TransactionType
}

// ValidateBetParameters validates betting parameters
func (s *BettingService) ValidateBetParameters(params BetParameters) error {
	if params.Probability <= 0 || params.Probability >= 1 {
		return errors.New("win probability must be between 0 and 1 (exclusive)")
	}
	
	if params.Amount <= 0 {
		return errors.New("bet amount must be positive")
	}
	
	if params.AvailableBalance < params.Amount {
		return errors.New("insufficient available balance")
	}
	
	return nil
}

// CheckDailyLimit validates if a bet amount would exceed the daily limit
func (s *BettingService) CheckDailyLimit(betAmount, dailyRisk, dailyLimit int64) (remainingLimit int64, err error) {
	if dailyRisk+betAmount > dailyLimit {
		remainingLimit = dailyLimit - dailyRisk
		if remainingLimit <= 0 {
			return 0, errors.New("daily gambling limit reached")
		}
		return remainingLimit, errors.New("bet amount would exceed daily limit")
	}
	
	return dailyLimit - dailyRisk, nil
}

// CalculateWinAmount calculates the potential win amount based on probability
func (s *BettingService) CalculateWinAmount(betAmount int64, winProbability float64) int64 {
	// If you bet X at probability P, you win X * ((1-P)/P) on success
	return int64(float64(betAmount) * ((1 - winProbability) / winProbability))
}

// ProcessBet processes a bet and returns the result
func (s *BettingService) ProcessBet(params BetParameters) (*BetResult, error) {
	if err := s.ValidateBetParameters(params); err != nil {
		return nil, err
	}
	
	winAmount := s.CalculateWinAmount(params.Amount, params.Probability)
	
	// Roll the dice
	won := rand.Float64() < params.Probability
	
	result := &BetResult{
		Won:       won,
		WinAmount: winAmount,
	}
	
	if won {
		result.NewBalance = params.CurrentBalance + winAmount
		result.ChangeAmount = winAmount
		result.TransactionType = entities.TransactionTypeBetWin
	} else {
		result.NewBalance = params.CurrentBalance - params.Amount
		result.ChangeAmount = -params.Amount
		result.TransactionType = entities.TransactionTypeBetLoss
	}
	
	return result, nil
}

// CalculateROI calculates the return on investment for a bet
func (s *BettingService) CalculateROI(betAmount, netProfit int64) float64 {
	if betAmount == 0 {
		return 0
	}
	return (float64(netProfit) / float64(betAmount)) * 100
}

// CalculateMultiplier calculates the payout multiplier for a bet
func (s *BettingService) CalculateMultiplier(betAmount, winAmount int64) float64 {
	if betAmount == 0 {
		return 0
	}
	return float64(winAmount) / float64(betAmount)
}

// IsWinningBet checks if a bet result represents a win
func (s *BettingService) IsWinningBet(result *BetResult) bool {
	return result.Won && result.ChangeAmount > 0
}

// CalculateHouseEdge calculates the house edge for given odds
func (s *BettingService) CalculateHouseEdge(probability float64, payoutMultiplier float64) float64 {
	expectedReturn := probability * payoutMultiplier
	return 1.0 - expectedReturn
}

// ValidateWinProbabilityRange ensures win probability is within acceptable bounds
func (s *BettingService) ValidateWinProbabilityRange(probability float64, minProb, maxProb float64) error {
	if probability < minProb {
		return errors.New("win probability too low")
	}
	if probability > maxProb {
		return errors.New("win probability too high")  
	}
	return nil
}