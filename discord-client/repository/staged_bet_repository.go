package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/domain/entities"

	"github.com/jackc/pgx/v5"
)

// StagedBetRepository implements the StagedBetRepository interface
type StagedBetRepository struct {
	q       Queryable
	guildID int64
}

// NewStagedBetRepositoryScoped creates a new staged bet repository with a transaction and guild scope
func NewStagedBetRepositoryScoped(tx Queryable, guildID int64) *StagedBetRepository {
	return &StagedBetRepository{
		q:       tx,
		guildID: guildID,
	}
}

// Upsert creates or updates a staged bet for a user/summoner/guild combination
func (r *StagedBetRepository) Upsert(ctx context.Context, stagedBet *entities.StagedBet) error {
	query := `
		INSERT INTO staged_bets (guild_id, discord_id, summoner_id, amount)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (guild_id, discord_id, summoner_id)
		DO UPDATE SET amount = $4, created_at = NOW()
		RETURNING id, created_at`

	err := r.q.QueryRow(ctx, query,
		stagedBet.GuildID, stagedBet.DiscordID, stagedBet.SummonerID, stagedBet.Amount,
	).Scan(&stagedBet.ID, &stagedBet.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert staged bet for user %d, summoner %d in guild %d: %w",
			stagedBet.DiscordID, stagedBet.SummonerID, stagedBet.GuildID, err)
	}

	return nil
}

// GetByGuildAndSummoner returns all staged bets for a guild+summoner pair
func (r *StagedBetRepository) GetByGuildAndSummoner(ctx context.Context, guildID int64, summonerID int64) ([]*entities.StagedBet, error) {
	query := `
		SELECT id, guild_id, discord_id, summoner_id, amount, created_at
		FROM staged_bets
		WHERE guild_id = $1 AND summoner_id = $2`

	rows, err := r.q.Query(ctx, query, guildID, summonerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get staged bets for guild %d, summoner %d: %w", guildID, summonerID, err)
	}
	defer rows.Close()

	var bets []*entities.StagedBet
	for rows.Next() {
		var bet entities.StagedBet
		err := rows.Scan(&bet.ID, &bet.GuildID, &bet.DiscordID, &bet.SummonerID, &bet.Amount, &bet.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staged bet: %w", err)
		}
		bets = append(bets, &bet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating staged bets: %w", err)
	}

	return bets, nil
}

// Delete removes a staged bet by ID
func (r *StagedBetRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM staged_bets WHERE id = $1`
	result, err := r.q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete staged bet %d: %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("staged bet %d not found", id)
	}

	return nil
}

// GetByUserAndSummoner returns a staged bet for a specific user/summoner/guild if one exists
func (r *StagedBetRepository) GetByUserAndSummoner(ctx context.Context, guildID, discordID, summonerID int64) (*entities.StagedBet, error) {
	query := `
		SELECT id, guild_id, discord_id, summoner_id, amount, created_at
		FROM staged_bets
		WHERE guild_id = $1 AND discord_id = $2 AND summoner_id = $3`

	var bet entities.StagedBet
	err := r.q.QueryRow(ctx, query, guildID, discordID, summonerID).Scan(
		&bet.ID, &bet.GuildID, &bet.DiscordID, &bet.SummonerID, &bet.Amount, &bet.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get staged bet for user %d, summoner %d in guild %d: %w",
			discordID, summonerID, guildID, err)
	}

	return &bet, nil
}
