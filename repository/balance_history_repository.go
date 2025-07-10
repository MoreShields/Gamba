package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gambler/database"
	"gambler/models"
)

// BalanceHistoryRepository implements the BalanceHistoryRepository interface
type BalanceHistoryRepository struct {
	q queryable
}

// NewBalanceHistoryRepository creates a new balance history repository
func NewBalanceHistoryRepository(db *database.DB) *BalanceHistoryRepository {
	return &BalanceHistoryRepository{q: db.Pool}
}

// newBalanceHistoryRepositoryWithTx creates a new balance history repository with a transaction
func newBalanceHistoryRepositoryWithTx(tx queryable) *BalanceHistoryRepository {
	return &BalanceHistoryRepository{q: tx}
}

// Record creates a new balance history entry
func (r *BalanceHistoryRepository) Record(ctx context.Context, history *models.BalanceHistory) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(history.TransactionMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction metadata: %w", err)
	}
	
	query := `
		INSERT INTO balance_history 
		(discord_id, balance_before, balance_after, change_amount, transaction_type, transaction_metadata, related_id, related_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	
	err = r.q.QueryRow(ctx, query,
		history.DiscordID,
		history.BalanceBefore,
		history.BalanceAfter,
		history.ChangeAmount,
		history.TransactionType,
		metadataJSON,
		history.RelatedID,
		history.RelatedType,
	).Scan(&history.ID, &history.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to record balance history for user %d: %w", history.DiscordID, err)
	}
	
	return nil
}

// GetByUser returns balance history for a specific user
func (r *BalanceHistoryRepository) GetByUser(ctx context.Context, discordID int64, limit int) ([]*models.BalanceHistory, error) {
	query := `
		SELECT id, discord_id, balance_before, balance_after, change_amount, 
		       transaction_type, transaction_metadata, created_at
		FROM balance_history
		WHERE discord_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	
	rows, err := r.q.Query(ctx, query, discordID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance history for user %d: %w", discordID, err)
	}
	defer rows.Close()
	
	var histories []*models.BalanceHistory
	for rows.Next() {
		var history models.BalanceHistory
		var metadataJSON []byte
		
		err := rows.Scan(
			&history.ID,
			&history.DiscordID,
			&history.BalanceBefore,
			&history.BalanceAfter,
			&history.ChangeAmount,
			&history.TransactionType,
			&metadataJSON,
			&history.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance history: %w", err)
		}
		
		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &history.TransactionMetadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal transaction metadata: %w", err)
			}
		}
		
		histories = append(histories, &history)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate balance history: %w", err)
	}
	
	return histories, nil
}

// GetByDateRange returns balance history within a date range
func (r *BalanceHistoryRepository) GetByDateRange(ctx context.Context, discordID int64, from, to time.Time) ([]*models.BalanceHistory, error) {
	query := `
		SELECT id, discord_id, balance_before, balance_after, change_amount, 
		       transaction_type, transaction_metadata, created_at
		FROM balance_history
		WHERE discord_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at DESC
	`
	
	rows, err := r.q.Query(ctx, query, discordID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance history for user %d in date range: %w", discordID, err)
	}
	defer rows.Close()
	
	var histories []*models.BalanceHistory
	for rows.Next() {
		var history models.BalanceHistory
		var metadataJSON []byte
		
		err := rows.Scan(
			&history.ID,
			&history.DiscordID,
			&history.BalanceBefore,
			&history.BalanceAfter,
			&history.ChangeAmount,
			&history.TransactionType,
			&metadataJSON,
			&history.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance history: %w", err)
		}
		
		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &history.TransactionMetadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal transaction metadata: %w", err)
			}
		}
		
		histories = append(histories, &history)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate balance history: %w", err)
	}
	
	return histories, nil
}