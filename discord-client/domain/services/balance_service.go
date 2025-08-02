package services

import (
	"errors"
	"time"

	"gambler/discord-client/domain/entities"
)

// BalanceService contains pure business logic for balance operations
type BalanceService struct{}

// NewBalanceService creates a new BalanceService
func NewBalanceService() *BalanceService {
	return &BalanceService{}
}

// BalanceChange represents a balance modification
type BalanceChange struct {
	UserID          int64
	BalanceBefore   int64
	BalanceAfter    int64
	ChangeAmount    int64
	TransactionType entities.TransactionType
	Metadata        map[string]any
}

// TransferParameters contains parameters for a balance transfer
type TransferParameters struct {
	FromUserID int64
	ToUserID   int64
	Amount     int64
	FromBalance int64
	FromAvailableBalance int64
	ToBalance  int64
}

// TransferResult contains the result of a balance transfer
type TransferResult struct {
	FromBalanceChange *BalanceChange
	ToBalanceChange   *BalanceChange
	TotalTransferred  int64
}

// ValidateTransferParameters validates transfer parameters
func (s *BalanceService) ValidateTransferParameters(params TransferParameters) error {
	if params.FromUserID == params.ToUserID {
		return errors.New("cannot transfer to yourself")
	}
	
	if params.Amount <= 0 {
		return errors.New("transfer amount must be positive")
	}
	
	if params.FromAvailableBalance < params.Amount {
		return errors.New("insufficient available balance for transfer")
	}
	
	return nil
}

// CalculateTransfer calculates the balance changes for a transfer
func (s *BalanceService) CalculateTransfer(params TransferParameters) (*TransferResult, error) {
	if err := s.ValidateTransferParameters(params); err != nil {
		return nil, err
	}
	
	fromChange := &BalanceChange{
		UserID:          params.FromUserID,
		BalanceBefore:   params.FromBalance,
		BalanceAfter:    params.FromBalance - params.Amount,
		ChangeAmount:    -params.Amount,
		TransactionType: entities.TransactionTypeTransferOut,
		Metadata: map[string]any{
			"transfer_to":     params.ToUserID,
			"transfer_amount": params.Amount,
		},
	}
	
	toChange := &BalanceChange{
		UserID:          params.ToUserID,
		BalanceBefore:   params.ToBalance,
		BalanceAfter:    params.ToBalance + params.Amount,
		ChangeAmount:    params.Amount,
		TransactionType: entities.TransactionTypeTransferIn,
		Metadata: map[string]any{
			"transfer_from":   params.FromUserID,
			"transfer_amount": params.Amount,
		},
	}
	
	return &TransferResult{
		FromBalanceChange: fromChange,
		ToBalanceChange:   toChange,
		TotalTransferred:  params.Amount,
	}, nil
}

// ValidateBalanceChange ensures a balance change is mathematically correct
func (s *BalanceService) ValidateBalanceChange(change *BalanceChange) error {
	expectedAfter := change.BalanceBefore + change.ChangeAmount
	if change.BalanceAfter != expectedAfter {
		return errors.New("balance calculation is inconsistent")
	}
	
	if change.ChangeAmount == 0 {
		return errors.New("change amount cannot be zero")
	}
	
	return nil
}

// CalculateNewBalance computes the new balance after a change
func (s *BalanceService) CalculateNewBalance(currentBalance, changeAmount int64) int64 {
	return currentBalance + changeAmount
}

// CalculateNewAvailableBalance computes available balance after accounting for pending amounts
func (s *BalanceService) CalculateNewAvailableBalance(totalBalance, pendingAmount int64) int64 {
	available := totalBalance - pendingAmount
	if available < 0 {
		return 0
	}
	return available
}

// CreateBalanceHistory creates a balance history entry from a balance change
func (s *BalanceService) CreateBalanceHistory(change *BalanceChange, guildID int64, relatedID *int64, relatedType *entities.RelatedType) *entities.BalanceHistory {
	return &entities.BalanceHistory{
		DiscordID:           change.UserID,
		GuildID:             guildID,
		BalanceBefore:       change.BalanceBefore,
		BalanceAfter:        change.BalanceAfter,
		ChangeAmount:        change.ChangeAmount,
		TransactionType:     change.TransactionType,
		TransactionMetadata: change.Metadata,
		RelatedID:           relatedID,
		RelatedType:         relatedType,
		CreatedAt:           time.Now(),
	}
}

// ValidateMinimumBalance ensures balance doesn't go below minimum
func (s *BalanceService) ValidateMinimumBalance(newBalance, minimumBalance int64) error {
	if newBalance < minimumBalance {
		return errors.New("balance cannot go below minimum threshold")
	}
	return nil
}

// CalculatePercentageChange calculates the percentage change in balance
func (s *BalanceService) CalculatePercentageChange(balanceBefore, balanceAfter int64) float64 {
	if balanceBefore == 0 {
		if balanceAfter > 0 {
			return 100.0 // 100% increase from zero
		}
		return 0.0
	}
	
	change := balanceAfter - balanceBefore
	return (float64(change) / float64(balanceBefore)) * 100.0
}

// IsSignificantChange determines if a balance change is significant
func (s *BalanceService) IsSignificantChange(changeAmount, threshold int64) bool {
	if changeAmount < 0 {
		changeAmount = -changeAmount
	}
	return changeAmount >= threshold
}

// CalculateDailyBalanceChange calculates the net balance change for a day
func (s *BalanceService) CalculateDailyBalanceChange(history []*entities.BalanceHistory, targetDate time.Time) int64 {
	var totalChange int64
	
	for _, entry := range history {
		if s.isSameDay(entry.CreatedAt, targetDate) {
			totalChange += entry.ChangeAmount
		}
	}
	
	return totalChange
}

// isSameDay checks if two times are on the same day
func (s *BalanceService) isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// ValidateInitialBalance ensures initial balance meets requirements
func (s *BalanceService) ValidateInitialBalance(initialBalance, minBalance, maxBalance int64) error {
	if initialBalance < minBalance {
		return errors.New("initial balance below minimum")
	}
	
	if initialBalance > maxBalance {
		return errors.New("initial balance above maximum")
	}
	
	return nil
}