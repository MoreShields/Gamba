package repository

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/database"
	"gambler/discord-client/domain/entities"

	"github.com/jackc/pgx/v5"
)

// availableBalanceSQL is a reusable SQL fragment that calculates available balance
// by subtracting locked amounts in active wagers and group wagers from total balance
const availableBalanceSQL = `uga.balance - COALESCE(
	(SELECT SUM(w.amount) 
	 FROM wagers w 
	 WHERE (w.proposer_discord_id = uga.discord_id OR w.target_discord_id = uga.discord_id)
	   AND w.guild_id = uga.guild_id
	   AND w.state = 'voting'), 
	0
) - COALESCE(
	(SELECT SUM(gwp.amount)
	 FROM group_wager_participants gwp
	 JOIN group_wagers gw ON gw.id = gwp.group_wager_id
	 WHERE gwp.discord_id = uga.discord_id
	   AND gw.guild_id = uga.guild_id
	   AND gw.state IN ('active', 'pending_resolution')),
	0
)`

// UserRepository implements the UserRepository interface
type UserRepository struct {
	q       Queryable
	guildID int64
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{q: db.Pool}
}

// NewUserRepositoryScoped creates a new user repository with a transaction and guild scope
func NewUserRepositoryScoped(tx Queryable, guildID int64) *UserRepository {
	return &UserRepository{
		q:       tx,
		guildID: guildID,
	}
}

// GetByDiscordID retrieves a user by their Discord ID in the current guild
func (r *UserRepository) GetByDiscordID(ctx context.Context, discordID int64) (*entities.User, error) {
	query := `
		SELECT 
			uga.id,
			uga.discord_id,
			uga.guild_id,
			uga.balance,
			uga.created_at,
			uga.updated_at,
			u.username,
			` + availableBalanceSQL + ` as available_balance
		FROM user_guild_accounts uga
		JOIN users u ON uga.discord_id = u.discord_id
		WHERE uga.discord_id = $1 AND uga.guild_id = $2
	`

	var account entities.UserGuildAccount
	var username string
	err := r.q.QueryRow(ctx, query, discordID, r.guildID).Scan(
		&account.ID,
		&account.DiscordID,
		&account.GuildID,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
		&username,
		&account.AvailableBalance,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by discord ID %d in guild %d: %w", discordID, r.guildID, err)
	}

	// Return User model with balance information from guild account
	user := &entities.User{
		DiscordID:        account.DiscordID,
		Username:         username,
		Balance:          account.Balance,
		AvailableBalance: account.AvailableBalance,
		CreatedAt:        account.CreatedAt,
		UpdatedAt:        account.UpdatedAt,
	}

	return user, nil
}

