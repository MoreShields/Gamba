package bot

import (
	"context"
	"fmt"

	"gambler/discord-client/domain"
	"gambler/discord-client/events"

	log "github.com/sirupsen/logrus"
)

// RegisterBotSubscriptions registers all bot-level event subscriptions
// This includes handlers for events that affect Discord-specific features like roles
func RegisterBotSubscriptions(
	subscriber domain.EventSubscriber,
	bot *Bot,
) error {
	// Subscribe to balance change events for high roller role updates
	if err := subscriber.Subscribe(events.EventTypeBalanceChange,
		func(ctx context.Context, event events.Event) error {
			return handleBalanceChangeForHighRoller(ctx, event, bot)
		}); err != nil {
		return fmt.Errorf("failed to subscribe to balance change events: %w", err)
	}

	log.Info("Bot event subscriptions registered successfully")
	return nil
}

// handleBalanceChangeForHighRoller processes balance change events to update high roller roles
func handleBalanceChangeForHighRoller(ctx context.Context, event events.Event, bot *Bot) error {
	balanceEvent, ok := event.(*events.BalanceChangeEvent)
	if !ok {
		return fmt.Errorf("received non-BalanceChangeEvent in balance change handler")
	}

	log.WithFields(log.Fields{
		"userID":          balanceEvent.UserID,
		"guildID":         balanceEvent.GuildID,
		"oldBalance":      balanceEvent.OldBalance,
		"newBalance":      balanceEvent.NewBalance,
		"transactionType": balanceEvent.TransactionType,
		"changeAmount":    balanceEvent.ChangeAmount,
	}).Info("Processing balance change event for high roller role update")

	if err := bot.UpdateHighRollerRole(ctx, balanceEvent.GuildID); err != nil {
		log.WithFields(log.Fields{
			"guildID": balanceEvent.GuildID,
			"error":   err,
		}).Error("Failed to update high roller role")
		return err
	}

	log.Debug("High roller role update completed successfully")
	return nil
}
