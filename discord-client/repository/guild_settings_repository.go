package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/models"
	"github.com/jackc/pgx/v5"
)

// GuildSettingsRepository implements the GuildSettingsRepository interface
type GuildSettingsRepository struct {
	q Queryable
}

// NewGuildSettingsRepository creates a new guild settings repository
func NewGuildSettingsRepository(db *database.DB) *GuildSettingsRepository {
	return &GuildSettingsRepository{q: db.Pool}
}

// newGuildSettingsRepositoryWithTx creates a new guild settings repository with a transaction
func NewGuildSettingsRepositoryWithTx(tx Queryable) *GuildSettingsRepository {
	return &GuildSettingsRepository{q: tx}
}

// GetOrCreateGuildSettings retrieves guild settings or creates default ones if not found
func (r *GuildSettingsRepository) GetOrCreateGuildSettings(ctx context.Context, guildID int64) (*models.GuildSettings, error) {
	// First try to get existing settings
	query := `
		SELECT guild_id, primary_channel_id, lol_channel_id, high_roller_role_id
		FROM guild_settings
		WHERE guild_id = $1
	`

	var settings models.GuildSettings
	err := r.q.QueryRow(ctx, query, guildID).Scan(
		&settings.GuildID,
		&settings.PrimaryChannelID,
		&settings.LolChannelID,
		&settings.HighRollerRoleID,
	)

	if err == nil {
		return &settings, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get guild settings for guild %d: %w", guildID, err)
	}

	// If not found, create default settings
	insertQuery := `
		INSERT INTO guild_settings (guild_id, primary_channel_id, lol_channel_id, high_roller_role_id)
		VALUES ($1, NULL, NULL, NULL)
		RETURNING guild_id, primary_channel_id, lol_channel_id, high_roller_role_id
	`

	err = r.q.QueryRow(ctx, insertQuery, guildID).Scan(
		&settings.GuildID,
		&settings.PrimaryChannelID,
		&settings.LolChannelID,
		&settings.HighRollerRoleID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create guild settings for guild %d: %w", guildID, err)
	}

	return &settings, nil
}

// UpdateGuildSettings updates guild settings
func (r *GuildSettingsRepository) UpdateGuildSettings(ctx context.Context, settings *models.GuildSettings) error {
	query := `
		UPDATE guild_settings
		SET primary_channel_id = $2,
		    lol_channel_id = $3,
		    high_roller_role_id = $4
		WHERE guild_id = $1
	`

	result, err := r.q.Exec(ctx, query,
		settings.GuildID,
		settings.PrimaryChannelID,
		settings.LolChannelID,
		settings.HighRollerRoleID,
	)

	if err != nil {
		return fmt.Errorf("failed to update guild settings for guild %d: %w", settings.GuildID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("guild settings for guild %d not found", settings.GuildID)
	}

	return nil
}
