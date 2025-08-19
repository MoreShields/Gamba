package services

import (
	"context"
	"fmt"
	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/utils"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type groupWagerService struct {
	config             *config.Config
	groupWagerRepo     interfaces.GroupWagerRepository
	userRepo           interfaces.UserRepository
	balanceHistoryRepo interfaces.BalanceHistoryRepository
	eventPublisher     interfaces.EventPublisher
}

// NewGroupWagerService creates a new group wager service
func NewGroupWagerService(
	groupWagerRepo interfaces.GroupWagerRepository,
	userRepo interfaces.UserRepository,
	balanceHistoryRepo interfaces.BalanceHistoryRepository,
	eventPublisher interfaces.EventPublisher,
) interfaces.GroupWagerService {
	return &groupWagerService{
		config:             config.Get(),
		groupWagerRepo:     groupWagerRepo,
		userRepo:           userRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		eventPublisher:     eventPublisher,
	}
}

// CreateGroupWager creates a new group wager with options
func (s *groupWagerService) CreateGroupWager(ctx context.Context, creatorID *int64, condition string, options []string, votingPeriodMinutes int, messageID, channelID int64, wagerType entities.GroupWagerType, oddsMultipliers []float64) (*entities.GroupWagerDetail, error) {
	// Validate inputs
	if condition == "" {
		return nil, fmt.Errorf("condition cannot be empty")
	}
	if len(options) < 2 {
		return nil, fmt.Errorf("must provide at least 2 options")
	}
	if votingPeriodMinutes < 5 || votingPeriodMinutes > 10080 {
		return nil, fmt.Errorf("voting period must be between 5 minutes and 168 hours (10080 minutes)")
	}

	// Validate wager type
	if wagerType != entities.GroupWagerTypePool && wagerType != entities.GroupWagerTypeHouse {
		return nil, fmt.Errorf("invalid wager type: %s", wagerType)
	}

	// Validate odds multipliers for house wagers
	if wagerType == entities.GroupWagerTypeHouse {
		if len(oddsMultipliers) != len(options) {
			return nil, fmt.Errorf("must provide odds multiplier for each option")
		}
		for i, multiplier := range oddsMultipliers {
			if multiplier <= 0 {
				return nil, fmt.Errorf("odds multiplier for option %d must be positive", i+1)
			}
		}
	} else if wagerType == entities.GroupWagerTypePool {
		// Pool wagers should not have odds multipliers provided
		if len(oddsMultipliers) > 0 {
			return nil, fmt.Errorf("pool wagers calculate their own odds, do not provide odds multipliers")
		}
	}

	// Check for duplicate options (case-insensitive)
	optionMap := make(map[string]bool)
	for _, option := range options {
		lowerOption := strings.ToLower(strings.TrimSpace(option))
		if optionMap[lowerOption] {
			return nil, fmt.Errorf("duplicate option found: '%s'. Each option must be unique", option)
		}
		optionMap[lowerOption] = true
	}

	// Check if creator exists (skip validation for system-created wagers)
	if creatorID != nil {
		creator, err := s.userRepo.GetByDiscordID(ctx, *creatorID)
		if err != nil {
			return nil, fmt.Errorf("failed to get creator: %w", err)
		}
		if creator == nil {
			return nil, fmt.Errorf("creator %d not found", *creatorID)
		}
	}

	// Calculate voting period times
	now := time.Now()
	votingEndTime := now.Add(time.Duration(votingPeriodMinutes) * time.Minute)

	// Set minimum participants to 0 for all wager types
	// Allow wagers to resolve with any number of participants
	minParticipants := 0

	// Create the group wager
	groupWager := &entities.GroupWager{
		CreatorDiscordID:    creatorID,
		Condition:           condition,
		State:               entities.GroupWagerStateActive,
		WagerType:           wagerType,
		TotalPot:            0,
		MinParticipants:     minParticipants,
		VotingPeriodMinutes: votingPeriodMinutes,
		VotingStartsAt:      &now,
		VotingEndsAt:        &votingEndTime,
		MessageID:           messageID,
		ChannelID:           channelID,
	}

	// Note: We'll create the wager with options in one atomic operation below

	// Create options
	var wagerOptions []*entities.GroupWagerOption
	for i, optionText := range options {
		var odds float64
		if wagerType == entities.GroupWagerTypeHouse {
			odds = oddsMultipliers[i]
		} else {
			// For pool wagers, start with 0 odds (will be calculated after creation)
			odds = 0
		}

		option := &entities.GroupWagerOption{
			OptionText:     optionText,
			OptionOrder:    int16(i),
			TotalAmount:    0,
			OddsMultiplier: odds,
		}
		wagerOptions = append(wagerOptions, option)
	}

	// Use CreateWithOptions to create wager and options atomically
	if err := s.groupWagerRepo.CreateWithOptions(ctx, groupWager, wagerOptions); err != nil {
		return nil, fmt.Errorf("failed to create group wager with options: %w", err)
	}

	return &entities.GroupWagerDetail{
		Wager:        groupWager,
		Options:      wagerOptions,
		Participants: []*entities.GroupWagerParticipant{},
	}, nil
}

