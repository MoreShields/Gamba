package application

import (
	"context"
	"fmt"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/services"
)

// StagedBetHandler orchestrates staged bet operations using UoW and domain services
type StagedBetHandler struct {
	uowFactory UnitOfWorkFactory
}

// NewStagedBetHandler creates a new staged bet handler
func NewStagedBetHandler(uowFactory UnitOfWorkFactory) *StagedBetHandler {
	return &StagedBetHandler{
		uowFactory: uowFactory,
	}
}

// PlaceStagedBet validates and upserts a staged bet for a summoner's next game.
// Funds are locked immediately via the availableBalance SQL calculation.
func (h *StagedBetHandler) PlaceStagedBet(ctx context.Context, req dto.PlaceStagedBetRequest) (*dto.PlaceStagedBetResponse, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	uow := h.uowFactory.CreateForGuild(req.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Validate the summoner is watched in this guild and get the summoner ID
	watchDetail, err := uow.SummonerWatchRepository().GetWatch(ctx, req.GuildID, req.SummonerName, req.TagLine)
	if err != nil {
		return nil, fmt.Errorf("failed to look up summoner: %w", err)
	}
	if watchDetail == nil {
		return nil, fmt.Errorf("summoner %s#%s is not being watched in this guild", req.SummonerName, req.TagLine)
	}

	// Get or create the user
	userService := services.NewUserService(
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)
	user, err := userService.GetOrCreateUser(ctx, req.DiscordID, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check for existing staged bet to calculate the net additional amount needed
	existing, err := uow.StagedBetRepository().GetByUserAndSummoner(ctx, req.GuildID, req.DiscordID, watchDetail.SummonerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing staged bet: %w", err)
	}

	// Available balance already accounts for existing staged bets.
	// If updating, the existing amount is already subtracted, so we only need
	// enough additional balance for the difference.
	additionalNeeded := req.Amount
	if existing != nil {
		additionalNeeded = req.Amount - existing.Amount
	}

	if additionalNeeded > user.AvailableBalance {
		return nil, fmt.Errorf("insufficient balance. You need %d more bits (available: %d)", additionalNeeded, user.AvailableBalance)
	}

	// Upsert the staged bet
	stagedBet := &entities.StagedBet{
		GuildID:    req.GuildID,
		DiscordID:  req.DiscordID,
		SummonerID: watchDetail.SummonerID,
		Amount:     req.Amount,
	}
	if err := uow.StagedBetRepository().Upsert(ctx, stagedBet); err != nil {
		return nil, fmt.Errorf("failed to save staged bet: %w", err)
	}

	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Calculate remaining balance after this staged bet
	remainingBalance := user.AvailableBalance - additionalNeeded

	return &dto.PlaceStagedBetResponse{
		SummonerName:     watchDetail.SummonerName,
		TagLine:          watchDetail.TagLine,
		Amount:           req.Amount,
		RemainingBalance: remainingBalance,
		IsUpdate:         existing != nil,
	}, nil
}
