package application

import (
	"context"
	"fmt"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/events"
	"gambler/discord-client/service"

	log "github.com/sirupsen/logrus"
)

// wagerStateEventHandler implements the WagerStateEventHandler interface
type wagerStateEventHandler struct {
	uowFactory    service.UnitOfWorkFactory
	discordPoster DiscordPoster
}

// NewWagerStateEventHandler creates a new WagerStateEventHandler
func NewWagerStateEventHandler(uowFactory service.UnitOfWorkFactory, discordPoster DiscordPoster) WagerStateEventHandler {
	return &wagerStateEventHandler{
		uowFactory:    uowFactory,
		discordPoster: discordPoster,
	}
}

// HandleGroupWagerStateChange handles GroupWagerStateChangeEvent and updates Discord messages
func (h *wagerStateEventHandler) HandleGroupWagerStateChange(ctx context.Context, event interface{}) error {
	e, err := AssertEventType[events.GroupWagerStateChangeEvent](event, "GroupWagerStateChangeEvent")
	if err != nil {
		return err
	}

	log.Infof("WagerStateEventHandler: handling state change for wager %d (state: %s -> %s)",
		e.GroupWagerID, e.OldState, e.NewState)

	// Skip if no message to update
	if e.MessageID == 0 || e.ChannelID == 0 {
		log.Errorf("Failed to handle group wager state change, missing messageID or channelID.")
		return fmt.Errorf("missing messageID or channelID")
	}

	// Create guild-scoped unit of work using GuildID from event
	uow := h.uowFactory.CreateForGuild(e.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Instantiate service with repositories from UnitOfWork
	groupWagerService := service.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	// Fetch the latest wager detail
	detail, err := groupWagerService.GetGroupWagerDetail(ctx, e.GroupWagerID)
	if err != nil {
		return fmt.Errorf("failed to get wager detail for ID %d: %w", e.GroupWagerID, err)
	}

	if detail == nil {
		return fmt.Errorf("wager with ID %d not found", e.GroupWagerID)
	}

	// We don't need to commit this transaction since we're only reading
	// But we need to properly close it
	if err := uow.Commit(); err != nil {
		log.Warnf("Failed to commit read-only transaction for wager %d: %v", e.GroupWagerID, err)
	}

	// Determine wager type and update accordingly
	if detail.Wager.IsHouseWager() {

		// Convert to HouseWagerPostDTO using our converter
		houseWagerDTO := dto.GroupWagerDetailToHouseWagerPostDTO(detail)

		// Update the Discord message
		return h.discordPoster.UpdateHouseWager(ctx, detail.Wager.MessageID, detail.Wager.ChannelID, houseWagerDTO)
	} else {
		log.Infof("Updating group wager message: wagerID=%d, state=%s", detail.Wager.ID, detail.Wager.State)

		// For regular group wagers, pass the detail directly
		return h.discordPoster.UpdateGroupWager(ctx, detail.Wager.MessageID, detail.Wager.ChannelID, detail)
	}
}
