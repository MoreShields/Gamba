package service

import (
	"context"
	"fmt"

	"gambler/models"
)

// userService implements the UserService interface
type userService struct {
	userRepo           UserRepository
	balanceHistoryRepo BalanceHistoryRepository
	eventPublisher     EventPublisher
}

// NewUserService creates a new user service
func NewUserService(userRepo UserRepository, balanceHistoryRepo BalanceHistoryRepository, eventPublisher EventPublisher) UserService {
	return &userService{
		userRepo:           userRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		eventPublisher:     eventPublisher,
	}
}

// GetOrCreateUser retrieves an existing user or creates a new one with initial balance
func (s *userService) GetOrCreateUser(ctx context.Context, discordID int64, username string) (*models.User, error) {
	// First try to get existing user
	user, err := s.userRepo.GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// User exists, return it
	if user != nil {
		return user, nil
	}

	// User doesn't exist, create new one with initial balance
	// Database unique constraint on discord_id prevents duplicate users
	user, err = s.userRepo.Create(ctx, discordID, username, InitialBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Record initial balance in history
	history := &models.BalanceHistory{
		DiscordID:       discordID,
		GuildID:         0, // Will be set by repository from UoW's guild scope
		BalanceBefore:   0,
		BalanceAfter:    InitialBalance,
		ChangeAmount:    InitialBalance,
		TransactionType: models.TransactionTypeInitial,
		TransactionMetadata: map[string]any{
			"username": username,
		},
	}

	if err := RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, history); err != nil {
		return nil, fmt.Errorf("failed to record initial balance: %w", err)
	}

	return user, nil
}

// GetCurrentHighRoller returns the user with the highest balance
func (s *userService) GetCurrentHighRoller(ctx context.Context) (*models.User, error) {
	// Get all users
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Find user with highest balance
	var highRoller *models.User
	var maxBalance int64 = 0

	for _, user := range users {
		if user.Balance > maxBalance {
			maxBalance = user.Balance
			highRoller = user
		}
	}

	return highRoller, nil
}

// TransferBetweenUsers transfers amount from sender to recipient
func (s *userService) TransferBetweenUsers(ctx context.Context, fromDiscordID, toDiscordID int64, amount int64, fromUsername, toUsername string) error {
	// Validate inputs
	if amount <= 0 {
		return fmt.Errorf("transfer amount must be positive")
	}
	if fromDiscordID == toDiscordID {
		return fmt.Errorf("cannot transfer to yourself")
	}

	// Get sender user
	fromUser, err := s.userRepo.GetByDiscordID(ctx, fromDiscordID)
	if err != nil {
		return fmt.Errorf("failed to get sender user: %w", err)
	}
	if fromUser == nil {
		return fmt.Errorf("sender user not found")
	}

	// Check if sender has sufficient available balance
	if fromUser.AvailableBalance < amount {
		return fmt.Errorf("insufficient balance: have %d available, need %d", fromUser.AvailableBalance, amount)
	}

	// Get recipient user
	toUser, err := s.userRepo.GetByDiscordID(ctx, toDiscordID)
	if err != nil {
		return fmt.Errorf("failed to get recipient user: %w", err)
	}
	if toUser == nil {
		return fmt.Errorf("recipient user not found")
	}

	// Calculate new balances
	newFromBalance := fromUser.Balance - amount
	newToBalance := toUser.Balance + amount

	// Update sender balance
	if err := s.userRepo.UpdateBalance(ctx, fromDiscordID, newFromBalance); err != nil {
		return fmt.Errorf("failed to update sender balance: %w", err)
	}

	// Update recipient balance
	if err := s.userRepo.UpdateBalance(ctx, toDiscordID, newToBalance); err != nil {
		return fmt.Errorf("failed to update recipient balance: %w", err)
	}

	// Create balance history record for sender (outgoing transfer)
	fromHistory := &models.BalanceHistory{
		DiscordID:       fromDiscordID,
		GuildID:         0, // Will be set by repository from UoW's guild scope
		BalanceBefore:   fromUser.Balance,
		BalanceAfter:    newFromBalance,
		ChangeAmount:    -amount,
		TransactionType: models.TransactionTypeTransferOut,
		TransactionMetadata: map[string]any{
			"recipient_discord_id": toDiscordID,
			"recipient_username":   toUsername,
			"transfer_amount":      amount,
		},
	}

	if err := RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, fromHistory); err != nil {
		return fmt.Errorf("failed to record sender balance change: %w", err)
	}

	// Create balance history record for recipient (incoming transfer)
	toHistory := &models.BalanceHistory{
		DiscordID:       toDiscordID,
		GuildID:         0, // Will be set by repository from UoW's guild scope
		BalanceBefore:   toUser.Balance,
		BalanceAfter:    newToBalance,
		ChangeAmount:    amount,
		TransactionType: models.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"sender_discord_id": fromDiscordID,
			"sender_username":   fromUsername,
			"transfer_amount":   amount,
		},
	}

	if err := RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, toHistory); err != nil {
		return fmt.Errorf("failed to record recipient balance change: %w", err)
	}

	return nil
}
