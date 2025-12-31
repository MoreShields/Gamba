package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/domain/entities"
)

// LotteryWinnerRepository implements lottery winner data access
type LotteryWinnerRepository struct {
	q       Queryable
	guildID int64
}

// NewLotteryWinnerRepositoryScoped creates a new lottery winner repository with guild scope
func NewLotteryWinnerRepositoryScoped(tx Queryable, guildID int64) *LotteryWinnerRepository {
	return &LotteryWinnerRepository{
		q:       tx,
		guildID: guildID,
	}
}

// Create creates a new lottery winner record
func (r *LotteryWinnerRepository) Create(ctx context.Context, winner *entities.LotteryWinner) error {
	query := `
		INSERT INTO lottery_winners (draw_id, discord_id, ticket_id, winning_amount, balance_history_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.q.QueryRow(ctx, query,
		winner.DrawID,
		winner.DiscordID,
		winner.TicketID,
		winner.WinningAmount,
		winner.BalanceHistoryID,
	).Scan(&winner.ID, &winner.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create lottery winner: %w", err)
	}

	return nil
}

// GetByDrawID returns all winners for a specific draw
func (r *LotteryWinnerRepository) GetByDrawID(ctx context.Context, drawID int64) ([]*entities.LotteryWinner, error) {
	query := `
		SELECT id, draw_id, discord_id, ticket_id, winning_amount, balance_history_id, created_at
		FROM lottery_winners
		WHERE draw_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.q.Query(ctx, query, drawID)
	if err != nil {
		return nil, fmt.Errorf("failed to get winners for draw %d: %w", drawID, err)
	}
	defer rows.Close()

	var winners []*entities.LotteryWinner
	for rows.Next() {
		var winner entities.LotteryWinner
		err := rows.Scan(
			&winner.ID,
			&winner.DrawID,
			&winner.DiscordID,
			&winner.TicketID,
			&winner.WinningAmount,
			&winner.BalanceHistoryID,
			&winner.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lottery winner: %w", err)
		}
		winners = append(winners, &winner)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate lottery winners: %w", err)
	}

	return winners, nil
}

// GetByUserID returns all wins for a specific user
func (r *LotteryWinnerRepository) GetByUserID(ctx context.Context, discordID int64) ([]*entities.LotteryWinner, error) {
	query := `
		SELECT id, draw_id, discord_id, ticket_id, winning_amount, balance_history_id, created_at
		FROM lottery_winners
		WHERE discord_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.q.Query(ctx, query, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wins for user %d: %w", discordID, err)
	}
	defer rows.Close()

	var winners []*entities.LotteryWinner
	for rows.Next() {
		var winner entities.LotteryWinner
		err := rows.Scan(
			&winner.ID,
			&winner.DrawID,
			&winner.DiscordID,
			&winner.TicketID,
			&winner.WinningAmount,
			&winner.BalanceHistoryID,
			&winner.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lottery winner: %w", err)
		}
		winners = append(winners, &winner)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate lottery winners: %w", err)
	}

	return winners, nil
}
