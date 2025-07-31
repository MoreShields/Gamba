package service

import (
	"context"
	"fmt"
	"gambler/discord-client/models"
	"time"

	log "github.com/sirupsen/logrus"
)

type wagerService struct {
	userRepo           UserRepository
	wagerRepo          WagerRepository
	wagerVoteRepo      WagerVoteRepository
	balanceHistoryRepo BalanceHistoryRepository
	eventPublisher     EventPublisher
}

// NewWagerService creates a new wager service
func NewWagerService(userRepo UserRepository, wagerRepo WagerRepository, wagerVoteRepo WagerVoteRepository, balanceHistoryRepo BalanceHistoryRepository, eventPublisher EventPublisher) WagerService {
	return &wagerService{
		userRepo:           userRepo,
		wagerRepo:          wagerRepo,
		wagerVoteRepo:      wagerVoteRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		eventPublisher:     eventPublisher,
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

	// Check if both users exist and have sufficient available balance
	proposer, err := s.userRepo.GetByDiscordID(ctx, proposerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposer: %w", err)
	}
	if proposer == nil {
		return nil, fmt.Errorf("proposer not found")
	}
	if proposer.AvailableBalance < amount {
		return nil, fmt.Errorf("insufficient balance: have %s available, need %s", FormatShortNotation(proposer.AvailableBalance), FormatShortNotation(amount))
	}

	target, err := s.userRepo.GetByDiscordID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w", err)
	}
	if target == nil {
		return nil, fmt.Errorf("target user not found")
	}
	if target.AvailableBalance < amount {
		return nil, fmt.Errorf("target user has insufficient balance: they have %s available, need %s", FormatShortNotation(target.AvailableBalance), FormatShortNotation(amount))
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

	if err := s.wagerRepo.Create(ctx, wager); err != nil {
		return nil, fmt.Errorf("failed to create wager: %w", err)
	}

	return wager, nil
}

// RespondToWager handles accepting or declining a wager
func (s *wagerService) RespondToWager(ctx context.Context, wagerID int64, responderID int64, accept bool) (*models.Wager, error) {

	// Get the wager
	wager, err := s.wagerRepo.GetByID(ctx, wagerID)
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
		proposer, err := s.userRepo.GetByDiscordID(ctx, wager.ProposerDiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get proposer: %w", err)
		}
		if proposer.AvailableBalance < wager.Amount {
			return nil, fmt.Errorf("proposer no longer has sufficient balance")
		}

		target, err := s.userRepo.GetByDiscordID(ctx, wager.TargetDiscordID)
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
	if err := s.wagerRepo.Update(ctx, wager); err != nil {
		return nil, fmt.Errorf("failed to update wager: %w", err)
	}

	return wager, nil
}

// CastVote records or updates a participant's vote on a wager
func (s *wagerService) CastVote(ctx context.Context, wagerID int64, voterID int64, voteForID int64) (*models.WagerVote, *models.VoteCount, error) {

	// Get the wager
	wager, err := s.wagerRepo.GetByID(ctx, wagerID)
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
	if !wager.IsParticipant(voterID) {
		return nil, nil, fmt.Errorf("only participants can settle their own wager")
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

	if err := s.wagerVoteRepo.CreateOrUpdate(ctx, vote); err != nil {
		return nil, nil, fmt.Errorf("failed to record vote: %w", err)
	}

	// Get updated vote counts
	voteCounts, err := s.wagerVoteRepo.GetVoteCounts(ctx, wagerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get vote counts: %w", err)
	}

	// Check if both participants have agreed on a winner
	winnerID := s.BothParticipantsAgree(wager, voteCounts)
	if winnerID != 0 {
		// Resolve the wager
		if err := s.resolveWager(ctx, wager, winnerID); err != nil {
			return nil, nil, fmt.Errorf("failed to resolve wager: %w", err)
		}
	}

	return vote, voteCounts, nil
}

// BothParticipantsAgree checks if both participants have voted for the same winner
// Returns the winner's Discord ID if they agree, 0 otherwise
func (s *wagerService) BothParticipantsAgree(wager *models.Wager, voteCounts *models.VoteCount) int64 {
	return voteCounts.GetAgreedWinnerID(wager.ProposerDiscordID, wager.TargetDiscordID)
}

// resolveWager handles the resolution of a wager (called within a transaction)
func (s *wagerService) resolveWager(ctx context.Context, wager *models.Wager, winnerID int64) error {
	// Determine loser
	loserID := wager.GetOpponent(winnerID)
	if loserID == 0 {
		return fmt.Errorf("invalid winner ID")
	}

	// Get current balances
	winner, err := s.userRepo.GetByDiscordID(ctx, winnerID)
	if err != nil {
		return fmt.Errorf("failed to get winner: %w", err)
	}

	loser, err := s.userRepo.GetByDiscordID(ctx, loserID)
	if err != nil {
		return fmt.Errorf("failed to get loser: %w", err)
	}

	// Calculate new balances for both users
	newWinnerBalance := winner.Balance + wager.Amount
	newLoserBalance := loser.Balance - wager.Amount

	// Transfer funds from loser to winner
	// Update loser balance
	if err := s.userRepo.UpdateBalance(ctx, loserID, newLoserBalance); err != nil {
		return fmt.Errorf("failed to update loser balance: %w", err)
	}

	// Update winner balance
	if err := s.userRepo.UpdateBalance(ctx, winnerID, newWinnerBalance); err != nil {
		return fmt.Errorf("failed to update winner balance: %w", err)
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

	if err := RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, winnerHistory); err != nil {
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

	if err := RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, loserHistory); err != nil {
		return fmt.Errorf("failed to record loser balance change: %w", err)
	}

	// Update wager with resolution details
	now := time.Now()
	wager.State = models.WagerStateResolved
	wager.WinnerDiscordID = &winnerID
	wager.WinnerBalanceHistoryID = &winnerHistory.ID
	wager.LoserBalanceHistoryID = &loserHistory.ID
	wager.ResolvedAt = &now

	if err := s.wagerRepo.Update(ctx, wager); err != nil {
		return fmt.Errorf("failed to update resolved wager: %w", err)
	}

	return nil
}

// GetWagerByID retrieves a wager by ID
func (s *wagerService) GetWagerByID(ctx context.Context, wagerID int64) (*models.Wager, error) {

	wager, err := s.wagerRepo.GetByID(ctx, wagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager: %w", err)
	}

	return wager, nil
}

// GetWagerByMessageID retrieves a wager by message ID
func (s *wagerService) GetWagerByMessageID(ctx context.Context, messageID int64) (*models.Wager, error) {

	wager, err := s.wagerRepo.GetByMessageID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wager: %w", err)
	}

	return wager, nil
}

// GetActiveWagersByUser returns active wagers for a user
func (s *wagerService) GetActiveWagersByUser(ctx context.Context, discordID int64) ([]*models.Wager, error) {

	wagers, err := s.wagerRepo.GetActiveByUser(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active wagers: %w", err)
	}

	return wagers, nil
}

// CancelWager cancels a proposed wager
func (s *wagerService) CancelWager(ctx context.Context, wagerID int64, cancellerID int64) error {

	wager, err := s.wagerRepo.GetByID(ctx, wagerID)
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
	if err := s.wagerRepo.Update(ctx, wager); err != nil {
		return fmt.Errorf("failed to update wager: %w", err)
	}

	return nil
}

// UpdateMessageIDs updates the message and channel IDs for a wager
func (s *wagerService) UpdateMessageIDs(ctx context.Context, wagerID int64, messageID int64, channelID int64) error {
	wager, err := s.wagerRepo.GetByID(ctx, wagerID)
	if err != nil {
		return fmt.Errorf("failed to get wager: %w", err)
	}
	if wager == nil {
		return fmt.Errorf("wager not found")
	}

	wager.MessageID = &messageID
	wager.ChannelID = &channelID

	if err := s.wagerRepo.Update(ctx, wager); err != nil {
		return fmt.Errorf("failed to update wager: %w", err)
	}

	return nil
}

// Helper function to get a pointer to a RelatedType
func relatedTypePtr(rt models.RelatedType) *models.RelatedType {
	return &rt
}
