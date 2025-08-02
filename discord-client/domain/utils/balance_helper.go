package utils

import (
	"context"
	"fmt"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/domain/interfaces"
	log "github.com/sirupsen/logrus"
)

const (
	// InitialBalance is the starting balance for new users (100k bits)
	InitialBalance int64 = 100000
)

// RecordBalanceChange records a balance history entry and emits appropriate events.
// This is the single entry point for all balance changes in the system.
func RecordBalanceChange(ctx context.Context, balanceHistoryRepo interfaces.BalanceHistoryRepository, eventPublisher interfaces.EventPublisher, history *entities.BalanceHistory) error {
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
	}).Debug("Publishing BalanceChangeEvent")
	if err := eventPublisher.Publish(event); err != nil {
		log.WithError(err).Error("Failed to publish balance change event")
	}

	// Also emit user created event if this is initial balance
	if history.TransactionType == entities.TransactionTypeInitial {
		if username, ok := history.TransactionMetadata["username"].(string); ok {
			userCreatedEvent := events.UserCreatedEvent{
				UserID:         history.DiscordID,
				DiscordID:      history.DiscordID,
				Username:       username,
				InitialBalance: history.BalanceAfter,
			}
			if err := eventPublisher.Publish(userCreatedEvent); err != nil {
				log.WithError(err).Error("Failed to publish user created event")
			}
		}
	}

	return nil
}
