package repository

import (
	"context"
	"fmt"

	"gambler/database"
	"gambler/models"
	"github.com/jackc/pgx/v5"
)

// UserRepository implements the UserRepository interface
type UserRepository struct {
	q queryable
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{q: db.Pool}
}

// newUserRepositoryWithTx creates a new user repository with a transaction
func newUserRepositoryWithTx(tx queryable) *UserRepository {
	return &UserRepository{q: tx}
}

// GetByDiscordID retrieves a user by their Discord ID
func (r *UserRepository) GetByDiscordID(ctx context.Context, discordID int64) (*models.User, error) {
	query := `
		SELECT 
			u.discord_id, 
			u.username, 
			u.balance, 
			u.created_at, 
			u.updated_at,
			u.balance - COALESCE(
				(SELECT SUM(w.amount) 
				 FROM wagers w 
				 WHERE (w.proposer_discord_id = u.discord_id OR w.target_discord_id = u.discord_id)
				   AND w.state IN ('proposed', 'voting')), 
				0
			) - COALESCE(
				(SELECT SUM(gwp.amount)
				 FROM group_wager_participants gwp
				 JOIN group_wagers gw ON gw.id = gwp.group_wager_id
				 WHERE gwp.discord_id = u.discord_id
				   AND gw.state = 'active'),
				0
			) as available_balance
		FROM users u
		WHERE u.discord_id = $1
	`
	
	var user models.User
	err := r.q.QueryRow(ctx, query, discordID).Scan(
		&user.DiscordID,
		&user.Username,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.AvailableBalance,
	)
	
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by discord ID %d: %w", discordID, err)
	}
	
	return &user, nil
}

// Create creates a new user with the initial balance
func (r *UserRepository) Create(ctx context.Context, discordID int64, username string, initialBalance int64) (*models.User, error) {
	query := `
		INSERT INTO users (discord_id, username, balance)
		VALUES ($1, $2, $3)
		RETURNING discord_id, username, balance, created_at, updated_at
	`
	
	var user models.User
	err := r.q.QueryRow(ctx, query, discordID, username, initialBalance).Scan(
		&user.DiscordID,
		&user.Username,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create user with discord ID %d: %w", discordID, err)
	}
	
	// For a new user, available balance equals balance (no pending wagers)
	user.AvailableBalance = user.Balance
	
	return &user, nil
}

// UpdateBalance updates a user's balance atomically
func (r *UserRepository) UpdateBalance(ctx context.Context, discordID int64, newBalance int64) error {
	query := `
		UPDATE users
		SET balance = $1
		WHERE discord_id = $2
	`
	
	result, err := r.q.Exec(ctx, query, newBalance, discordID)
	if err != nil {
		return fmt.Errorf("failed to update balance for user %d: %w", discordID, err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with discord ID %d not found", discordID)
	}
	
	return nil
}

// GetUsersWithPositiveBalance returns all users with balance > 0
func (r *UserRepository) GetUsersWithPositiveBalance(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT 
			u.discord_id, 
			u.username, 
			u.balance, 
			u.created_at, 
			u.updated_at,
			u.balance - COALESCE(
				(SELECT SUM(w.amount) 
				 FROM wagers w 
				 WHERE (w.proposer_discord_id = u.discord_id OR w.target_discord_id = u.discord_id)
				   AND w.state IN ('proposed', 'voting')), 
				0
			) as available_balance
		FROM users u
		WHERE u.balance > 0
		ORDER BY u.balance DESC
	`
	
	rows, err := r.q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with positive balance: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.DiscordID,
			&user.Username,
			&user.Balance,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.AvailableBalance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}
	
	return users, nil
}

// AddBalance adds to a user's balance atomically
func (r *UserRepository) AddBalance(ctx context.Context, discordID int64, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	query := `
		UPDATE users
		SET balance = balance + $1, updated_at = NOW()
		WHERE discord_id = $2
	`
	
	result, err := r.q.Exec(ctx, query, amount, discordID)
	if err != nil {
		return fmt.Errorf("failed to add balance for user %d: %w", discordID, err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with discord ID %d not found", discordID)
	}
	
	return nil
}

// DeductBalance deducts from a user's balance atomically, failing if insufficient funds
func (r *UserRepository) DeductBalance(ctx context.Context, discordID int64, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Update only if the user has sufficient available balance (balance - pending wagers)
	query := `
		UPDATE users u
		SET balance = balance - $1, updated_at = NOW()
		WHERE u.discord_id = $2 
		  AND u.balance - COALESCE(
			  (SELECT SUM(w.amount) 
			   FROM wagers w 
			   WHERE (w.proposer_discord_id = u.discord_id OR w.target_discord_id = u.discord_id)
			     AND w.state IN ('proposed', 'voting')), 
			  0
		  ) - COALESCE(
			  (SELECT SUM(gwp.amount)
			   FROM group_wager_participants gwp
			   JOIN group_wagers gw ON gw.id = gwp.group_wager_id
			   WHERE gwp.discord_id = u.discord_id
			     AND gw.state = 'active'),
			  0
		  ) >= $1
	`
	
	result, err := r.q.Exec(ctx, query, amount, discordID)
	if err != nil {
		return fmt.Errorf("failed to deduct balance for user %d: %w", discordID, err)
	}
	
	if result.RowsAffected() == 0 {
		// Check if user exists or has insufficient available balance
		user, err := r.GetByDiscordID(ctx, discordID)
		if err != nil {
			return fmt.Errorf("failed to check user: %w", err)
		}
		if user == nil {
			return fmt.Errorf("user with discord ID %d not found", discordID)
		}
		return fmt.Errorf("insufficient balance: have %d available, need %d", user.AvailableBalance, amount)
	}
	
	return nil
}

// GetAll returns all users
func (r *UserRepository) GetAll(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT 
			u.discord_id, 
			u.username, 
			u.balance, 
			u.created_at, 
			u.updated_at,
			u.balance - COALESCE(
				(SELECT SUM(w.amount) 
				 FROM wagers w 
				 WHERE (w.proposer_discord_id = u.discord_id OR w.target_discord_id = u.discord_id)
				   AND w.state IN ('proposed', 'voting')), 
				0
			) - COALESCE(
				(SELECT SUM(gwp.amount)
				 FROM group_wager_participants gwp
				 JOIN group_wagers gw ON gw.id = gwp.group_wager_id
				 WHERE gwp.discord_id = u.discord_id
				   AND gw.state = 'active'),
				0
			) as available_balance
		FROM users u
		ORDER BY u.created_at DESC
	`
	
	rows, err := r.q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.DiscordID,
			&user.Username,
			&user.Balance,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.AvailableBalance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}
	
	return users, nil
}