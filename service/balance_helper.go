package service

import (
	"context"
	"fmt"

	"gambler/events"
	"gambler/models"
)

// RecordBalanceChange records a balance history entry and emits appropriate events.
// This is the single entry point for all balance changes in the system.
func RecordBalanceChange(ctx context.Context, uow UnitOfWork, history *models.BalanceHistory) error {
	// Record the balance history
	if err := uow.BalanceHistoryRepository().Record(ctx, history); err != nil {
		return fmt.Errorf("failed to record balance history: %w", err)
	}
	
	// Emit balance change event (will be flushed after transaction commits)
	event := events.BalanceChangeEvent{
		UserID:          history.DiscordID,
		OldBalance:      history.BalanceBefore,
		NewBalance:      history.BalanceAfter,
		TransactionType: history.TransactionType,
		ChangeAmount:    history.ChangeAmount,
	}
	uow.EventBus().Publish(event)
	
	// Also emit user created event if this is initial balance
	if history.TransactionType == models.TransactionTypeInitial {
		if username, ok := history.TransactionMetadata["username"].(string); ok {
			userCreatedEvent := events.UserCreatedEvent{
				UserID:         history.DiscordID,
				DiscordID:      history.DiscordID,
				Username:       username,
				InitialBalance: history.BalanceAfter,
			}
			uow.EventBus().Publish(userCreatedEvent)
		}
	}
	
	return nil
}