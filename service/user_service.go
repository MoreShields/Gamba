package service

import (
	"context"
	"fmt"

	"gambler/models"
)

// userService implements the UserService interface
type userService struct {
	uowFactory UnitOfWorkFactory
}

// NewUserService creates a new user service
func NewUserService(uowFactory UnitOfWorkFactory) UserService {
	return &userService{
		uowFactory: uowFactory,
	}
}

// GetOrCreateUser retrieves an existing user or creates a new one with initial balance
func (s *userService) GetOrCreateUser(ctx context.Context, discordID int64, username string) (*models.User, error) {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	// First try to get existing user
	user, err := uow.UserRepository().GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// User exists, return it
	if user != nil {
		// No need to commit since we didn't make changes
		return user, nil
	}

	// User doesn't exist, create new one with initial balance
	// Database unique constraint on discord_id prevents duplicate users
	user, err = uow.UserRepository().Create(ctx, discordID, username, InitialBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Record initial balance in history
	history := &models.BalanceHistory{
		DiscordID:       discordID,
		BalanceBefore:   0,
		BalanceAfter:    InitialBalance,
		ChangeAmount:    InitialBalance,
		TransactionType: models.TransactionTypeInitial,
		TransactionMetadata: map[string]any{
			"username": username,
		},
	}

	if err := RecordBalanceChange(ctx, uow, history); err != nil {
		return nil, fmt.Errorf("failed to record initial balance: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by Discord ID
func (s *userService) GetUser(ctx context.Context, discordID int64) (*models.User, error) {
	// Create unit of work for consistency
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	user, err := uow.UserRepository().GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user with discord ID %d not found", discordID)
	}

	// No need to commit since we didn't make changes
	return user, nil
}

// GetCurrentHighRoller returns the user with the highest balance
func (s *userService) GetCurrentHighRoller(ctx context.Context) (*models.User, error) {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	// Get all users
	users, err := uow.UserRepository().GetAll(ctx)
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

	// No need to commit since we didn't make changes
	return highRoller, nil
}
