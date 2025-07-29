package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/models"
	"github.com/jackc/pgx/v5"
)

// WagerRepository implements wager data access
type WagerRepository struct {
	q       Queryable
	guildID int64
}

// NewWagerRepository creates a new wager repository
func NewWagerRepository(db *database.DB) *WagerRepository {
	return &WagerRepository{q: db.Pool}
}

// newWagerRepositoryWithTx creates a new wager repository with a transaction
func newWagerRepositoryWithTx(tx Queryable) *WagerRepository {
	return &WagerRepository{q: tx}
}

// newWagerRepository creates a new wager repository with a transaction and guild scope
func NewWagerRepositoryScoped(tx Queryable, guildID int64) *WagerRepository {
	return &WagerRepository{
		q:       tx,
		guildID: guildID,
	}
}

// Create creates a new wager
func (r *WagerRepository) Create(ctx context.Context, wager *models.Wager) error {
	query := `
		INSERT INTO wagers (
			proposer_discord_id, target_discord_id, guild_id, amount, condition, 
			state, message_id, channel_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	err := r.q.QueryRow(ctx, query,
		wager.ProposerDiscordID,
		wager.TargetDiscordID,
		r.guildID, // Use repository's guild scope
		wager.Amount,
		wager.Condition,
		wager.State,
		wager.MessageID,
		wager.ChannelID,
	).Scan(&wager.ID, &wager.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create wager: %w", err)
	}

	return nil
}

// GetByID retrieves a wager by its ID
func (r *WagerRepository) GetByID(ctx context.Context, id int64) (*models.Wager, error) {
	query := `
		SELECT 
			id, proposer_discord_id, target_discord_id, guild_id, amount, condition,
			state, winner_discord_id, winner_balance_history_id, loser_balance_history_id,
			message_id, channel_id, created_at, accepted_at, resolved_at
		FROM wagers
		WHERE id = $1
	`

	var wager models.Wager
	err := r.q.QueryRow(ctx, query, id).Scan(
		&wager.ID,
		&wager.ProposerDiscordID,
		&wager.TargetDiscordID,
		&wager.GuildID,
		&wager.Amount,
		&wager.Condition,
		&wager.State,
		&wager.WinnerDiscordID,
		&wager.WinnerBalanceHistoryID,
		&wager.LoserBalanceHistoryID,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.CreatedAt,
		&wager.AcceptedAt,
		&wager.ResolvedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wager by ID %d: %w", id, err)
	}

	return &wager, nil
}

// GetByMessageID retrieves a wager by its Discord message ID
func (r *WagerRepository) GetByMessageID(ctx context.Context, messageID int64) (*models.Wager, error) {
	query := `
		SELECT 
			id, proposer_discord_id, target_discord_id, guild_id, amount, condition,
			state, winner_discord_id, winner_balance_history_id, loser_balance_history_id,
			message_id, channel_id, created_at, accepted_at, resolved_at
		FROM wagers
		WHERE message_id = $1
	`

	var wager models.Wager
	err := r.q.QueryRow(ctx, query, messageID).Scan(
		&wager.ID,
		&wager.ProposerDiscordID,
		&wager.TargetDiscordID,
		&wager.GuildID,
		&wager.Amount,
		&wager.Condition,
		&wager.State,
		&wager.WinnerDiscordID,
		&wager.WinnerBalanceHistoryID,
		&wager.LoserBalanceHistoryID,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.CreatedAt,
		&wager.AcceptedAt,
		&wager.ResolvedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wager by message ID %d: %w", messageID, err)
	}

	return &wager, nil
}

// Update updates a wager's state and related fields
func (r *WagerRepository) Update(ctx context.Context, wager *models.Wager) error {
	query := `
		UPDATE wagers
		SET state = $2, 
		    winner_discord_id = $3,
		    winner_balance_history_id = $4,
		    loser_balance_history_id = $5,
		    message_id = $6,
		    channel_id = $7,
		    accepted_at = $8,
		    resolved_at = $9
		WHERE id = $1
	`

	result, err := r.q.Exec(ctx, query,
		wager.ID,
		wager.State,
		wager.WinnerDiscordID,
		wager.WinnerBalanceHistoryID,
		wager.LoserBalanceHistoryID,
		wager.MessageID,
		wager.ChannelID,
		wager.AcceptedAt,
		wager.ResolvedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update wager %d: %w", wager.ID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("wager with ID %d not found", wager.ID)
	}

	return nil
}

// GetActiveByUser returns all active wagers for a user (as proposer or target)
func (r *WagerRepository) GetActiveByUser(ctx context.Context, discordID int64) ([]*models.Wager, error) {
	query := `
		SELECT 
			id, proposer_discord_id, target_discord_id, guild_id, amount, condition,
			state, winner_discord_id, winner_balance_history_id, loser_balance_history_id,
			message_id, channel_id, created_at, accepted_at, resolved_at
		FROM wagers
		WHERE (proposer_discord_id = $1 OR target_discord_id = $1)
		  AND guild_id = $2
		  AND state IN ('proposed', 'voting')
		ORDER BY created_at DESC
	`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active wagers for user %d: %w", discordID, err)
	}
	defer rows.Close()

	var wagers []*models.Wager
	for rows.Next() {
		var wager models.Wager
		err := rows.Scan(
			&wager.ID,
			&wager.ProposerDiscordID,
			&wager.TargetDiscordID,
			&wager.GuildID,
			&wager.Amount,
			&wager.Condition,
			&wager.State,
			&wager.WinnerDiscordID,
			&wager.WinnerBalanceHistoryID,
			&wager.LoserBalanceHistoryID,
			&wager.MessageID,
			&wager.ChannelID,
			&wager.CreatedAt,
			&wager.AcceptedAt,
			&wager.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wager: %w", err)
		}
		wagers = append(wagers, &wager)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate wagers: %w", err)
	}

	return wagers, nil
}

// GetAllByUser returns all wagers for a user (including resolved)
func (r *WagerRepository) GetAllByUser(ctx context.Context, discordID int64, limit int) ([]*models.Wager, error) {
	query := `
		SELECT 
			id, proposer_discord_id, target_discord_id, guild_id, amount, condition,
			state, winner_discord_id, winner_balance_history_id, loser_balance_history_id,
			message_id, channel_id, created_at, accepted_at, resolved_at
		FROM wagers
		WHERE (proposer_discord_id = $1 OR target_discord_id = $1)
		  AND guild_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get wagers for user %d: %w", discordID, err)
	}
	defer rows.Close()

	var wagers []*models.Wager
	for rows.Next() {
		var wager models.Wager
		err := rows.Scan(
			&wager.ID,
			&wager.ProposerDiscordID,
			&wager.TargetDiscordID,
			&wager.GuildID,
			&wager.Amount,
			&wager.Condition,
			&wager.State,
			&wager.WinnerDiscordID,
			&wager.WinnerBalanceHistoryID,
			&wager.LoserBalanceHistoryID,
			&wager.MessageID,
			&wager.ChannelID,
			&wager.CreatedAt,
			&wager.AcceptedAt,
			&wager.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wager: %w", err)
		}
		wagers = append(wagers, &wager)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate wagers: %w", err)
	}

	return wagers, nil
}