// PlaceBet allows a user to place or update their bet on a group wager option
func (s *groupWagerService) PlaceBet(ctx context.Context, groupWagerID int64, userID int64, optionID int64, amount int64) (*entities.GroupWagerParticipant, error) {
	// Validate amount
	if amount <= 0 {
		return nil, fmt.Errorf("bet amount must be positive")
	}

	// Get full detail including options and wager
	detail, err := s.groupWagerRepo.GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil || detail.Wager == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	groupWager := detail.Wager

	// Check if betting is allowed
	if !groupWager.CanAcceptBets() {
		if groupWager.IsActive() && groupWager.IsVotingPeriodExpired() {
			return nil, fmt.Errorf("voting period has ended, bets can no longer be placed or changed")
		}
		// Provide user-friendly error messages for specific states
		switch groupWager.State {
		case entities.GroupWagerStateResolved:
			return nil, fmt.Errorf("cannot place bets on resolved wager")
		case entities.GroupWagerStateCancelled:
			return nil, fmt.Errorf("cannot place bets on cancelled wager")
		default:
			return nil, fmt.Errorf("group wager is not accepting bets (state: %s)", groupWager.State)
		}
	}

	options := detail.Options

	var selectedOption *entities.GroupWagerOption
	for _, opt := range options {
		if opt.ID == optionID {
			selectedOption = opt
			break
		}
	}
	if selectedOption == nil {
		return nil, fmt.Errorf("invalid option ID")
	}

	// Check max wager limit for LoL and TFT games
	if groupWager.ExternalRef != nil &&
		(groupWager.ExternalRef.System == entities.SystemLeagueOfLegends ||
			groupWager.ExternalRef.System == entities.SystemTFT) {
		if s.config.MaxLolWagerPerGame > 0 && amount > s.config.MaxLolWagerPerGame {
			return nil, fmt.Errorf("bet amount exceeds maximum of %s bits per game", utils.FormatShortNotation(s.config.MaxLolWagerPerGame))
		}
	}

	// Check if user has sufficient balance
	user, err := s.userRepo.GetByDiscordID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user %d not found", userID)
	}

	// Check for existing participation
	existingParticipant, err := s.groupWagerRepo.GetParticipant(ctx, groupWagerID, userID)
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
		return nil, fmt.Errorf("insufficient balance: have %s available, need %s more", utils.FormatShortNotation(user.AvailableBalance), utils.FormatShortNotation(netChange))
	}

	// Create or update participant
	var participant *entities.GroupWagerParticipant
	if existingParticipant != nil {
		// Update existing
		existingParticipant.OptionID = optionID
		existingParticipant.Amount = amount
		if err := s.groupWagerRepo.SaveParticipant(ctx, existingParticipant); err != nil {
			return nil, fmt.Errorf("failed to update participant: %w", err)
		}
		participant = existingParticipant
	} else {
		// Create new
		participant = &entities.GroupWagerParticipant{
			GroupWagerID: groupWagerID,
			DiscordID:    userID,
			OptionID:     optionID,
			Amount:       amount,
		}
		if err := s.groupWagerRepo.SaveParticipant(ctx, participant); err != nil {
			return nil, fmt.Errorf("failed to create participant: %w", err)
		}
	}

	// Update option totals
	if previousOptionID != 0 && previousOptionID != optionID {
		// User changed options, update both
		for _, opt := range options {
			if opt.ID == previousOptionID {
				opt.TotalAmount -= previousAmount
				if err := s.groupWagerRepo.UpdateOptionTotal(ctx, opt.ID, opt.TotalAmount); err != nil {
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
	if err := s.groupWagerRepo.UpdateOptionTotal(ctx, selectedOption.ID, selectedOption.TotalAmount); err != nil {
		return nil, fmt.Errorf("failed to update option total: %w", err)
	}

	// Update group wager total pot
	groupWager.TotalPot += netChange
	if err := s.groupWagerRepo.Update(ctx, groupWager); err != nil {
		return nil, fmt.Errorf("failed to update group wager pot: %w", err)
	}

	// For pool wagers, recalculate and update odds for all options
	if groupWager.IsPoolWager() && groupWager.TotalPot > 0 {
		oddsUpdates := make(map[int64]float64)
		for _, opt := range options {
			multiplier := opt.CalculateMultiplier(groupWager.TotalPot)
			oddsUpdates[opt.ID] = multiplier
		}
		if err := s.groupWagerRepo.UpdateAllOptionOdds(ctx, groupWagerID, oddsUpdates); err != nil {
			return nil, fmt.Errorf("failed to update option odds: %w", err)
		}
	}

	return participant, nil
}

// calculateMaxWinnerBet finds the highest bet amount among winners
func calculateMaxWinnerBet(winners []*entities.GroupWagerParticipant) int64 {
	maxBet := int64(0)
	for _, winner := range winners {
		if winner.Amount > maxBet {
			maxBet = winner.Amount
		}
	}
	return maxBet
}

// calculateEffectiveLoss calculates the actual loss amount considering exposure cap
func calculateEffectiveLoss(betAmount, maxWinnerBet int64) int64 {
	if maxWinnerBet > 0 && betAmount > maxWinnerBet {
		return maxWinnerBet
	}
	return betAmount
}

// processParticipantBalanceChange updates participant balance and records history
func (s *groupWagerService) processParticipantBalanceChange(
	ctx context.Context,
	participant *entities.GroupWagerParticipant,
	balanceChange int64,
	transactionType entities.TransactionType,
	groupWagerID int64,
	groupWager *entities.GroupWager,
	maxWinnerBet int64,
) (*entities.BalanceHistory, error) {
	// Get user for balance history
	user, err := s.userRepo.GetByDiscordID(ctx, participant.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update balance
	newBalance := user.Balance + balanceChange
	if err := s.userRepo.UpdateBalance(ctx, participant.DiscordID, newBalance); err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Build transaction metadata
	metadata := map[string]any{
		"group_wager_id": groupWagerID,
		"bet_amount":     participant.Amount,
		"condition":      groupWager.Condition,
		"wager_type":     string(groupWager.WagerType),
	}

	// Add payout info for winners
	if transactionType == entities.TransactionTypeGroupWagerWin && participant.PayoutAmount != nil {
		metadata["payout_amount"] = *participant.PayoutAmount
	}

	// Add capped loss info for losers if applicable
	if transactionType == entities.TransactionTypeGroupWagerLoss &&
		groupWager.IsPoolWager() && maxWinnerBet > 0 && participant.Amount > maxWinnerBet {
		metadata["capped_loss"] = maxWinnerBet
		metadata["original_bet"] = participant.Amount
	}

	// Record balance history
	history := &entities.BalanceHistory{
		DiscordID:           participant.DiscordID,
		BalanceBefore:       user.Balance,
		BalanceAfter:        newBalance,
		ChangeAmount:        balanceChange,
		TransactionType:     transactionType,
		TransactionMetadata: metadata,
		RelatedID:           &groupWagerID,
		RelatedType:         relatedTypePtr(entities.RelatedTypeGroupWager),
	}

	if err := utils.RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, history); err != nil {
		return nil, fmt.Errorf("failed to record balance change: %w", err)
	}

	return history, nil
}

// ResolveGroupWager resolves a group wager with the winning option
func (s *groupWagerService) ResolveGroupWager(ctx context.Context, groupWagerID int64, resolverID *int64, winningOptionID int64) (*entities.GroupWagerResult, error) {
	// Check if user is a resolver (skip check for system resolution when resolverID is nil)
	if resolverID != nil && !s.IsResolver(*resolverID) {
		return nil, fmt.Errorf("user is not authorized to resolve group wagers")
	}

	// Get full detail to get participants and options (this includes the wager with external reference)
	detail, err := s.groupWagerRepo.GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil || detail.Wager == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	groupWager := detail.Wager

	// Check if wager can be resolved
	if !groupWager.IsActive() && !groupWager.IsPendingResolution() {
		return nil, fmt.Errorf("group wager cannot be resolved (current state: %s)", groupWager.State)
	}

	participants := detail.Participants

	// Check minimum participants and multiple options
	participantsByOption := make(map[int64][]*entities.GroupWagerParticipant)
	for _, p := range participants {
		participantsByOption[p.OptionID] = append(participantsByOption[p.OptionID], p)
	}

	// No restrictions on participant distribution - allow resolution in any case

	// Get the winning option from detail by ID
	options := detail.Options

	var winningOption *entities.GroupWagerOption
	for _, opt := range options {
		if opt.ID == winningOptionID {
			winningOption = opt
			break
		}
	}
	if winningOption == nil {
		return nil, fmt.Errorf("no option found with ID: %d", winningOptionID)
	}

	// Calculate payouts
	totalPot := groupWager.TotalPot
	winningOptionTotal := winningOption.TotalAmount

	var winners []*entities.GroupWagerParticipant
	var losers []*entities.GroupWagerParticipant
	payoutDetails := make(map[int64]int64)

	// Separate winners and losers
	for _, participant := range participants {
		if participant.OptionID == winningOption.ID {
			winners = append(winners, participant)
		} else {
			losers = append(losers, participant)
		}
	}

	// Calculate max winner bet once for pool wagers
	var maxWinnerBet int64
	if groupWager.IsPoolWager() {
		maxWinnerBet = calculateMaxWinnerBet(winners)
	}

	// Calculate payouts based on wager type
	if groupWager.IsPoolWager() {

		// Calculate effective prize pool with capped losses
		effectivePrizePool := int64(0)

		// Add capped losses from losers
		for _, loser := range losers {
			effectiveLoss := calculateEffectiveLoss(loser.Amount, maxWinnerBet)
			effectivePrizePool += effectiveLoss
		}

		// Add winner contributions to prize pool
		for _, winner := range winners {
			effectivePrizePool += winner.Amount
		}

		// Calculate proportional payouts for winners
		for _, winner := range winners {
			var payout int64
			if winningOptionTotal > 0 {
				payout = (winner.Amount * effectivePrizePool) / winningOptionTotal
			}
			winner.PayoutAmount = &payout
			payoutDetails[winner.DiscordID] = payout
		}

		// Set loser payouts to 0
		for _, loser := range losers {
			zero := int64(0)
			loser.PayoutAmount = &zero
			payoutDetails[loser.DiscordID] = 0
		}
	} else {
		// House wager: use existing logic with odds multipliers
		for _, winner := range winners {
			payout := int64(float64(winner.Amount) * winningOption.OddsMultiplier)
			winner.PayoutAmount = &payout
			payoutDetails[winner.DiscordID] = payout
		}

		for _, loser := range losers {
			zero := int64(0)
			loser.PayoutAmount = &zero
			payoutDetails[loser.DiscordID] = 0
		}
	}

	// Process payouts
	for i, winner := range winners {
		// Calculate balance change: net win (payout - original bet)
		balanceChange := *winner.PayoutAmount - winner.Amount

		// Process balance update and history
		history, err := s.processParticipantBalanceChange(
			ctx, winner, balanceChange, entities.TransactionTypeGroupWagerWin,
			groupWagerID, groupWager, maxWinnerBet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to process winner balance: %w", err)
		}

		// Update participant with balance history ID
		winners[i].BalanceHistoryID = &history.ID
	}

	// Process losers
	for i, loser := range losers {
		// Calculate balance change
		var balanceChange int64
		if groupWager.IsPoolWager() {
			// Pool wager with exposure cap
			balanceChange = -calculateEffectiveLoss(loser.Amount, maxWinnerBet)
		} else {
			// House wager: lose full bet amount
			balanceChange = -loser.Amount
		}

		// Process balance update and history
		history, err := s.processParticipantBalanceChange(
			ctx, loser, balanceChange, entities.TransactionTypeGroupWagerLoss,
			groupWagerID, groupWager, maxWinnerBet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to process loser balance: %w", err)
		}

		// Update participant with balance history ID
		losers[i].BalanceHistoryID = &history.ID
	}

	// Update participant records with payouts and balance history IDs
	allParticipants := append(winners, losers...)
	if err := s.groupWagerRepo.UpdateParticipantPayouts(ctx, allParticipants); err != nil {
		return nil, fmt.Errorf("failed to update participant payouts: %w", err)
	}

	// Update group wager as resolved
	now := time.Now()
	oldState := groupWager.State
	groupWager.State = entities.GroupWagerStateResolved
	groupWager.ResolverDiscordID = resolverID
	groupWager.WinningOptionID = &winningOptionID
	groupWager.ResolvedAt = &now

	if err := s.groupWagerRepo.Update(ctx, groupWager); err != nil {
		return nil, fmt.Errorf("failed to update resolved group wager: %w", err)
	}

	// Publish state change event
	if err := s.eventPublisher.Publish(events.GroupWagerStateChangeEvent{
		GroupWagerID: groupWager.ID,
		GuildID:      groupWager.GuildID,
		OldState:     string(oldState),
		NewState:     string(groupWager.State),
		MessageID:    groupWager.MessageID,
		ChannelID:    groupWager.ChannelID,
	}); err != nil {
		log.WithError(err).Error("Failed to publish group wager state change event")
	}

	return &entities.GroupWagerResult{
		GroupWager:    groupWager,
		WinningOption: winningOption,
		Winners:       winners,
		Losers:        losers,
		TotalPot:      totalPot,
		PayoutDetails: payoutDetails,
	}, nil
}

// GetGroupWagerDetail retrieves full details of a group wager
func (s *groupWagerService) GetGroupWagerDetail(ctx context.Context, groupWagerID int64) (*entities.GroupWagerDetail, error) {
	detail, err := s.groupWagerRepo.GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil {
		return nil, fmt.Errorf("group wager not found")
	}

	return detail, nil
}

// GetGroupWagerByMessageID retrieves a group wager by message ID
func (s *groupWagerService) GetGroupWagerByMessageID(ctx context.Context, messageID int64) (*entities.GroupWagerDetail, error) {
	detail, err := s.groupWagerRepo.GetDetailByMessageID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager detail: %w", err)
	}

	return detail, nil
}

// GetActiveGroupWagersByUser returns active group wagers where user is participating
func (s *groupWagerService) GetActiveGroupWagersByUser(ctx context.Context, discordID int64) ([]*entities.GroupWager, error) {
	wagers, err := s.groupWagerRepo.GetActiveByUser(ctx, discordID)
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
	// Get the group wager detail
	detail, err := s.groupWagerRepo.GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil || detail.Wager == nil {
		return fmt.Errorf("group wager not found")
	}

	groupWager := detail.Wager

	// Update message and channel IDs
	groupWager.MessageID = messageID
	groupWager.ChannelID = channelID

	// Save the update
	if err := s.groupWagerRepo.Update(ctx, groupWager); err != nil {
		return fmt.Errorf("failed to update group wager: %w", err)
	}

	return nil
}

// TransitionExpiredWagers finds and transitions active wagers to pending_resolution once their betting window is exhausted
func (s *groupWagerService) TransitionExpiredWagers(ctx context.Context) error {
	// Find expired active wagers
	expiredWagers, err := s.groupWagerRepo.GetExpiredActiveWagers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get expired active wagers: %w", err)
	}

	// Transition each wager
	for _, wager := range expiredWagers {
		oldState := wager.State
		wager.State = entities.GroupWagerStatePendingResolution

		// Update the wager
		if err := s.groupWagerRepo.Update(ctx, wager); err != nil {
			return fmt.Errorf("failed to update wager %d to pending_resolution: %w", wager.ID, err)
		}

		// Publish state change event
		if err := s.eventPublisher.Publish(events.GroupWagerStateChangeEvent{
			GroupWagerID: wager.ID,
			GuildID:      wager.GuildID,
			OldState:     string(oldState),
			NewState:     string(wager.State),
			MessageID:    wager.MessageID,
			ChannelID:    wager.ChannelID,
		}); err != nil {
			log.WithError(err).Error("Failed to publish group wager state change event")
		}
	}

	return nil
}

// CancelGroupWager cancels an active group wager
func (s *groupWagerService) CancelGroupWager(ctx context.Context, groupWagerID int64, cancellerID *int64) error {
	// Get the group wager detail
	detail, err := s.groupWagerRepo.GetDetailByID(ctx, groupWagerID)
	if err != nil {
		return fmt.Errorf("failed to get group wager detail: %w", err)
	}
	if detail == nil || detail.Wager == nil {
		return fmt.Errorf("group wager not found")
	}

	groupWager := detail.Wager

	// Check if canceller is authorized (creator or resolver)
	// Allow system cancellation when cancellerID is nil
	if cancellerID != nil {
		isCreator := groupWager.CreatorDiscordID != nil && *cancellerID == *groupWager.CreatorDiscordID
		isResolver := s.IsResolver(*cancellerID)
		if !isCreator && !isResolver {
			return fmt.Errorf("only the creator or a resolver can cancel a group wager")
		}
	}

	// Check if wager can be cancelled (only active or pending_resolution states)
	if groupWager.State != entities.GroupWagerStateActive && groupWager.State != entities.GroupWagerStatePendingResolution {
		return fmt.Errorf("can only cancel active or pending resolution group wagers")
	}

	// Update state to cancelled
	oldState := groupWager.State
	groupWager.State = entities.GroupWagerStateCancelled

	// Save the update
	if err := s.groupWagerRepo.Update(ctx, groupWager); err != nil {
		return fmt.Errorf("failed to update group wager: %w", err)
	}

	// Publish state change event
	if err := s.eventPublisher.Publish(events.GroupWagerStateChangeEvent{
		GroupWagerID: groupWager.ID,
		GuildID:      groupWager.GuildID,
		OldState:     string(oldState),
		NewState:     string(groupWager.State),
		MessageID:    groupWager.MessageID,
		ChannelID:    groupWager.ChannelID,
	}); err != nil {
		log.WithError(err).Error("Failed to publish group wager state change event")
	}

	return nil
}
