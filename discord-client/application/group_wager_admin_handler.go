package application

import (
	"context"
	"fmt"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/services"
)

type GroupWagerAdminHandler struct {
	uowFactory UnitOfWorkFactory
}

func NewGroupWagerAdminHandler(uowFactory UnitOfWorkFactory) *GroupWagerAdminHandler {
	return &GroupWagerAdminHandler{uowFactory: uowFactory}
}

func (h *GroupWagerAdminHandler) OverrideResult(ctx context.Context, req dto.OverrideGroupWagerResultRequest) (*dto.OverrideGroupWagerResultResponse, error) {
	if req.GuildID == 0 {
		return nil, fmt.Errorf("guild ID is required")
	}
	if req.AdminDiscordID == 0 {
		return nil, fmt.Errorf("admin Discord ID is required")
	}
	if req.GroupWagerID == 0 {
		return nil, fmt.Errorf("group wager ID is required")
	}
	if req.WinningOptionText == "" {
		return nil, fmt.Errorf("winning option is required")
	}

	uow := h.uowFactory.CreateForGuild(req.GuildID)
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	groupWagerService := services.NewGroupWagerService(
		uow.GroupWagerRepository(),
		uow.UserRepository(),
		uow.BalanceHistoryRepository(),
		uow.EventBus(),
	)

	wagerDetail, err := groupWagerService.GetGroupWagerDetail(ctx, req.GroupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager details: %w", err)
	}
	if wagerDetail == nil || wagerDetail.Wager == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	var winningOptionID int64
	found := false
	for _, option := range wagerDetail.Options {
		if option.OptionText == req.WinningOptionText {
			winningOptionID = option.ID
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("no option found with text: %s", req.WinningOptionText)
	}

	previousState := wagerDetail.Wager.State
	previousWinningOption := "No winning option"
	if wagerDetail.Wager.WinningOptionID != nil {
		for _, option := range wagerDetail.Options {
			if option.ID == *wagerDetail.Wager.WinningOptionID {
				previousWinningOption = option.OptionText
				break
			}
		}
	}

	result, err := groupWagerService.OverrideGroupWagerResult(ctx, req.GroupWagerID, &req.AdminDiscordID, winningOptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to override wager: %w", err)
	}

	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit override: %w", err)
	}

	winners := make([]dto.OverrideGroupWagerWinnerDTO, 0, len(result.Winners))
	for _, winner := range result.Winners {
		payout := int64(0)
		if winner.PayoutAmount != nil {
			payout = *winner.PayoutAmount
		}
		winners = append(winners, dto.OverrideGroupWagerWinnerDTO{
			DiscordID:    winner.DiscordID,
			PayoutAmount: payout,
		})
	}

	return &dto.OverrideGroupWagerResultResponse{
		GroupWagerID:          result.GroupWager.ID,
		PreviousState:         string(previousState),
		PreviousWinningOption: previousWinningOption,
		CorrectedState:        string(result.GroupWager.State),
		WinningOption:         result.WinningOption.OptionText,
		WinningOptionID:       result.WinningOption.ID,
		Winners:               winners,
		OriginalMessageID:     result.GroupWager.MessageID,
		OriginalChannelID:     result.GroupWager.ChannelID,
	}, nil
}
