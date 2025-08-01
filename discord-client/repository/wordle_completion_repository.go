package repository

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/database"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"

	"github.com/jackc/pgx/v5"
)

// wordleCompletionDB is a local struct for database mapping
type wordleCompletionDB struct {
	ID          int64     `db:"id"`
	DiscordID   int64     `db:"discord_id"`
	GuildID     int64     `db:"guild_id"`
	GuessCount  int       `db:"guess_count"`
	CompletedAt time.Time `db:"completed_at"`
	CreatedAt   time.Time `db:"created_at"`
}

// toDomain converts the database struct to the domain model
func (w *wordleCompletionDB) toDomain() (*entities.WordleCompletion, error) {
	score, err := entities.NewWordleScore(w.GuessCount)
	if err != nil {
		return nil, fmt.Errorf("invalid wordle score data: %w", err)
	}

	return &entities.WordleCompletion{
		ID:          w.ID,
		DiscordID:   w.DiscordID,
		GuildID:     w.GuildID,
		Score:       score,
		CompletedAt: w.CompletedAt,
		CreatedAt:   w.CreatedAt,
	}, nil
}

// wordleCompletionRepository implements services.WordleCompletionRepository
type wordleCompletionRepository struct {
	q       Queryable
	guildID int64
}

// NewWordleCompletionRepository creates a new wordle completion repository
func NewWordleCompletionRepository(db *database.DB) interfaces.WordleCompletionRepository {
	return &wordleCompletionRepository{q: db.Pool}
}

// NewWordleCompletionRepositoryScoped creates a new wordle completion repository with guild scope
func NewWordleCompletionRepositoryScoped(tx Queryable, guildID int64) interfaces.WordleCompletionRepository {
	return &wordleCompletionRepository{
		q:       tx,
		guildID: guildID,
	}
}

// Create creates a new wordle completion record
func (r *wordleCompletionRepository) Create(ctx context.Context, completion *entities.WordleCompletion) error {
	query := `
		INSERT INTO wordle_completions (discord_id, guild_id, guess_count, completed_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := r.q.QueryRow(ctx, query,
		completion.DiscordID,
		r.guildID, // Use repository's guild scope
		completion.Score.Guesses,
		completion.CompletedAt,
	).Scan(&completion.ID, &completion.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create wordle completion: %w", err)
	}

	// Update the completion's guild ID to match what was actually stored
	completion.GuildID = r.guildID

	return nil
}

// GetByUserToday retrieves today's completion for a specific user
func (r *wordleCompletionRepository) GetByUserToday(ctx context.Context, discordID, guildID int64) (*entities.WordleCompletion, error) {
	query := `
		SELECT id, discord_id, guild_id, guess_count, completed_at, created_at
		FROM wordle_completions
		WHERE discord_id = $1 AND guild_id = $2 AND DATE(completed_at) = CURRENT_DATE`

	var dbCompletion wordleCompletionDB
	err := r.q.QueryRow(ctx, query, discordID, r.guildID).Scan(
		&dbCompletion.ID,
		&dbCompletion.DiscordID,
		&dbCompletion.GuildID,
		&dbCompletion.GuessCount,
		&dbCompletion.CompletedAt,
		&dbCompletion.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get today's wordle completion: %w", err)
	}

	return dbCompletion.toDomain()
}

// GetRecentCompletions returns recent completions for streak calculation
func (r *wordleCompletionRepository) GetRecentCompletions(ctx context.Context, discordID, guildID int64, limit int) ([]*entities.WordleCompletion, error) {
	query := `
		SELECT id, discord_id, guild_id, guess_count, completed_at, created_at
		FROM wordle_completions
		WHERE discord_id = $1 AND guild_id = $2
		ORDER BY completed_at DESC
		LIMIT $3`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent wordle completions: %w", err)
	}
	defer rows.Close()

	var completions []*entities.WordleCompletion
	for rows.Next() {
		var dbCompletion wordleCompletionDB
		err := rows.Scan(
			&dbCompletion.ID,
			&dbCompletion.DiscordID,
			&dbCompletion.GuildID,
			&dbCompletion.GuessCount,
			&dbCompletion.CompletedAt,
			&dbCompletion.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wordle completion: %w", err)
		}

		completion, err := dbCompletion.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert wordle completion: %w", err)
		}
		completions = append(completions, completion)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wordle completions: %w", err)
	}

	return completions, nil
}

// GetTodaysCompletions retrieves all completions for today in this repository's guild
func (r *wordleCompletionRepository) GetTodaysCompletions(ctx context.Context) ([]*entities.WordleCompletion, error) {
	query := `
		SELECT id, discord_id, guild_id, guess_count, completed_at, created_at
		FROM wordle_completions
		WHERE guild_id = $1 AND DATE(completed_at) = CURRENT_DATE
		ORDER BY guess_count ASC, completed_at ASC`

	rows, err := r.q.Query(ctx, query, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to query today's wordle completions: %w", err)
	}
	defer rows.Close()

	var completions []*entities.WordleCompletion
	for rows.Next() {
		var dbCompletion wordleCompletionDB
		err := rows.Scan(
			&dbCompletion.ID,
			&dbCompletion.DiscordID,
			&dbCompletion.GuildID,
			&dbCompletion.GuessCount,
			&dbCompletion.CompletedAt,
			&dbCompletion.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wordle completion: %w", err)
		}

		completion, err := dbCompletion.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert wordle completion: %w", err)
		}
		completions = append(completions, completion)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wordle completions: %w", err)
	}

	return completions, nil
}
