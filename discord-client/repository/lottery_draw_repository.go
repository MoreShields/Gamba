package repository

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/domain/entities"

	"github.com/jackc/pgx/v5"
)

// LotteryDrawRepository implements lottery draw data access
type LotteryDrawRepository struct {
	q       Queryable
	guildID int64
}

// NewLotteryDrawRepositoryScoped creates a new lottery draw repository with guild scope
func NewLotteryDrawRepositoryScoped(tx Queryable, guildID int64) *LotteryDrawRepository {
	return &LotteryDrawRepository{
		q:       tx,
		guildID: guildID,
	}
}

// GetOrCreateCurrentDraw gets the current open draw or creates a new one
func (r *LotteryDrawRepository) GetOrCreateCurrentDraw(ctx context.Context, guildID int64, nextDrawTime time.Time, difficulty, ticketCost int64) (*entities.LotteryDraw, error) {
	// First try to get existing open draw
	draw, err := r.GetCurrentOpenDraw(ctx, guildID)
	if err != nil {
		return nil, err
	}
	if draw != nil {
		return draw, nil
	}

	// Create new draw
	query := `
		INSERT INTO lottery_draws (guild_id, difficulty, ticket_cost, draw_time, total_pot)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING id, guild_id, difficulty, ticket_cost, winning_number, draw_time,
		          total_pot, completed_at, message_id, channel_id, created_at
	`

	var newDraw entities.LotteryDraw
	err = r.q.QueryRow(ctx, query, guildID, difficulty, ticketCost, nextDrawTime).Scan(
		&newDraw.ID,
		&newDraw.GuildID,
		&newDraw.Difficulty,
		&newDraw.TicketCost,
		&newDraw.WinningNumber,
		&newDraw.DrawTime,
		&newDraw.TotalPot,
		&newDraw.CompletedAt,
		&newDraw.MessageID,
		&newDraw.ChannelID,
		&newDraw.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create lottery draw: %w", err)
	}

	return &newDraw, nil
}

// GetByID retrieves a draw by its ID
func (r *LotteryDrawRepository) GetByID(ctx context.Context, id int64) (*entities.LotteryDraw, error) {
	query := `
		SELECT id, guild_id, difficulty, ticket_cost, winning_number, draw_time,
		       total_pot, completed_at, message_id, channel_id, created_at
		FROM lottery_draws
		WHERE id = $1
	`

	var draw entities.LotteryDraw
	err := r.q.QueryRow(ctx, query, id).Scan(
		&draw.ID,
		&draw.GuildID,
		&draw.Difficulty,
		&draw.TicketCost,
		&draw.WinningNumber,
		&draw.DrawTime,
		&draw.TotalPot,
		&draw.CompletedAt,
		&draw.MessageID,
		&draw.ChannelID,
		&draw.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lottery draw by ID %d: %w", id, err)
	}

	return &draw, nil
}

// GetByIDForUpdate retrieves a draw by ID with row lock for update
func (r *LotteryDrawRepository) GetByIDForUpdate(ctx context.Context, id int64) (*entities.LotteryDraw, error) {
	query := `
		SELECT id, guild_id, difficulty, ticket_cost, winning_number, draw_time,
		       total_pot, completed_at, message_id, channel_id, created_at
		FROM lottery_draws
		WHERE id = $1
		FOR UPDATE
	`

	var draw entities.LotteryDraw
	err := r.q.QueryRow(ctx, query, id).Scan(
		&draw.ID,
		&draw.GuildID,
		&draw.Difficulty,
		&draw.TicketCost,
		&draw.WinningNumber,
		&draw.DrawTime,
		&draw.TotalPot,
		&draw.CompletedAt,
		&draw.MessageID,
		&draw.ChannelID,
		&draw.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lottery draw for update by ID %d: %w", id, err)
	}

	return &draw, nil
}

