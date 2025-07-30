package application

import (
	"context"

	"gambler/discord-client/domain"
	"gambler/discord-client/events"

	log "github.com/sirupsen/logrus"
)

// LocalHandlerRegistry allows registering handlers for local event processing
type LocalHandlerRegistry interface {
	RegisterLocalHandler(eventType events.EventType, handler func(context.Context, events.Event) error)
}

// RegisterApplicationSubscriptions registers all application-level event subscriptions
// This includes handlers for internal domain events that update Discord UI
func RegisterApplicationSubscriptions(
	subscriber domain.EventSubscriber,
	uowFactory UnitOfWorkFactory,
	discordPoster DiscordPoster,
) error {
	// Create the wager state event handler
	wagerStateHandler := NewWagerStateEventHandler(uowFactory, discordPoster)

	// Create the Wordle handler
	wordleHandler := NewWordleHandler(uowFactory)

	// Register as local handler to handle events published within the same process
	// Since NATS doesn't deliver messages back to the publisher, we need local handling
	if localRegistry, ok := uowFactory.(LocalHandlerRegistry); ok {
		localRegistry.RegisterLocalHandler(events.EventTypeGroupWagerStateChange,
			func(ctx context.Context, event events.Event) error {
				return wagerStateHandler.HandleGroupWagerStateChange(ctx, event)
			})
		log.Info("Registered local handler for GroupWagerStateChange events")

		// Register Discord message handler for Wordle bot processing
		localRegistry.RegisterLocalHandler(events.EventTypeDiscordMessage,
			func(ctx context.Context, event events.Event) error {
				return wordleHandler.HandleDiscordMessage(ctx, event)
			})
		log.Info("Registered local handler for Discord message events")
	} else {
		log.Warn("UnitOfWorkFactory does not support local handler registration")
	}

	return nil
}
