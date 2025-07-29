package repository

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/database"
	"gambler/discord-client/models"
	"gambler/discord-client/service"
	"github.com/jackc/pgx/v5"
)

type betRepository struct {
	q       queryable
	guildID int64
}

// NewBetRepository creates a new bet repository
func NewBetRepository(db *database.DB) service.BetRepository {
	return &betRepository{q: db.Pool}
}

// newBetRepositoryWithTx creates a new bet repository with a transaction
func newBetRepositoryWithTx(tx queryable) service.BetRepository {
	return &betRepository{q: tx}
}

// newBetRepository creates a new bet repository with a transaction and guild scope
func newBetRepository(tx queryable, guildID int64) service.BetRepository {
	return &betRepository{
		q:       tx,
		guildID: guildID,
	}
}

func (r *betRepository) Create(ctx context.Context, bet *models.Bet) error {
	query := `
		INSERT INTO bets (discord_id, guild_id, amount, win_probability, won, win_amount, balance_history_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.q.QueryRow(ctx, query,
		bet.DiscordID,
		r.guildID, // Use repository's guild scope
		bet.Amount,
		bet.WinProbability,
		bet.Won,
		bet.WinAmount,
		bet.BalanceHistoryID,
	).Scan(&bet.ID, &bet.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create bet: %w", err)
	}

	return nil
}

func (r *betRepository) GetByID(ctx context.Context, id int64) (*models.Bet, error) {
	query := `
		SELECT id, discord_id, guild_id, amount, win_probability, won, win_amount, balance_history_id, created_at
		FROM bets
		WHERE id = $1 AND guild_id = $2`

	var bet models.Bet
	err := r.q.QueryRow(ctx, query, id, r.guildID).Scan(
		&bet.ID,
		&bet.DiscordID,
		&bet.GuildID,
		&bet.Amount,
		&bet.WinProbability,
		&bet.Won,
		&bet.WinAmount,
		&bet.BalanceHistoryID,
		&bet.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bet: %w", err)
	}

	return &bet, nil
}

func (r *betRepository) GetByUser(ctx context.Context, discordID int64, limit int) ([]*models.Bet, error) {
	query := `
		SELECT id, discord_id, guild_id, amount, win_probability, won, win_amount, balance_history_id, created_at
		FROM bets
		WHERE discord_id = $1 AND guild_id = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query bets: %w", err)
	}
	defer rows.Close()

	var bets []*models.Bet
	for rows.Next() {
		var bet models.Bet
		err := rows.Scan(
			&bet.ID,
			&bet.DiscordID,
			&bet.GuildID,
			&bet.Amount,
			&bet.WinProbability,
			&bet.Won,
			&bet.WinAmount,
			&bet.BalanceHistoryID,
			&bet.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, &bet)
	}

	return bets, nil
}

func (r *betRepository) GetStats(ctx context.Context, discordID int64) (*models.BetStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_bets,
			COUNT(CASE WHEN won = true THEN 1 END) as total_wins,
			COUNT(CASE WHEN won = false THEN 1 END) as total_losses,
			COALESCE(SUM(amount), 0) as total_wagered,
			COALESCE(SUM(CASE WHEN won = true THEN win_amount ELSE 0 END), 0) as total_won,
			COALESCE(SUM(CASE WHEN won = false THEN amount ELSE 0 END), 0) as total_lost,
			COALESCE(MAX(CASE WHEN won = true THEN win_amount ELSE 0 END), 0) as biggest_win,
			COALESCE(MAX(CASE WHEN won = false THEN amount ELSE 0 END), 0) as biggest_loss
		FROM bets
		WHERE discord_id = $1 AND guild_id = $2`

	var stats models.BetStats
	err := r.q.QueryRow(ctx, query, discordID, r.guildID).Scan(
		&stats.TotalBets,
		&stats.TotalWins,
		&stats.TotalLosses,
		&stats.TotalWagered,
		&stats.TotalWon,
		&stats.TotalLost,
		&stats.BiggestWin,
		&stats.BiggestLoss,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get bet stats: %w", err)
	}

	return &stats, nil
}

func (r *betRepository) GetByUserSince(ctx context.Context, discordID int64, since time.Time) ([]*models.Bet, error) {
	query := `
		SELECT id, discord_id, guild_id, amount, win_probability, won, win_amount, balance_history_id, created_at
		FROM bets
		WHERE discord_id = $1 AND guild_id = $2 AND created_at >= $3
		ORDER BY created_at DESC`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query bets since %v: %w", since, err)
	}
	defer rows.Close()

	var bets []*models.Bet
	for rows.Next() {
		var bet models.Bet
		err := rows.Scan(
			&bet.ID,
			&bet.DiscordID,
			&bet.GuildID,
			&bet.Amount,
			&bet.WinProbability,
			&bet.Won,
			&bet.WinAmount,
			&bet.BalanceHistoryID,
			&bet.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, &bet)
	}

	return bets, nil
}