// Update updates a draw record
func (r *LotteryDrawRepository) Update(ctx context.Context, draw *entities.LotteryDraw) error {
	query := `
		UPDATE lottery_draws
		SET winning_number = $2,
		    total_pot = $3,
		    completed_at = $4,
		    message_id = $5,
		    channel_id = $6
		WHERE id = $1
	`

	result, err := r.q.Exec(ctx, query,
		draw.ID,
		draw.WinningNumber,
		draw.TotalPot,
		draw.CompletedAt,
		draw.MessageID,
		draw.ChannelID,
	)

	if err != nil {
		return fmt.Errorf("failed to update lottery draw %d: %w", draw.ID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("lottery draw with ID %d not found", draw.ID)
	}

	return nil
}

// GetPendingDrawsForTime returns all draws that are due for processing
func (r *LotteryDrawRepository) GetPendingDrawsForTime(ctx context.Context, beforeTime time.Time) ([]*entities.LotteryDraw, error) {
	query := `
		SELECT id, guild_id, difficulty, ticket_cost, winning_number, draw_time,
		       total_pot, completed_at, message_id, channel_id, created_at
		FROM lottery_draws
		WHERE completed_at IS NULL
		  AND draw_time <= $1
		ORDER BY draw_time ASC
	`

	rows, err := r.q.Query(ctx, query, beforeTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending lottery draws: %w", err)
	}
	defer rows.Close()

	var draws []*entities.LotteryDraw
	for rows.Next() {
		var draw entities.LotteryDraw
		err := rows.Scan(
			&draw.ID,
			&draw.GuildID,
			&draw.Difficulty,
			&draw.TicketCost,
			&draw.WinningNumber,
			&draw.DrawTime,
			&draw.TotalPot,
			&draw.CompletedAt,
			&draw.MessageID,
			&draw.ChannelID,
			&draw.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lottery draw: %w", err)
		}
		draws = append(draws, &draw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate lottery draws: %w", err)
	}

	return draws, nil
}

// IncrementPot atomically increments the pot amount with row locking
func (r *LotteryDrawRepository) IncrementPot(ctx context.Context, drawID, amount int64) error {
	query := `
		UPDATE lottery_draws
		SET total_pot = total_pot + $2
		WHERE id = $1
		  AND completed_at IS NULL
	`

	result, err := r.q.Exec(ctx, query, drawID, amount)
	if err != nil {
		return fmt.Errorf("failed to increment pot for draw %d: %w", drawID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("lottery draw %d not found or already completed", drawID)
	}

	return nil
}

// GetCurrentOpenDraw returns the current open draw for a guild if one exists
func (r *LotteryDrawRepository) GetCurrentOpenDraw(ctx context.Context, guildID int64) (*entities.LotteryDraw, error) {
	query := `
		SELECT id, guild_id, difficulty, ticket_cost, winning_number, draw_time,
		       total_pot, completed_at, message_id, channel_id, created_at
		FROM lottery_draws
		WHERE guild_id = $1
		  AND completed_at IS NULL
		ORDER BY draw_time ASC
		LIMIT 1
	`

	var draw entities.LotteryDraw
	err := r.q.QueryRow(ctx, query, guildID).Scan(
		&draw.ID,
		&draw.GuildID,
		&draw.Difficulty,
		&draw.TicketCost,
		&draw.WinningNumber,
		&draw.DrawTime,
		&draw.TotalPot,
		&draw.CompletedAt,
		&draw.MessageID,
		&draw.ChannelID,
		&draw.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current open draw for guild %d: %w", guildID, err)
	}

	return &draw, nil
}

// GetNextPendingDrawTime returns the earliest draw_time of pending draws
func (r *LotteryDrawRepository) GetNextPendingDrawTime(ctx context.Context) (*time.Time, error) {
	query := `
		SELECT MIN(draw_time)
		FROM lottery_draws
		WHERE completed_at IS NULL
	`

	var drawTime *time.Time
	err := r.q.QueryRow(ctx, query).Scan(&drawTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending draw time: %w", err)
	}

	return drawTime, nil
}
