package entities

// TransactionType represents the type of balance change
type TransactionType string

// All transaction types supported by the system
const (
	// Gambling-related transactions
	TransactionTypeBetWin         TransactionType = "bet_win"
	TransactionTypeBetLoss        TransactionType = "bet_loss"
	TransactionTypeWagerWin       TransactionType = "wager_win"
	TransactionTypeWagerLoss      TransactionType = "wager_loss"
	TransactionTypeGroupWagerWin  TransactionType = "group_wager_win"
	TransactionTypeGroupWagerLoss TransactionType = "group_wager_loss"

	// Transfer transactions
	TransactionTypeTransferIn  TransactionType = "transfer_in"
	TransactionTypeTransferOut TransactionType = "transfer_out"

	// Lottery transactions
	TransactionTypeLottoTicket TransactionType = "lotto_ticket"
	TransactionTypeLottoWin    TransactionType = "lotto_win"

	// System transactions
	TransactionTypeInitial            TransactionType = "initial"
	TransactionTypeWordleReward       TransactionType = "wordle_reward"
	TransactionTypeHighRollerPurchase TransactionType = "high_roller_purchase"
)

// IsWinType returns true if the transaction type represents a win
func (tt TransactionType) IsWinType() bool {
	return tt == TransactionTypeBetWin ||
		tt == TransactionTypeWagerWin ||
		tt == TransactionTypeGroupWagerWin
}

// IsLossType returns true if the transaction type represents a loss
func (tt TransactionType) IsLossType() bool {
	return tt == TransactionTypeBetLoss ||
		tt == TransactionTypeWagerLoss ||
		tt == TransactionTypeGroupWagerLoss
}

// IsTransferType returns true if the transaction type represents a transfer
func (tt TransactionType) IsTransferType() bool {
	return tt == TransactionTypeTransferIn ||
		tt == TransactionTypeTransferOut
}

// IsGamblingRelated returns true if the transaction type is gambling-related
func (tt TransactionType) IsGamblingRelated() bool {
	return tt.IsWinType() || tt.IsLossType()
}

// IsSystemGenerated returns true if the transaction type is system-generated
func (tt TransactionType) IsSystemGenerated() bool {
	return tt == TransactionTypeInitial ||
		tt == TransactionTypeWordleReward ||
		tt == TransactionTypeHighRollerPurchase
}

// String returns the string representation of the transaction type
func (tt TransactionType) String() string {
	return string(tt)
}