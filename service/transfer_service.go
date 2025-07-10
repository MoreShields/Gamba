package service

import (
	"context"
	"fmt"

	"gambler/models"
)

type transferService struct {
	uowFactory UnitOfWorkFactory
}

// NewTransferService creates a new transfer service
func NewTransferService(uowFactory UnitOfWorkFactory) TransferService {
	return &transferService{
		uowFactory: uowFactory,
	}
}

func (s *transferService) Transfer(ctx context.Context, fromDiscordID int64, toDiscordID int64, amount int64) (*models.TransferResult, error) {
	// Validate inputs
	if amount <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}
	if fromDiscordID == toDiscordID {
		return nil, fmt.Errorf("cannot transfer to yourself")
	}

	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	// Get sender user
	fromUser, err := uow.UserRepository().GetByDiscordID(ctx, fromDiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender user: %w", err)
	}
	if fromUser == nil {
		return nil, fmt.Errorf("sender user not found")
	}

	// Check if sender has sufficient balance
	if fromUser.Balance < amount {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d", fromUser.Balance, amount)
	}

	// Get recipient user
	toUser, err := uow.UserRepository().GetByDiscordID(ctx, toDiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipient user: %w", err)
	}
	if toUser == nil {
		return nil, fmt.Errorf("recipient user not found")
	}

	// Calculate new balances
	newFromBalance := fromUser.Balance - amount
	newToBalance := toUser.Balance + amount

	// Deduct amount from sender
	if err := uow.UserRepository().DeductBalance(ctx, fromDiscordID, amount); err != nil {
		return nil, fmt.Errorf("failed to deduct transfer amount: %w", err)
	}

	// Add amount to recipient
	if err := uow.UserRepository().AddBalance(ctx, toDiscordID, amount); err != nil {
		return nil, fmt.Errorf("failed to add transfer amount: %w", err)
	}

	// Create balance history record for sender (outgoing transfer)
	fromHistory := &models.BalanceHistory{
		DiscordID:       fromDiscordID,
		BalanceBefore:   fromUser.Balance,
		BalanceAfter:    newFromBalance,
		ChangeAmount:    -amount,
		TransactionType: models.TransactionTypeTransferOut,
		TransactionMetadata: map[string]any{
			"recipient_discord_id": toDiscordID,
			"recipient_username":   toUser.Username,
			"transfer_amount":      amount,
		},
	}

	if err := RecordBalanceChange(ctx, uow, fromHistory); err != nil {
		return nil, fmt.Errorf("failed to record sender balance change: %w", err)
	}

	// Create balance history record for recipient (incoming transfer)
	toHistory := &models.BalanceHistory{
		DiscordID:       toDiscordID,
		BalanceBefore:   toUser.Balance,
		BalanceAfter:    newToBalance,
		ChangeAmount:    amount,
		TransactionType: models.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"sender_discord_id": fromDiscordID,
			"sender_username":   fromUser.Username,
			"transfer_amount":   amount,
		},
	}

	if err := RecordBalanceChange(ctx, uow, toHistory); err != nil {
		return nil, fmt.Errorf("failed to record recipient balance change: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.TransferResult{
		Amount:        amount,
		RecipientName: toUser.Username,
		NewBalance:    newFromBalance,
	}, nil
}