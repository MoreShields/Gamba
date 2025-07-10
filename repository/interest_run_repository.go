package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gambler/database"
	"gambler/models"
	"github.com/jackc/pgx/v5"
)

// InterestRunRepository implements the InterestRunRepository interface
type InterestRunRepository struct {
	db *database.DB
}

// NewInterestRunRepository creates a new interest run repository
func NewInterestRunRepository(db *database.DB) *InterestRunRepository {
	return &InterestRunRepository{db: db}
}

// GetByDate checks if an interest run exists for a specific date
func (r *InterestRunRepository) GetByDate(ctx context.Context, date time.Time) (*models.InterestRun, error) {
	// Normalize date to start of day
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	
	query := `
		SELECT id, run_date, total_interest_distributed, users_affected, 
		       execution_summary, created_at
		FROM interest_runs
		WHERE run_date = $1
	`
	
	var run models.InterestRun
	var summaryJSON []byte
	
	err := r.db.QueryRow(ctx, query, dateOnly).Scan(
		&run.ID,
		&run.RunDate,
		&run.TotalInterestDistributed,
		&run.UsersAffected,
		&summaryJSON,
		&run.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get interest run for date %s: %w", dateOnly.Format("2006-01-02"), err)
	}
	
	// Unmarshal execution summary
	if len(summaryJSON) > 0 {
		if err := json.Unmarshal(summaryJSON, &run.ExecutionSummary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal execution summary: %w", err)
		}
	}
	
	return &run, nil
}

// Create creates a new interest run record
func (r *InterestRunRepository) Create(ctx context.Context, run *models.InterestRun) error {
	// Normalize date to start of day
	run.RunDate = time.Date(run.RunDate.Year(), run.RunDate.Month(), run.RunDate.Day(), 
		0, 0, 0, 0, run.RunDate.Location())
	
	// Convert summary to JSON
	summaryJSON, err := json.Marshal(run.ExecutionSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal execution summary: %w", err)
	}
	
	query := `
		INSERT INTO interest_runs 
		(run_date, total_interest_distributed, users_affected, execution_summary)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	
	err = r.db.QueryRow(ctx, query,
		run.RunDate,
		run.TotalInterestDistributed,
		run.UsersAffected,
		summaryJSON,
	).Scan(&run.ID, &run.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create interest run for date %s: %w", 
			run.RunDate.Format("2006-01-02"), err)
	}
	
	return nil
}

// GetLatest returns the most recent interest run
func (r *InterestRunRepository) GetLatest(ctx context.Context) (*models.InterestRun, error) {
	query := `
		SELECT id, run_date, total_interest_distributed, users_affected, 
		       execution_summary, created_at
		FROM interest_runs
		ORDER BY run_date DESC
		LIMIT 1
	`
	
	var run models.InterestRun
	var summaryJSON []byte
	
	err := r.db.QueryRow(ctx, query).Scan(
		&run.ID,
		&run.RunDate,
		&run.TotalInterestDistributed,
		&run.UsersAffected,
		&summaryJSON,
		&run.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest interest run: %w", err)
	}
	
	// Unmarshal execution summary
	if len(summaryJSON) > 0 {
		if err := json.Unmarshal(summaryJSON, &run.ExecutionSummary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal execution summary: %w", err)
		}
	}
	
	return &run, nil
}