// GetStats returns wager statistics for a user
func (r *WagerRepository) GetStats(ctx context.Context, discordID int64) (*models.WagerStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_wagers,
			COUNT(CASE WHEN state = 'proposed' THEN 1 END) as total_proposed,
			COUNT(CASE WHEN state = 'voting' THEN 1 END) as total_accepted,
			COUNT(CASE WHEN state = 'declined' THEN 1 END) as total_declined,
			COUNT(CASE WHEN state = 'resolved' THEN 1 END) as total_resolved,
			COUNT(CASE WHEN state = 'resolved' AND winner_discord_id = $1 THEN 1 END) as total_won,
			COUNT(CASE WHEN state = 'resolved' AND winner_discord_id != $1 THEN 1 END) as total_lost,
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(SUM(CASE WHEN state = 'resolved' AND winner_discord_id = $1 THEN amount ELSE 0 END), 0) as total_won_amount,
			COALESCE(MAX(CASE WHEN state = 'resolved' AND winner_discord_id = $1 THEN amount ELSE 0 END), 0) as biggest_win,
			COALESCE(MAX(CASE WHEN state = 'resolved' AND winner_discord_id != $1 THEN amount ELSE 0 END), 0) as biggest_loss
		FROM wagers
		WHERE (proposer_discord_id = $1 OR target_discord_id = $1)
		  AND guild_id = $2`

	var stats models.WagerStats
	err := r.q.QueryRow(ctx, query, discordID, r.guildID).Scan(
		&stats.TotalWagers,
		&stats.TotalProposed,
		&stats.TotalAccepted,
		&stats.TotalDeclined,
		&stats.TotalResolved,
		&stats.TotalWon,
		&stats.TotalLost,
		&stats.TotalAmount,
		&stats.TotalWonAmount,
		&stats.BiggestWin,
		&stats.BiggestLoss,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get wager stats: %w", err)
	}

	return &stats, nil
}
