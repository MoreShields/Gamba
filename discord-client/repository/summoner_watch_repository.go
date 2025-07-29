package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/models"
)

// SummonerWatchRepository implements the SummonerWatchRepository interface
type SummonerWatchRepository struct {
	q       Queryable
	guildID int64
}

// NewSummonerWatchRepository creates a new summoner watch repository
func NewSummonerWatchRepository(db *database.DB) *SummonerWatchRepository {
	return &SummonerWatchRepository{q: db.Pool}
}

// newSummonerWatchRepository creates a new summoner watch repository with a transaction and guild scope
func NewSummonerWatchRepositoryScoped(tx Queryable, guildID int64) *SummonerWatchRepository {
	return &SummonerWatchRepository{
		q:       tx,
		guildID: guildID,
	}
}

// CreateWatch creates a new summoner watch for a guild
// Handles upsert of summoner and creation of watch relationship
func (r *SummonerWatchRepository) CreateWatch(ctx context.Context, guildID int64, summonerName, tagLine string) (*models.SummonerWatchDetail, error) {
	// First, upsert the summoner
	summonerQuery := `
		INSERT INTO summoners (game_name, tag_line)
		VALUES (LOWER($1), LOWER($2))
		ON CONFLICT (LOWER(game_name), LOWER(tag_line)) 
		DO UPDATE SET updated_at = NOW()
		RETURNING id, game_name, tag_line, created_at, updated_at`

	var summoner models.Summoner
	err := r.q.QueryRow(ctx, summonerQuery, summonerName, tagLine).Scan(
		&summoner.ID, &summoner.SummonerName, &summoner.TagLine,
		&summoner.CreatedAt, &summoner.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert summoner %s#%s: %w", summonerName, tagLine, err)
	}

	// Then, create the guild watch relationship
	watchQuery := `
		INSERT INTO guild_summoner_watches (guild_id, summoner_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, summoner_id) 
		DO UPDATE SET created_at = guild_summoner_watches.created_at
		RETURNING id, guild_id, summoner_id, created_at`

	var watch models.GuildSummonerWatch
	err = r.q.QueryRow(ctx, watchQuery, guildID, summoner.ID).Scan(
		&watch.ID, &watch.GuildID, &watch.SummonerID, &watch.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create guild summoner watch for guild %d, summoner %d: %w", guildID, summoner.ID, err)
	}

	// Return the combined detail view
	return &models.SummonerWatchDetail{
		GuildID:      watch.GuildID,
		WatchedAt:    watch.CreatedAt,
		SummonerName: summoner.SummonerName,
		TagLine:      summoner.TagLine,
		CreatedAt:    summoner.CreatedAt,
		UpdatedAt:    summoner.UpdatedAt,
	}, nil
}

// GetWatchesByGuild returns all summoner watches for a specific guild
func (r *SummonerWatchRepository) GetWatchesByGuild(ctx context.Context, guildID int64) ([]*models.SummonerWatchDetail, error) {
	query := `
		SELECT 
			gsw.guild_id,
			gsw.created_at as watched_at,
			s.game_name,
			s.tag_line,
			s.created_at,
			s.updated_at
		FROM guild_summoner_watches gsw
		JOIN summoners s ON gsw.summoner_id = s.id
		WHERE gsw.guild_id = $1
		ORDER BY gsw.created_at DESC`

	rows, err := r.q.Query(ctx, query, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watches for guild %d: %w", guildID, err)
	}
	defer rows.Close()

	var watches []*models.SummonerWatchDetail
	for rows.Next() {
		var watch models.SummonerWatchDetail
		err := rows.Scan(
			&watch.GuildID, &watch.WatchedAt,
			&watch.SummonerName, &watch.TagLine,
			&watch.CreatedAt, &watch.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch detail: %w", err)
		}
		watches = append(watches, &watch)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over watch rows: %w", err)
	}

	return watches, nil
}

// GetGuildsWatchingSummoner returns all guild-summoner watch relationships for a specific summoner
func (r *SummonerWatchRepository) GetGuildsWatchingSummoner(ctx context.Context, summonerName, tagLine string) ([]*models.GuildSummonerWatch, error) {
	query := `
		SELECT gsw.id, gsw.guild_id, gsw.summoner_id, gsw.created_at
		FROM guild_summoner_watches gsw
		JOIN summoners s ON gsw.summoner_id = s.id
		WHERE LOWER(s.game_name) = LOWER($1) AND LOWER(s.tag_line) = LOWER($2)
		ORDER BY gsw.created_at DESC`

	rows, err := r.q.Query(ctx, query, summonerName, tagLine)
	if err != nil {
		return nil, fmt.Errorf("failed to get guilds watching summoner %s#%s: %w", summonerName, tagLine, err)
	}
	defer rows.Close()

	var watches []*models.GuildSummonerWatch
	for rows.Next() {
		var watch models.GuildSummonerWatch
		err := rows.Scan(&watch.ID, &watch.GuildID, &watch.SummonerID, &watch.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guild summoner watch: %w", err)
		}
		watches = append(watches, &watch)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over guild watch rows: %w", err)
	}

	return watches, nil
}

// DeleteWatch removes a summoner watch for a guild
func (r *SummonerWatchRepository) DeleteWatch(ctx context.Context, guildID int64, summonerName, tagLine string) error {
	query := `
		DELETE FROM guild_summoner_watches 
		WHERE guild_id = $1 
		AND summoner_id = (
			SELECT id FROM summoners 
			WHERE LOWER(game_name) = LOWER($2) AND LOWER(tag_line) = LOWER($3)
		)`
	result, err := r.q.Exec(ctx, query, guildID, summonerName, tagLine)
	if err != nil {
		return fmt.Errorf("failed to delete watch for guild %d, summoner %s#%s: %w", guildID, summonerName, tagLine, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no watch found for guild %d, summoner %s#%s", guildID, summonerName, tagLine)
	}

	return nil
}

// GetWatch retrieves a specific summoner watch for a guild
func (r *SummonerWatchRepository) GetWatch(ctx context.Context, guildID int64, summonerName, tagLine string) (*models.SummonerWatchDetail, error) {
	query := `
		SELECT 
			gsw.guild_id,
			gsw.created_at as watched_at,
			s.game_name,
			s.tag_line,
			s.created_at,
			s.updated_at
		FROM guild_summoner_watches gsw
		JOIN summoners s ON gsw.summoner_id = s.id
		WHERE gsw.guild_id = $1 AND LOWER(s.game_name) = LOWER($2) AND LOWER(s.tag_line) = LOWER($3)`

	var watch models.SummonerWatchDetail
	err := r.q.QueryRow(ctx, query, guildID, summonerName, tagLine).Scan(
		&watch.GuildID, &watch.WatchedAt,
		&watch.SummonerName, &watch.TagLine,
		&watch.CreatedAt, &watch.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get watch for guild %d, summoner %s#%s: %w", guildID, summonerName, tagLine, err)
	}

	return &watch, nil
}
