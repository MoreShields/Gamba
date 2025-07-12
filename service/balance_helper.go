package service

import (
	"context"
	"fmt"

	"gambler/events"
	"gambler/models"
	log "github.com/sirupsen/logrus"
)

// RecordBalanceChange records a balance history entry and emits appropriate events.
// This is the single entry point for all balance changes in the system.
func RecordBalanceChange(ctx context.Context, balanceHistoryRepo BalanceHistoryRepository, eventPublisher EventPublisher, history *models.BalanceHistory) error {
	// Record the balance history
	if err := balanceHistoryRepo.Record(ctx, history); err != nil {
		return fmt.Errorf("failed to record balance history: %w", err)
	}
	
	// Emit balance change event
	event := events.BalanceChangeEvent{
		UserID:          history.DiscordID,
		GuildID:         history.GuildID,
		OldBalance:      history.BalanceBefore,
		NewBalance:      history.BalanceAfter,
		TransactionType: history.TransactionType,
		ChangeAmount:    history.ChangeAmount,
	}
	log.WithFields(log.Fields{
		"userID":          event.UserID,
		"guildID":         event.GuildID,
		"oldBalance":      event.OldBalance,
		"newBalance":      event.NewBalance,
		"transactionType": event.TransactionType,
		"changeAmount":    event.ChangeAmount,
	}).Debug("Publishing BalanceChangeEvent to transactional bus")
	eventPublisher.Publish(event)
	
	// Also emit user created event if this is initial balance
	if history.TransactionType == models.TransactionTypeInitial {
		if username, ok := history.TransactionMetadata["username"].(string); ok {
			userCreatedEvent := events.UserCreatedEvent{
				UserID:         history.DiscordID,
				DiscordID:      history.DiscordID,
				Username:       username,
				InitialBalance: history.BalanceAfter,
			}
			eventPublisher.Publish(userCreatedEvent)
		}
	}
	
	return nil
}