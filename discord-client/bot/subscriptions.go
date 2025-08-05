package bot

import (
	"gambler/discord-client/domain"

	log "github.com/sirupsen/logrus"
)

// RegisterBotSubscriptions registers all bot-level event subscriptions
// This includes handlers for events that affect Discord-specific features
func RegisterBotSubscriptions(
	subscriber domain.EventSubscriber,
	bot *Bot,
) error {
	// Currently no bot-level subscriptions needed
	// High roller updates are now manual through purchase commands
	
	log.Info("Bot event subscriptions registered successfully")
	return nil
}
