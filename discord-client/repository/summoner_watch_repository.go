package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/models"
)

// SummonerWatchRepository implements the SummonerWatchRepository interface
type SummonerWatchRepository struct {
	q       queryable
	guildID int64
}

// NewSummonerWatchRepository creates a new summoner watch repository
func NewSummonerWatchRepository(db *database.DB) *SummonerWatchRepository {
	return &SummonerWatchRepository{q: db.Pool}
}

// newSummonerWatchRepository creates a new summoner watch repository with a transaction and guild scope
func newSummonerWatchRepository(tx queryable, guildID int64) *SummonerWatchRepository {
	return &SummonerWatchRepository{
		q:       tx,
		guildID: guildID,
	}
}

// CreateWatch creates a new summoner watch for a guild
// Handles upsert of summoner and creation of watch relationship
func (r *SummonerWatchRepository) CreateWatch(ctx context.Context, guildID int64, summonerName, region string) (*models.SummonerWatchDetail, error) {
	// First, upsert the summoner
	summonerQuery := `
		INSERT INTO summoners (summoner_name, region)
		VALUES ($1, $2)
		ON CONFLICT (summoner_name, region) 
		DO UPDATE SET updated_at = NOW()
		RETURNING id, summoner_name, region, created_at, updated_at`

	var summoner models.Summoner
	err := r.q.QueryRow(ctx, summonerQuery, summonerName, region).Scan(
		&summoner.ID, &summoner.SummonerName, &summoner.Region,
		&summoner.CreatedAt, &summoner.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert summoner %s in %s: %w", summonerName, region, err)
	}

	// Then, create the guild watch relationship
	watchQuery := `
		INSERT INTO guild_summoner_watches (guild_id, summoner_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, summoner_id) DO NOTHING
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
		WatchID:      watch.ID,
		GuildID:      watch.GuildID,
		WatchedAt:    watch.CreatedAt,
		SummonerID:   summoner.ID,
		SummonerName: summoner.SummonerName,
		Region:       summoner.Region,
		CreatedAt:    summoner.CreatedAt,
		UpdatedAt:    summoner.UpdatedAt,
	}, nil
}

// GetWatchesByGuild returns all summoner watches for a specific guild
func (r *SummonerWatchRepository) GetWatchesByGuild(ctx context.Context, guildID int64) ([]*models.SummonerWatchDetail, error) {
	query := `
		SELECT 
			gsw.id as watch_id,
			gsw.guild_id,
			gsw.created_at as watched_at,
			s.id as summoner_id,
			s.summoner_name,
			s.region,
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
			&watch.WatchID, &watch.GuildID, &watch.WatchedAt,
			&watch.SummonerID, &watch.SummonerName, &watch.Region,
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
func (r *SummonerWatchRepository) GetGuildsWatchingSummoner(ctx context.Context, summonerName, region string) ([]*models.GuildSummonerWatch, error) {
	query := `
		SELECT gsw.id, gsw.guild_id, gsw.summoner_id, gsw.created_at
		FROM guild_summoner_watches gsw
		JOIN summoners s ON gsw.summoner_id = s.id
		WHERE s.summoner_name = $1 AND s.region = $2
		ORDER BY gsw.created_at DESC`

	rows, err := r.q.Query(ctx, query, summonerName, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get guilds watching summoner %s in %s: %w", summonerName, region, err)
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
func (r *SummonerWatchRepository) DeleteWatch(ctx context.Context, guildID int64, summonerName, region string) error {
	query := `
		DELETE FROM guild_summoner_watches 
		WHERE guild_id = $1 
		AND summoner_id = (
			SELECT id FROM summoners 
			WHERE summoner_name = $2 AND region = $3
		)`

	result, err := r.q.Exec(ctx, query, guildID, summonerName, region)
	if err != nil {
		return fmt.Errorf("failed to delete watch for guild %d, summoner %s in %s: %w", guildID, summonerName, region, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no watch found for guild %d, summoner %s in %s", guildID, summonerName, region)
	}

	return nil
}

// GetWatch retrieves a specific summoner watch for a guild
func (r *SummonerWatchRepository) GetWatch(ctx context.Context, guildID int64, summonerName, region string) (*models.SummonerWatchDetail, error) {
	query := `
		SELECT 
			gsw.id as watch_id,
			gsw.guild_id,
			gsw.created_at as watched_at,
			s.id as summoner_id,
			s.summoner_name,
			s.region,
			s.created_at,
			s.updated_at
		FROM guild_summoner_watches gsw
		JOIN summoners s ON gsw.summoner_id = s.id
		WHERE gsw.guild_id = $1 AND s.summoner_name = $2 AND s.region = $3`

	var watch models.SummonerWatchDetail
	err := r.q.QueryRow(ctx, query, guildID, summonerName, region).Scan(
		&watch.WatchID, &watch.GuildID, &watch.WatchedAt,
		&watch.SummonerID, &watch.SummonerName, &watch.Region,
		&watch.CreatedAt, &watch.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get watch for guild %d, summoner %s in %s: %w", guildID, summonerName, region, err)
	}

	return &watch, nil
}