// Create creates a new user with the initial balance in the current guild
func (r *UserRepository) Create(ctx context.Context, discordID int64, username string, initialBalance int64) (*entities.User, error) {
	// First ensure the user exists in the users table
	userQuery := `
		INSERT INTO users (discord_id, username)
		VALUES ($1, $2)
		ON CONFLICT (discord_id) DO UPDATE SET username = EXCLUDED.username
		RETURNING created_at, updated_at
	`

	var userCreatedAt, userUpdatedAt time.Time
	err := r.q.QueryRow(ctx, userQuery, discordID, username).Scan(&userCreatedAt, &userUpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update user %d: %w", discordID, err)
	}

	// Then create the guild account
	accountQuery := `
		INSERT INTO user_guild_accounts (discord_id, guild_id, balance)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	var accountID int64
	var accountCreatedAt, accountUpdatedAt time.Time
	err = r.q.QueryRow(ctx, accountQuery, discordID, r.guildID, initialBalance).Scan(
		&accountID,
		&accountCreatedAt,
		&accountUpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user guild account for discord ID %d in guild %d: %w", discordID, r.guildID, err)
	}

	// Return User model with balance information
	user := &entities.User{
		DiscordID:        discordID,
		Username:         username,
		Balance:          initialBalance,
		AvailableBalance: initialBalance, // For a new user, available balance equals balance
		CreatedAt:        accountCreatedAt,
		UpdatedAt:        accountUpdatedAt,
	}

	return user, nil
}

// UpdateBalance updates a user's balance atomically
func (r *UserRepository) UpdateBalance(ctx context.Context, discordID int64, newBalance int64) error {
	query := `
		UPDATE user_guild_accounts
		SET balance = $1, updated_at = NOW()
		WHERE discord_id = $2 AND guild_id = $3
	`
	result, err := r.q.Exec(ctx, query, newBalance, discordID, r.guildID)
	if err != nil {
		return fmt.Errorf("failed to update balance for user %d in guild %d: %w", discordID, r.guildID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with discord ID %d not found in guild %d", discordID, r.guildID)
	}

	return nil
}

// GetUsersWithPositiveBalance returns all users with balance > 0 in the current guild
func (r *UserRepository) GetUsersWithPositiveBalance(ctx context.Context) ([]*entities.User, error) {
	query := `
		SELECT 
			uga.discord_id,
			u.username,
			uga.balance,
			uga.created_at,
			uga.updated_at,
			` + availableBalanceSQL + ` as available_balance
		FROM user_guild_accounts uga
		JOIN users u ON uga.discord_id = u.discord_id
		WHERE uga.guild_id = $1 AND uga.balance > 0
		ORDER BY uga.balance DESC
	`

	rows, err := r.q.Query(ctx, query, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with positive balance in guild %d: %w", r.guildID, err)
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		var user entities.User
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

// GetAll returns all users in the current guild
func (r *UserRepository) GetAll(ctx context.Context) ([]*entities.User, error) {
	query := `
		SELECT 
			uga.discord_id,
			u.username,
			uga.balance,
			uga.created_at,
			uga.updated_at,
			` + availableBalanceSQL + ` as available_balance
		FROM user_guild_accounts uga
		JOIN users u ON uga.discord_id = u.discord_id
		WHERE uga.guild_id = $1
		ORDER BY uga.created_at DESC
	`

	rows, err := r.q.Query(ctx, query, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users in guild %d: %w", r.guildID, err)
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		var user entities.User
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

// GetScoreboardData returns all user scoreboard data in a single optimized query
func (r *UserRepository) GetScoreboardData(ctx context.Context) ([]*entities.ScoreboardEntry, int64, error) {
	query := `
		WITH user_wager_stats AS (
			SELECT 
				COALESCE(proposer_discord_id, target_discord_id) as discord_id,
				COUNT(*) FILTER (WHERE state IN ('proposed', 'voting')) as active_wagers,
				COUNT(*) as total_wagers,
				COUNT(*) FILTER (WHERE state = 'resolved') as resolved_wagers,
				COUNT(*) FILTER (WHERE state = 'resolved' AND winner_discord_id = COALESCE(proposer_discord_id, target_discord_id)) as won_wagers
			FROM wagers
			WHERE guild_id = $1
			GROUP BY COALESCE(proposer_discord_id, target_discord_id)
		),
		user_bet_stats AS (
			SELECT 
				b.discord_id,
				COUNT(*) as total_bets,
				COUNT(*) FILTER (WHERE b.won = true) as won_bets
			FROM bets b
			JOIN user_guild_accounts uga ON uga.discord_id = b.discord_id
			WHERE uga.guild_id = $1 AND b.guild_id = $1
			GROUP BY b.discord_id
		),
		user_volume_stats AS (
			SELECT 
				discord_id,
				SUM(ABS(change_amount)) as total_volume,
				SUM(ABS(change_amount)) FILTER (WHERE transaction_type = 'transfer_out') as total_donations
			FROM balance_history
			WHERE guild_id = $1
			GROUP BY discord_id
		),
		server_totals AS (
			SELECT COALESCE(SUM(balance), 0) as total_bits
			FROM user_guild_accounts
			WHERE guild_id = $1
		)
		SELECT 
			uga.discord_id,
			u.username,
			uga.balance as total_balance,
			` + availableBalanceSQL + ` as available_balance,
			COALESCE(ws.active_wagers, 0) as active_wager_count,
			COALESCE(ws.won_wagers::float / NULLIF(ws.resolved_wagers, 0) * 100, 0) as wager_win_rate,
			COALESCE(bs.won_bets::float / NULLIF(bs.total_bets, 0) * 100, 0) as bet_win_rate,
			COALESCE(vs.total_volume, 0) as total_volume,
			COALESCE(vs.total_donations, 0) as total_donations,
			(SELECT total_bits FROM server_totals) as server_total_bits
		FROM user_guild_accounts uga
		JOIN users u ON u.discord_id = uga.discord_id
		LEFT JOIN user_wager_stats ws ON uga.discord_id = ws.discord_id
		LEFT JOIN user_bet_stats bs ON uga.discord_id = bs.discord_id
		LEFT JOIN user_volume_stats vs ON uga.discord_id = vs.discord_id
		WHERE uga.guild_id = $1 AND uga.balance > 0
		ORDER BY uga.balance DESC
	`

	rows, err := r.q.Query(ctx, query, r.guildID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query scoreboard data: %w", err)
	}
	defer rows.Close()

	var entries []*entities.ScoreboardEntry
	var totalBits int64
	rank := 1
	for rows.Next() {
		var entry entities.ScoreboardEntry
		var serverTotal int64
		err := rows.Scan(
			&entry.DiscordID,
			&entry.Username,
			&entry.TotalBalance,
			&entry.AvailableBalance,
			&entry.ActiveWagerCount,
			&entry.WagerWinRate,
			&entry.BetWinRate,
			&entry.TotalVolume,
			&entry.TotalDonations,
			&serverTotal,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan scoreboard entry: %w", err)
		}
		// Store total from first row (same for all rows)
		if rank == 1 {
			totalBits = serverTotal
		}
		entry.Rank = rank
		rank++
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate scoreboard entries: %w", err)
	}

	return entries, totalBits, nil
}