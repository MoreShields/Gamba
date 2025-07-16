package database

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// getMigrationDatabaseURL constructs the database URL for migrations
// This doesn't use the full config to avoid requiring DISCORD_TOKEN for migrations
func getMigrationDatabaseURL() string {
	baseURL := os.Getenv("DATABASE_URL")
	databaseName := os.Getenv("DATABASE_NAME")

	return ConstructDatabaseURL(baseURL, databaseName)
}

// MigrateUp runs all pending migrations
func MigrateUp() error {
	databaseURL := getMigrationDatabaseURL()
	log.Printf("Migration connecting to: %s", databaseURL)

	m, err := getMigrate(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No new migrations to apply")
	} else {
		version, _, _ := m.Version()
		log.Printf("Successfully migrated to version %d", version)
	}

	return nil
}

// MigrateDown rolls back the specified number of migrations
func MigrateDown(stepsStr string) error {
	databaseURL := getMigrationDatabaseURL()

	steps, err := strconv.Atoi(stepsStr)
	if err != nil {
		return fmt.Errorf("invalid steps value: %w", err)
	}

	m, err := getMigrate(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No migrations to rollback")
	} else {
		version, _, _ := m.Version()
		log.Printf("Successfully rolled back to version %d", version)
	}

	return nil
}

// MigrateStatus shows the current migration status
func MigrateStatus() error {
	databaseURL := getMigrationDatabaseURL()

	m, err := getMigrate(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		log.Println("No migrations have been applied yet")
		return nil
	}

	status := "clean"
	if dirty {
		status = "dirty"
	}

	log.Printf("Current migration version: %d (status: %s)", version, status)
	return nil
}

// RunMigrationsWithURL runs all pending migrations with a custom database URL
// This is useful for test environments where the URL is dynamically generated
func RunMigrationsWithURL(databaseURL string) error {
	m, err := getMigrate(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// getMigrate creates a new migrate instance
func getMigrate(databaseURL string) (*migrate.Migrate, error) {
	// Parse connection config
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Create stdlib connection
	db := stdlib.OpenDB(*config.ConnConfig)

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create source driver from embedded files
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}
