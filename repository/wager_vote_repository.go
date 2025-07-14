package repository

import (
	"context"
	"fmt"

	"gambler/database"
	"gambler/models"
	"github.com/jackc/pgx/v5"
)

// WagerVoteRepository implements wager vote data access
type WagerVoteRepository struct {
	q       queryable
	guildID int64
}

// NewWagerVoteRepository creates a new wager vote repository
func NewWagerVoteRepository(db *database.DB) *WagerVoteRepository {
	return &WagerVoteRepository{q: db.Pool}
}

// newWagerVoteRepositoryWithTx creates a new wager vote repository with a transaction
func newWagerVoteRepositoryWithTx(tx queryable) *WagerVoteRepository {
	return &WagerVoteRepository{q: tx}
}

// newWagerVoteRepository creates a new wager vote repository with a transaction and guild scope
func newWagerVoteRepository(tx queryable, guildID int64) *WagerVoteRepository {
	return &WagerVoteRepository{
		q:       tx,
		guildID: guildID,
	}
}

// CreateOrUpdate creates a new vote or updates an existing one
func (r *WagerVoteRepository) CreateOrUpdate(ctx context.Context, vote *models.WagerVote) error {
	query := `
		INSERT INTO wager_votes (wager_id, guild_id, voter_discord_id, vote_for_discord_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (wager_id, voter_discord_id) 
		DO UPDATE SET 
			vote_for_discord_id = EXCLUDED.vote_for_discord_id,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`
	
	err := r.q.QueryRow(ctx, query,
		vote.WagerID,
		r.guildID, // Use repository's guild scope
		vote.VoterDiscordID,
		vote.VoteForDiscordID,
	).Scan(&vote.ID, &vote.CreatedAt, &vote.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create or update vote: %w", err)
	}
	
	return nil
}

// GetByWager returns all votes for a specific wager
func (r *WagerVoteRepository) GetByWager(ctx context.Context, wagerID int64) ([]*models.WagerVote, error) {
	query := `
		SELECT id, wager_id, guild_id, voter_discord_id, vote_for_discord_id, created_at, updated_at
		FROM wager_votes
		WHERE wager_id = $1
		ORDER BY created_at ASC
	`
	
	rows, err := r.q.Query(ctx, query, wagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes for wager %d: %w", wagerID, err)
	}
	defer rows.Close()
	
	var votes []*models.WagerVote
	for rows.Next() {
		var vote models.WagerVote
		err := rows.Scan(
			&vote.ID,
			&vote.WagerID,
			&vote.GuildID,
			&vote.VoterDiscordID,
			&vote.VoteForDiscordID,
			&vote.CreatedAt,
			&vote.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vote: %w", err)
		}
		votes = append(votes, &vote)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate votes: %w", err)
	}
	
	return votes, nil
}

// GetVoteCounts returns the participant vote counts and status for a wager
func (r *WagerVoteRepository) GetVoteCounts(ctx context.Context, wagerID int64) (*models.VoteCount, error) {
	// First, get the wager to know who the participants are
	wagerQuery := `
		SELECT proposer_discord_id, target_discord_id
		FROM wagers
		WHERE id = $1
	`
	
	var proposerID, targetID int64
	err := r.q.QueryRow(ctx, wagerQuery, wagerID).Scan(&proposerID, &targetID)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("wager %d not found", wagerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wager: %w", err)
	}
	
	// Now count the participant votes and check their voting status
	countQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE vote_for_discord_id = $2) as proposer_votes,
			COUNT(*) FILTER (WHERE vote_for_discord_id = $3) as target_votes,
			COUNT(*) as total_votes,
			COUNT(*) FILTER (WHERE voter_discord_id = $2) > 0 as proposer_voted,
			COUNT(*) FILTER (WHERE voter_discord_id = $3) > 0 as target_voted
		FROM wager_votes
		WHERE wager_id = $1
	`
	
	var voteCounts models.VoteCount
	err = r.q.QueryRow(ctx, countQuery, wagerID, proposerID, targetID).Scan(
		&voteCounts.ProposerVotes,
		&voteCounts.TargetVotes,
		&voteCounts.TotalVotes,
		&voteCounts.ProposerVoted,
		&voteCounts.TargetVoted,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get vote counts for wager %d: %w", wagerID, err)
	}
	
	return &voteCounts, nil
}

// GetByVoter returns a vote by a specific voter for a wager
func (r *WagerVoteRepository) GetByVoter(ctx context.Context, wagerID int64, voterDiscordID int64) (*models.WagerVote, error) {
	query := `
		SELECT id, wager_id, guild_id, voter_discord_id, vote_for_discord_id, created_at, updated_at
		FROM wager_votes
		WHERE wager_id = $1 AND voter_discord_id = $2
	`
	
	var vote models.WagerVote
	err := r.q.QueryRow(ctx, query, wagerID, voterDiscordID).Scan(
		&vote.ID,
		&vote.WagerID,
		&vote.GuildID,
		&vote.VoterDiscordID,
		&vote.VoteForDiscordID,
		&vote.CreatedAt,
		&vote.UpdatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vote: %w", err)
	}
	
	return &vote, nil
}

// DeleteByWager deletes all votes for a wager (used when cancelling)
func (r *WagerVoteRepository) DeleteByWager(ctx context.Context, wagerID int64) error {
	query := `DELETE FROM wager_votes WHERE wager_id = $1`
	
	_, err := r.q.Exec(ctx, query, wagerID)
	if err != nil {
		return fmt.Errorf("failed to delete votes for wager %d: %w", wagerID, err)
	}
	
	return nil
}