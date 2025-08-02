package entities

import (
	"errors"
	"time"
)


// RelatedType represents what type of entity the related_id refers to
type RelatedType string

const (
	RelatedTypeBet        RelatedType = "bet"
	RelatedTypeWager      RelatedType = "wager"
	RelatedTypeGroupWager RelatedType = "group_wager"
)

// BalanceHistory represents a historical balance change
type BalanceHistory struct {
	ID                  int64                  `db:"id"`
	DiscordID           int64                  `db:"discord_id"`
	GuildID             int64                  `db:"guild_id"`
	BalanceBefore       int64                  `db:"balance_before"`
	BalanceAfter        int64                  `db:"balance_after"`
	ChangeAmount        int64                  `db:"change_amount"`
	TransactionType     TransactionType        `db:"transaction_type"`
	TransactionMetadata map[string]any         `db:"transaction_metadata"`
	RelatedID           *int64                 `db:"related_id"`
	RelatedType         *RelatedType           `db:"related_type"`
	CreatedAt           time.Time              `db:"created_at"`
}

// IsPositiveChange returns true if the change amount is positive
func (bh *BalanceHistory) IsPositiveChange() bool {
	return bh.ChangeAmount > 0
}

// IsNegativeChange returns true if the change amount is negative
func (bh *BalanceHistory) IsNegativeChange() bool {
	return bh.ChangeAmount < 0
}

// IsWinTransaction returns true if this is a win-type transaction
func (bh *BalanceHistory) IsWinTransaction() bool {
	return bh.TransactionType.IsWinType()
}

// IsLossTransaction returns true if this is a loss-type transaction
func (bh *BalanceHistory) IsLossTransaction() bool {
	return bh.TransactionType.IsLossType()
}

// IsTransferTransaction returns true if this is a transfer transaction
func (bh *BalanceHistory) IsTransferTransaction() bool {
	return bh.TransactionType.IsTransferType()
}

// IsGamblingTransaction returns true if this is a gambling-related transaction
func (bh *BalanceHistory) IsGamblingTransaction() bool {
	return bh.TransactionType.IsGamblingRelated()
}

// IsSystemTransaction returns true if this is a system-generated transaction
func (bh *BalanceHistory) IsSystemTransaction() bool {
	return bh.TransactionType.IsSystemGenerated()
}

// GetTransactionDescription returns a human-readable description of the transaction
func (bh *BalanceHistory) GetTransactionDescription() string {
	switch bh.TransactionType {
	case TransactionTypeBetWin:
		return "Bet win"
	case TransactionTypeBetLoss:
		return "Bet loss"
	case TransactionTypeWagerWin:
		return "Wager win"
	case TransactionTypeWagerLoss:
		return "Wager loss"
	case TransactionTypeGroupWagerWin:
		return "Group wager win"
	case TransactionTypeGroupWagerLoss:
		return "Group wager loss"
	case TransactionTypeTransferIn:
		return "Transfer received"
	case TransactionTypeTransferOut:
		return "Transfer sent"
	case TransactionTypeInitial:
		return "Initial balance"
	case TransactionTypeWordleReward:
		return "Wordle reward"
	default:
		return string(bh.TransactionType)
	}
}

// ValidateTransaction performs basic validation on the transaction
func (bh *BalanceHistory) ValidateTransaction() error {
	if bh.ChangeAmount == 0 {
		return errors.New("change amount cannot be zero")
	}
	
	if bh.BalanceAfter != bh.BalanceBefore+bh.ChangeAmount {
		return errors.New("balance calculation is inconsistent")
	}
	
	return nil
}