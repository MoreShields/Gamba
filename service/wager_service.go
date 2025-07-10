package service

import (
	"context"
	"fmt"
	"gambler/models"
	"time"

	log "github.com/sirupsen/logrus"
)

type wagerService struct {
	uowFactory UnitOfWorkFactory
}

// NewWagerService creates a new wager service
func NewWagerService(uowFactory UnitOfWorkFactory) WagerService {
	return &wagerService{
		uowFactory: uowFactory,
	}
}

// ProposeWager creates a new wager proposal
func (s *wagerService) ProposeWager(ctx context.Context, proposerID, targetID int64, amount int64, condition string, messageID, channelID int64) (*models.Wager, error) {
	// Validate inputs
	if proposerID == targetID {
		return nil, fmt.Errorf("cannot create a wager with yourself")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("wager amount must be positive")
	}
	if condition == "" {
		return nil, fmt.Errorf("wager condition cannot be empty")
	}

	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Check if both users exist and have sufficient available balance
	proposer, err := uow.UserRepository().GetByDiscordID(ctx, proposerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposer: %w", err)
	}
	if proposer == nil {
		return nil, fmt.Errorf("proposer not found")
	}
	if proposer.AvailableBalance < amount {
		return nil, fmt.Errorf("insufficient balance: have %d available, need %d", proposer.AvailableBalance, amount)
	}

	target, err := uow.UserRepository().GetByDiscordID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w", err)
	}
	if target == nil {
		return nil, fmt.Errorf("target user not found")
	}
	if target.AvailableBalance < amount {
		return nil, fmt.Errorf("target user has insufficient balance: they have %d available, need %d", target.AvailableBalance, amount)
	}

	// Create the wager
	wager := &models.Wager{
		ProposerDiscordID: proposerID,
		TargetDiscordID:   targetID,
		Amount:            amount,
		Condition:         condition,
		State:             models.WagerStateProposed,
		MessageID:         &messageID,
		ChannelID:         &channelID,
	}

	if err := uow.WagerRepository().Create(ctx, wager); err != nil {
		return nil, fmt.Errorf("failed to create wager: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return wager, nil
}

// RespondToWager handles accepting or declining a wager
func (s *wagerService) RespondToWager(ctx context.Context, wagerID int64, responderID int64, accept bool) (*models.Wager, error) {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get the wager
	wager, err := uow.WagerRepository().GetByID(ctx, wagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager: %w", err)
	}
	if wager == nil {
		return nil, fmt.Errorf("wager not found")
	}

	// Validate the responder
	if wager.TargetDiscordID != responderID {
		return nil, fmt.Errorf("only the target user can respond to this wager")
	}
	if wager.State != models.WagerStateProposed {
		return nil, fmt.Errorf("wager is not in proposed state")
	}

	// Update wager state based on response
	now := time.Now()
	if accept {
		// Double-check both users still have sufficient balance
		proposer, err := uow.UserRepository().GetByDiscordID(ctx, wager.ProposerDiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get proposer: %w", err)
		}
		if proposer.AvailableBalance < wager.Amount {
			return nil, fmt.Errorf("proposer no longer has sufficient balance")
		}

		target, err := uow.UserRepository().GetByDiscordID(ctx, wager.TargetDiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get target: %w", err)
		}
		if target.AvailableBalance < wager.Amount {
			return nil, fmt.Errorf("you no longer have sufficient balance")
		}

		wager.State = models.WagerStateVoting
		wager.AcceptedAt = &now
	} else {
		wager.State = models.WagerStateDeclined
	}

	// Update the wager
	if err := uow.WagerRepository().Update(ctx, wager); err != nil {
		return nil, fmt.Errorf("failed to update wager: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return wager, nil
}

// CastVote records or updates a vote on a wager
func (s *wagerService) CastVote(ctx context.Context, wagerID int64, voterID int64, voteForID int64) (*models.WagerVote, *models.VoteCount, error) {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get the wager
	wager, err := uow.WagerRepository().GetByID(ctx, wagerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get wager: %w", err)
	}
	if wager == nil {
		return nil, nil, fmt.Errorf("wager not found")
	}

	// Validate the vote
	if wager.State != models.WagerStateVoting {
		return nil, nil, fmt.Errorf("wager is not open for voting")
	}
	if wager.IsParticipant(voterID) {
		return nil, nil, fmt.Errorf("participants cannot vote on their own wager")
	}
	if !wager.IsParticipant(voteForID) {
		return nil, nil, fmt.Errorf("can only vote for one of the participants")
	}

	// Create or update the vote
	vote := &models.WagerVote{
		WagerID:          wagerID,
		VoterDiscordID:   voterID,
		VoteForDiscordID: voteForID,
	}

	if err := uow.WagerVoteRepository().CreateOrUpdate(ctx, vote); err != nil {
		return nil, nil, fmt.Errorf("failed to record vote: %w", err)
	}

	// Get updated vote counts
	voteCounts, err := uow.WagerVoteRepository().GetVoteCounts(ctx, wagerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get vote counts: %w", err)
	}

	// Check if we have a majority winner
	if voteCounts.HasMajority() {
		winnerID := voteCounts.GetMajorityWinnerID(wager.ProposerDiscordID, wager.TargetDiscordID)
		if winnerID != 0 {
			// Resolve the wager
			if err := s.resolveWager(ctx, uow, wager, winnerID); err != nil {
				return nil, nil, fmt.Errorf("failed to resolve wager: %w", err)
			}
		}
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return vote, voteCounts, nil
}

// resolveWager handles the resolution of a wager (called within a transaction)
func (s *wagerService) resolveWager(ctx context.Context, uow UnitOfWork, wager *models.Wager, winnerID int64) error {
	// Determine loser
	loserID := wager.GetOpponent(winnerID)
	if loserID == 0 {
		return fmt.Errorf("invalid winner ID")
	}

	// Get current balances
	winner, err := uow.UserRepository().GetByDiscordID(ctx, winnerID)
	if err != nil {
		return fmt.Errorf("failed to get winner: %w", err)
	}

	loser, err := uow.UserRepository().GetByDiscordID(ctx, loserID)
	if err != nil {
		return fmt.Errorf("failed to get loser: %w", err)
	}

	// Transfer funds from loser to winner
	// First, deduct from loser
	if err := uow.UserRepository().DeductBalance(ctx, loserID, wager.Amount); err != nil {
		return fmt.Errorf("failed to deduct from loser: %w", err)
	}

	// Then, add to winner
	if err := uow.UserRepository().AddBalance(ctx, winnerID, wager.Amount); err != nil {
		return fmt.Errorf("failed to add to winner: %w", err)
	}

	// Create balance history for winner
	winnerHistory := &models.BalanceHistory{
		DiscordID:       winnerID,
		BalanceBefore:   winner.Balance,
		BalanceAfter:    winner.Balance + wager.Amount,
		ChangeAmount:    wager.Amount,
		TransactionType: models.TransactionTypeWagerWin,
		TransactionMetadata: map[string]any{
			"wager_id":  wager.ID,
			"opponent":  loserID,
			"amount":    wager.Amount,
			"condition": wager.Condition,
		},
		RelatedID:   &wager.ID,
		RelatedType: relatedTypePtr(models.RelatedTypeWager),
	}

	if err := RecordBalanceChange(ctx, uow, winnerHistory); err != nil {
		return fmt.Errorf("failed to record winner balance change: %w", err)
	}

	// Create balance history for loser
	loserHistory := &models.BalanceHistory{
		DiscordID:       loserID,
		BalanceBefore:   loser.Balance,
		BalanceAfter:    loser.Balance - wager.Amount,
		ChangeAmount:    -wager.Amount,
		TransactionType: models.TransactionTypeWagerLoss,
		TransactionMetadata: map[string]any{
			"wager_id":  wager.ID,
			"opponent":  winnerID,
			"amount":    wager.Amount,
			"condition": wager.Condition,
		},
		RelatedID:   &wager.ID,
		RelatedType: relatedTypePtr(models.RelatedTypeWager),
	}

	if err := RecordBalanceChange(ctx, uow, loserHistory); err != nil {
		return fmt.Errorf("failed to record loser balance change: %w", err)
	}

	// Update wager with resolution details
	now := time.Now()
	wager.State = models.WagerStateResolved
	wager.WinnerDiscordID = &winnerID
	wager.WinnerBalanceHistoryID = &winnerHistory.ID
	wager.LoserBalanceHistoryID = &loserHistory.ID
	wager.ResolvedAt = &now

	if err := uow.WagerRepository().Update(ctx, wager); err != nil {
		return fmt.Errorf("failed to update resolved wager: %w", err)
	}

	return nil
}

// GetWagerByID retrieves a wager by ID
func (s *wagerService) GetWagerByID(ctx context.Context, wagerID int64) (*models.Wager, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	wager, err := uow.WagerRepository().GetByID(ctx, wagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager: %w", err)
	}

	return wager, nil
}

// GetWagerByMessageID retrieves a wager by message ID
func (s *wagerService) GetWagerByMessageID(ctx context.Context, messageID int64) (*models.Wager, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	wager, err := uow.WagerRepository().GetByMessageID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager: %w", err)
	}

	return wager, nil
}

// GetActiveWagersByUser returns active wagers for a user
func (s *wagerService) GetActiveWagersByUser(ctx context.Context, discordID int64) ([]*models.Wager, error) {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	wagers, err := uow.WagerRepository().GetActiveByUser(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active wagers: %w", err)
	}

	return wagers, nil
}

// CancelWager cancels a proposed wager
func (s *wagerService) CancelWager(ctx context.Context, wagerID int64, cancellerID int64) error {
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	wager, err := uow.WagerRepository().GetByID(ctx, wagerID)
	if err != nil {
		return fmt.Errorf("failed to get wager: %w", err)
	}
	log.Infof("Retrieved wager ID: %d", wager.ID)
	if wager == nil {
		return fmt.Errorf("wager not found")
	}

	if wager.State == models.WagerStateResolved || wager.State == models.WagerStateDeclined {
		return fmt.Errorf("wager ID 5 is already resolved")
	}
	if wager.State == models.WagerStateVoting {
		return fmt.Errorf("you can't cancel an active wager")
	}
	if wager.ProposerDiscordID != cancellerID {
		return fmt.Errorf("you can't cancel someone else's wager")
	}

	wager.State = models.WagerStateDeclined
	if err := uow.WagerRepository().Update(ctx, wager); err != nil {
		return fmt.Errorf("failed to update wager: %w", err)
	}

	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to get a pointer to a RelatedType
func relatedTypePtr(rt models.RelatedType) *models.RelatedType {
	return &rt
}
