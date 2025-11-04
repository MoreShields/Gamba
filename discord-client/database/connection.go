package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB represents a database connection pool
type DB struct {
	*pgxpool.Pool
}

// NewConnection creates a new database connection pool
func NewConnection(ctx context.Context, databaseURL string) (*DB, error) {
	// Parse config to set timezone
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set timezone to UTC for all connections
	config.ConnConfig.RuntimeParams["timezone"] = "UTC"

	// Create pool with UTC timezone
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
}
