package models

import (
	"time"
)

// TransactionType represents the type of balance change
type TransactionType string

const (
	TransactionTypeBetWin         TransactionType = "bet_win"
	TransactionTypeBetLoss        TransactionType = "bet_loss"
	TransactionTypeTransferIn     TransactionType = "transfer_in"
	TransactionTypeTransferOut    TransactionType = "transfer_out"
	TransactionTypeInitial        TransactionType = "initial"
	TransactionTypeWagerWin       TransactionType = "wager_win"
	TransactionTypeWagerLoss      TransactionType = "wager_loss"
	TransactionTypeGroupWagerWin  TransactionType = "group_wager_win"
	TransactionTypeGroupWagerLoss TransactionType = "group_wager_loss"
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
	ID                  int64           `db:"id"`
	DiscordID           int64           `db:"discord_id"`
	BalanceBefore       int64           `db:"balance_before"`
	BalanceAfter        int64           `db:"balance_after"`
	ChangeAmount        int64           `db:"change_amount"`
	TransactionType     TransactionType `db:"transaction_type"`
	TransactionMetadata map[string]any  `db:"transaction_metadata"`
	RelatedID           *int64          `db:"related_id"`
	RelatedType         *RelatedType    `db:"related_type"`
	CreatedAt           time.Time       `db:"created_at"`
}
