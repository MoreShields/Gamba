package repository

import (
	"context"
	"errors"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"github.com/jackc/pgx/v5"
)

type highRollerPurchaseRepository struct {
	q       Queryable
	guildID int64
}

// NewHighRollerPurchaseRepository creates a new high roller purchase repository
func NewHighRollerPurchaseRepository(db *database.DB) interfaces.HighRollerPurchaseRepository {
	return &highRollerPurchaseRepository{
		q: db.Pool,
	}
}

// NewHighRollerPurchaseRepositoryScoped creates a new high roller purchase repository with transaction
func NewHighRollerPurchaseRepositoryScoped(tx Queryable, guildID int64) interfaces.HighRollerPurchaseRepository {
	return &highRollerPurchaseRepository{
		q:       tx,
		guildID: guildID,
	}
}

// GetLatestPurchase retrieves the most recent high roller purchase for a guild
func (r *highRollerPurchaseRepository) GetLatestPurchase(ctx context.Context, guildID int64) (*entities.HighRollerPurchase, error) {
	query := `
		SELECT id, guild_id, discord_id, purchase_price, purchased_at
		FROM high_roller_purchases
		WHERE guild_id = $1
		ORDER BY purchased_at DESC
		LIMIT 1
	`

	var purchase entities.HighRollerPurchase
	err := r.q.QueryRow(ctx, query, guildID).Scan(
		&purchase.ID,
		&purchase.GuildID,
		&purchase.DiscordID,
		&purchase.PurchasePrice,
		&purchase.PurchasedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // No purchase found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest purchase: %w", err)
	}

	return &purchase, nil
}

// CreatePurchase creates a new high roller purchase record
func (r *highRollerPurchaseRepository) CreatePurchase(ctx context.Context, purchase *entities.HighRollerPurchase) error {
	if purchase.GuildID != r.guildID {
		return errors.New("guild ID mismatch")
	}

	query := `
		INSERT INTO high_roller_purchases (guild_id, discord_id, purchase_price, purchased_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.q.QueryRow(
		ctx,
		query,
		purchase.GuildID,
		purchase.DiscordID,
		purchase.PurchasePrice,
		purchase.PurchasedAt,
	).Scan(&purchase.ID)

	if err != nil {
		return fmt.Errorf("failed to create purchase: %w", err)
	}

	return nil
}

// GetPurchaseHistory retrieves the purchase history for a guild
func (r *highRollerPurchaseRepository) GetPurchaseHistory(ctx context.Context, guildID int64, limit int) ([]*entities.HighRollerPurchase, error) {
	query := `
		SELECT id, guild_id, discord_id, purchase_price, purchased_at
		FROM high_roller_purchases
		WHERE guild_id = $1
		ORDER BY purchased_at DESC
		LIMIT $2
	`

	rows, err := r.q.Query(ctx, query, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase history: %w", err)
	}
	defer rows.Close()

	var purchases []*entities.HighRollerPurchase
	for rows.Next() {
		var purchase entities.HighRollerPurchase
		err := rows.Scan(
			&purchase.ID,
			&purchase.GuildID,
			&purchase.DiscordID,
			&purchase.PurchasePrice,
			&purchase.PurchasedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}
		purchases = append(purchases, &purchase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating purchase rows: %w", err)
	}

	return purchases, nil
}