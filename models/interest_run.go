package models

import (
	"time"
)

// InterestRun represents a daily interest computation run
type InterestRun struct {
	ID                       int64                  `db:"id"`
	RunDate                  time.Time              `db:"run_date"`
	TotalInterestDistributed int64                  `db:"total_interest_distributed"`
	UsersAffected            int                    `db:"users_affected"`
	ExecutionSummary         map[string]interface{} `db:"execution_summary"`
	CreatedAt                time.Time              `db:"created_at"`
}