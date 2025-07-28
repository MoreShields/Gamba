package application

import (
	"context"

	"gambler/discord-client/domain"
	"gambler/discord-client/events"
	"gambler/discord-client/service"
)


// RegisterApplicationSubscriptions registers all application-level event subscriptions
// This includes handlers for internal domain events that update Discord UI
func RegisterApplicationSubscriptions(
	subscriber domain.EventSubscriber,
	uowFactory service.UnitOfWorkFactory,
	discordPoster DiscordPoster,
) error {
	// Create the wager state event handler
	wagerStateHandler := NewWagerStateEventHandler(uowFactory, discordPoster)
	
	// Subscribe to group wager state change events
	return subscriber.Subscribe(events.EventTypeGroupWagerStateChange, 
		func(ctx context.Context, event events.Event) error {
			return wagerStateHandler.HandleGroupWagerStateChange(ctx, event)
		})
}