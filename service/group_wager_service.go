package service

import (
	"context"
	"fmt"
	"gambler/config"
	"gambler/models"
	"time"
)

type groupWagerService struct {
	uowFactory UnitOfWorkFactory
	config     *config.Config
}

// NewGroupWagerService creates a new group wager service
func NewGroupWagerService(uowFactory UnitOfWorkFactory, cfg *config.Config) GroupWagerService {
	return &groupWagerService{
		uowFactory: uowFactory,
		config:     cfg,
	}
}

// CreateGroupWager creates a new group wager with options
func (s *groupWagerService) CreateGroupWager(ctx context.Context, creatorID int64, condition string, options []string, votingPeriodHours int, messageID, channelID int64) (*models.GroupWagerDetail, error) {
	// Validate inputs
	if condition == "" {
		return nil, fmt.Errorf("condition cannot be empty")
	}
	if len(options) < 2 {
		return nil, fmt.Errorf("must provide at least 2 options")
	}
	if votingPeriodHours < 1 || votingPeriodHours > 168 {
		return nil, fmt.Errorf("voting period must be between 1 and 168 hours")
	}

	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Check if creator exists
	creator, err := uow.UserRepository().GetByDiscordID(ctx, creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator: %w", err)
	}
	if creator == nil {
		return nil, fmt.Errorf("creator not found")
	}

	// Calculate voting period times
	now := time.Now()
	votingEndTime := now.Add(time.Duration(votingPeriodHours) * time.Hour)
	
	// Create the group wager
	groupWager := &models.GroupWager{
		CreatorDiscordID:  creatorID,
		Condition:         condition,
		State:             models.GroupWagerStateActive,
		TotalPot:          0,
		MinParticipants:   3,
		VotingPeriodHours: votingPeriodHours,
		VotingStartsAt:    &now,
		VotingEndsAt:      &votingEndTime,
		MessageID:         messageID,
		ChannelID:         channelID,
	}

	// Note: We'll create the wager with options in one atomic operation below

	// Create options
	var wagerOptions []*models.GroupWagerOption
	for i, optionText := range options {
		option := &models.GroupWagerOption{
			OptionText:   optionText,
			OptionOrder:  int16(i),
			TotalAmount:  0,
		}
		wagerOptions = append(wagerOptions, option)
	}

	// Use CreateWithOptions to create wager and options atomically
	if err := uow.GroupWagerRepository().CreateWithOptions(ctx, groupWager, wagerOptions); err != nil {
		return nil, fmt.Errorf("failed to create group wager with options: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.GroupWagerDetail{
		Wager:        groupWager,
		Options:      wagerOptions,
		Participants: []*models.GroupWagerParticipant{},
	}, nil
}

// PlaceBet allows a user to place or update their bet on a group wager option
func (s *groupWagerService) PlaceBet(ctx context.Context, groupWagerID int64, userID int64, optionID int64, amount int64) (*models.GroupWagerParticipant, error) {
	// Validate amount
	if amount <= 0 {
		return nil, fmt.Errorf("bet amount must be positive")
	}

	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get the group wager
	groupWager, err := uow.GroupWagerRepository().GetByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager: %w", err)
	}
	if groupWager == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	// Check if betting is allowed
	if !groupWager.CanAcceptBets() {
		if groupWager.IsActive() && groupWager.IsVotingPeriodExpired() {
			return nil, fmt.Errorf("voting period has ended, bets can no longer be placed or changed")
		}
		return nil, fmt.Errorf("group wager is not accepting bets (state: %s)", groupWager.State)
	}

	// Get full detail including options
	detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	options := detail.Options

	var selectedOption *models.GroupWagerOption
	for _, opt := range options {
		if opt.ID == optionID {
			selectedOption = opt
			break
		}
	}
	if selectedOption == nil {
		return nil, fmt.Errorf("invalid option ID")
	}

	// Check if user has sufficient balance
	user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check for existing participation
	existingParticipant, err := uow.GroupWagerRepository().GetParticipant(ctx, groupWagerID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing participation: %w", err)
	}

	var previousAmount int64 = 0
	var previousOptionID int64 = 0
	if existingParticipant != nil {
		previousAmount = existingParticipant.Amount
		previousOptionID = existingParticipant.OptionID
	}

	// Calculate the net change in balance needed
	netChange := amount - previousAmount
	if user.AvailableBalance < netChange {
		return nil, fmt.Errorf("insufficient balance: have %d available, need %d more", user.AvailableBalance, netChange)
	}

	// Create or update participant
	var participant *models.GroupWagerParticipant
	if existingParticipant != nil {
		// Update existing
		existingParticipant.OptionID = optionID
		existingParticipant.Amount = amount
		if err := uow.GroupWagerRepository().SaveParticipant(ctx, existingParticipant); err != nil {
			return nil, fmt.Errorf("failed to update participant: %w", err)
		}
		participant = existingParticipant
	} else {
		// Create new
		participant = &models.GroupWagerParticipant{
			GroupWagerID: groupWagerID,
			DiscordID:    userID,
			OptionID:     optionID,
			Amount:       amount,
		}
		if err := uow.GroupWagerRepository().SaveParticipant(ctx, participant); err != nil {
			return nil, fmt.Errorf("failed to create participant: %w", err)
		}
	}

	// Update option totals
	if previousOptionID != 0 && previousOptionID != optionID {
		// User changed options, update both
		for _, opt := range options {
			if opt.ID == previousOptionID {
				opt.TotalAmount -= previousAmount
				if err := uow.GroupWagerRepository().UpdateOptionTotal(ctx, opt.ID, opt.TotalAmount); err != nil {
					return nil, fmt.Errorf("failed to update previous option total: %w", err)
				}
			}
		}
		// When changing options, add the full amount to the new option
		selectedOption.TotalAmount += amount
	} else {
		// Same option, just update by the net change
		selectedOption.TotalAmount += netChange
	}
	if err := uow.GroupWagerRepository().UpdateOptionTotal(ctx, selectedOption.ID, selectedOption.TotalAmount); err != nil {
		return nil, fmt.Errorf("failed to update option total: %w", err)
	}

	// Update group wager total pot
	groupWager.TotalPot += netChange
	if err := uow.GroupWagerRepository().Update(ctx, groupWager); err != nil {
		return nil, fmt.Errorf("failed to update group wager pot: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return participant, nil
}

// ResolveGroupWager resolves a group wager with the winning option
func (s *groupWagerService) ResolveGroupWager(ctx context.Context, groupWagerID int64, resolverID int64, winningOptionID int64) (*models.GroupWagerResult, error) {
	// Check if user is a resolver
	if !s.IsResolver(resolverID) {
		return nil, fmt.Errorf("user is not authorized to resolve group wagers")
	}

	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get the group wager
	groupWager, err := uow.GroupWagerRepository().GetByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager: %w", err)
	}
	if groupWager == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	// Check if wager is active
	if !groupWager.IsActive() {
		return nil, fmt.Errorf("group wager is not active")
	}

	// Get full detail to get participants and options
	detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	participants := detail.Participants

	// Check minimum participants and multiple options
	participantsByOption := make(map[int64][]*models.GroupWagerParticipant)
	for _, p := range participants {
		participantsByOption[p.OptionID] = append(participantsByOption[p.OptionID], p)
	}

	if len(participants) < groupWager.MinParticipants {
		return nil, fmt.Errorf("insufficient participants: have %d, need %d", len(participants), groupWager.MinParticipants)
	}

	optionsWithParticipants := 0
	for _, parts := range participantsByOption {
		if len(parts) > 0 {
			optionsWithParticipants++
		}
	}
	if optionsWithParticipants < 2 {
		return nil, fmt.Errorf("need participants on at least 2 different options")
	}

	// Get the winning option from detail
	options := detail.Options

	var winningOption *models.GroupWagerOption
	for _, opt := range options {
		if opt.ID == winningOptionID {
			winningOption = opt
			break
		}
	}
	if winningOption == nil {
		return nil, fmt.Errorf("invalid winning option ID")
	}

	// Calculate payouts
	totalPot := groupWager.TotalPot
	winningOptionTotal := winningOption.TotalAmount

	if winningOptionTotal == 0 {
		return nil, fmt.Errorf("no participants on winning option")
	}

	var winners []*models.GroupWagerParticipant
	var losers []*models.GroupWagerParticipant
	payoutDetails := make(map[int64]int64)

	for _, participant := range participants {
		if participant.OptionID == winningOptionID {
			// Winner - calculate proportional payout
			payout := participant.CalculatePayout(winningOptionTotal, totalPot)
			participant.PayoutAmount = &payout
			winners = append(winners, participant)
			payoutDetails[participant.DiscordID] = payout
		} else {
			// Loser
			zero := int64(0)
			participant.PayoutAmount = &zero
			losers = append(losers, participant)
			payoutDetails[participant.DiscordID] = 0
		}
	}

	// Process payouts
	for _, winner := range winners {
		// Get user for balance history
		user, err := uow.UserRepository().GetByDiscordID(ctx, winner.DiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get winner user: %w", err)
		}

		// Calculate net win (payout - original bet)
		netWin := *winner.PayoutAmount - winner.Amount

		// Only update balance if there's a net change
		if netWin != 0 {
			if err := uow.UserRepository().AddBalance(ctx, winner.DiscordID, netWin); err != nil {
				return nil, fmt.Errorf("failed to update winner balance: %w", err)
			}

			// Record balance history
			history := &models.BalanceHistory{
				DiscordID:       winner.DiscordID,
				BalanceBefore:   user.Balance,
				BalanceAfter:    user.Balance + netWin,
				ChangeAmount:    netWin,
				TransactionType: models.TransactionTypeGroupWagerWin,
				TransactionMetadata: map[string]any{
					"group_wager_id": groupWagerID,
					"bet_amount":     winner.Amount,
					"payout_amount":  *winner.PayoutAmount,
					"condition":      groupWager.Condition,
				},
				RelatedID:   &groupWagerID,
				RelatedType: relatedTypePtr(models.RelatedTypeGroupWager),
			}

			if err := RecordBalanceChange(ctx, uow, history); err != nil {
				return nil, fmt.Errorf("failed to record winner balance change: %w", err)
			}

			// Update participant with balance history ID
			for i, w := range winners {
				if w.ID == winner.ID {
					winners[i].BalanceHistoryID = &history.ID
					break
				}
			}
		}
	}

	// Process losers
	for _, loser := range losers {
		// Get user for balance history
		user, err := uow.UserRepository().GetByDiscordID(ctx, loser.DiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get loser user: %w", err)
		}

		// Deduct the bet amount
		if err := uow.UserRepository().DeductBalance(ctx, loser.DiscordID, loser.Amount); err != nil {
			return nil, fmt.Errorf("failed to deduct loser balance: %w", err)
		}

		// Record balance history
		history := &models.BalanceHistory{
			DiscordID:       loser.DiscordID,
			BalanceBefore:   user.Balance,
			BalanceAfter:    user.Balance - loser.Amount,
			ChangeAmount:    -loser.Amount,
			TransactionType: models.TransactionTypeGroupWagerLoss,
			TransactionMetadata: map[string]any{
				"group_wager_id": groupWagerID,
				"bet_amount":     loser.Amount,
				"condition":      groupWager.Condition,
			},
			RelatedID:   &groupWagerID,
			RelatedType: relatedTypePtr(models.RelatedTypeGroupWager),
		}

		if err := RecordBalanceChange(ctx, uow, history); err != nil {
			return nil, fmt.Errorf("failed to record loser balance change: %w", err)
		}

		// Update participant with balance history ID
		for i, l := range losers {
			if l.ID == loser.ID {
				losers[i].BalanceHistoryID = &history.ID
				break
			}
		}
	}

	// Update participant records with payouts and balance history IDs
	allParticipants := append(winners, losers...)
	if err := uow.GroupWagerRepository().UpdateParticipantPayouts(ctx, allParticipants); err != nil {
		return nil, fmt.Errorf("failed to update participant payouts: %w", err)
	}

	// Update group wager as resolved
	now := time.Now()
	groupWager.State = models.GroupWagerStateResolved
	groupWager.ResolverDiscordID = &resolverID
	groupWager.WinningOptionID = &winningOptionID
	groupWager.ResolvedAt = &now

	if err := uow.GroupWagerRepository().Update(ctx, groupWager); err != nil {
		return nil, fmt.Errorf("failed to update resolved group wager: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.GroupWagerResult{
		GroupWager:    groupWager,
		WinningOption: winningOption,
		Winners:       winners,
		Losers:        losers,
		TotalPot:      totalPot,
		PayoutDetails: payoutDetails,
	}, nil
}

// GetGroupWagerDetail retrieves full details of a group wager
func (s *groupWagerService) GetGroupWagerDetail(ctx context.Context, groupWagerID int64) (*models.GroupWagerDetail, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	return detail, nil
}

// GetGroupWagerByMessageID retrieves a group wager by message ID
func (s *groupWagerService) GetGroupWagerByMessageID(ctx context.Context, messageID int64) (*models.GroupWagerDetail, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	detail, err := uow.GroupWagerRepository().GetDetailByMessageID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}

	return detail, nil
}

// GetActiveGroupWagersByUser returns active group wagers where user is participating
func (s *groupWagerService) GetActiveGroupWagersByUser(ctx context.Context, discordID int64) ([]*models.GroupWager, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	wagers, err := uow.GroupWagerRepository().GetActiveByUser(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active group wagers: %w", err)
	}

	return wagers, nil
}

// IsResolver checks if a user can resolve group wagers
func (s *groupWagerService) IsResolver(discordID int64) bool {
	for _, resolverID := range s.config.ResolverDiscordIDs {
		if discordID == resolverID {
			return true
		}
	}
	return false
}

// UpdateMessageIDs updates the message and channel IDs for a group wager
func (s *groupWagerService) UpdateMessageIDs(ctx context.Context, groupWagerID int64, messageID int64, channelID int64) error {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get the group wager
	groupWager, err := uow.GroupWagerRepository().GetByID(ctx, groupWagerID)
	if err != nil {
		return fmt.Errorf("failed to get group wager: %w", err)
	}

	// Update message and channel IDs
	groupWager.MessageID = messageID
	groupWager.ChannelID = channelID

	// Save the update
	if err := uow.GroupWagerRepository().Update(ctx, groupWager); err != nil {
		return fmt.Errorf("failed to update group wager: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

