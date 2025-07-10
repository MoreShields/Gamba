package repository

import (
	"context"
	"testing"
	"time"

	"gambler/models"
	"gambler/repository/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterestRunRepository_GetByDate(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewInterestRunRepository(testDB.DB)
	ctx := context.Background()

	testDate := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)

	t.Run("no run found", func(t *testing.T) {
		run, err := repo.GetByDate(ctx, testDate)
		require.NoError(t, err)
		assert.Nil(t, run)
	})

	t.Run("run found", func(t *testing.T) {
		// Create a test run
		originalRun := testutil.CreateTestInterestRun(testDate)
		err := repo.Create(ctx, originalRun)
		require.NoError(t, err)

		// Retrieve by date
		run, err := repo.GetByDate(ctx, testDate)
		require.NoError(t, err)
		require.NotNil(t, run)

		assert.Equal(t, originalRun.TotalInterestDistributed, run.TotalInterestDistributed)
		assert.Equal(t, originalRun.UsersAffected, run.UsersAffected)
		assert.NotNil(t, run.ExecutionSummary)

		// Date should be normalized to start of day
		expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, testDate.Location())
		assert.Equal(t, expectedDate, run.RunDate)
	})

	t.Run("date normalization", func(t *testing.T) {
		// Create run with a specific time
		runDate := time.Date(2024, 2, 20, 14, 25, 30, 500, time.UTC)
		run := testutil.CreateTestInterestRun(runDate)
		err := repo.Create(ctx, run)
		require.NoError(t, err)

		// Query with different time on same date
		queryDate := time.Date(2024, 2, 20, 9, 45, 15, 123, time.UTC)
		retrievedRun, err := repo.GetByDate(ctx, queryDate)
		require.NoError(t, err)
		require.NotNil(t, retrievedRun)

		// Should find the run despite different times
		assert.Equal(t, run.TotalInterestDistributed, retrievedRun.TotalInterestDistributed)
	})
}

func TestInterestRunRepository_Create(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewInterestRunRepository(testDB.DB)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		testDate := time.Date(2024, 3, 10, 15, 30, 0, 0, time.UTC)
		run := testutil.CreateTestInterestRunWithDetails(testDate, 25000, 50)
		run.ExecutionSummary = map[string]interface{}{
			"total_users_checked": 75,
			"users_with_balance":  50,
			"average_interest":    500,
			"max_interest":        2000,
			"min_interest":        50,
			"execution_time_ms":   1250,
		}

		err := repo.Create(ctx, run)
		require.NoError(t, err)
		assert.NotZero(t, run.ID)
		assert.False(t, run.CreatedAt.IsZero())

		// Date should be normalized to start of day
		expectedDate := time.Date(2024, 3, 10, 0, 0, 0, 0, testDate.Location())
		assert.Equal(t, expectedDate, run.RunDate)
	})

	t.Run("duplicate date constraint", func(t *testing.T) {
		testDate := time.Date(2024, 4, 5, 10, 0, 0, 0, time.UTC)
		
		// Create first run
		run1 := testutil.CreateTestInterestRun(testDate)
		err := repo.Create(ctx, run1)
		require.NoError(t, err)

		// Try to create second run for same date
		run2 := testutil.CreateTestInterestRun(testDate)
		err = repo.Create(ctx, run2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unique") // PostgreSQL unique constraint error
	})

	t.Run("empty execution summary", func(t *testing.T) {
		testDate := time.Date(2024, 5, 1, 8, 0, 0, 0, time.UTC)
		run := testutil.CreateTestInterestRun(testDate)
		run.ExecutionSummary = nil

		err := repo.Create(ctx, run)
		require.NoError(t, err)
		assert.NotZero(t, run.ID)
	})

	t.Run("complex execution summary", func(t *testing.T) {
		testDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		run := testutil.CreateTestInterestRun(testDate)
		run.ExecutionSummary = map[string]interface{}{
			"users": []map[string]interface{}{
				{
					"discord_id": 123456,
					"username":   "user1",
					"interest":   500,
				},
				{
					"discord_id": 789012,
					"username":   "user2", 
					"interest":   750,
				},
			},
			"statistics": map[string]interface{}{
				"total_balance_before": 1000000,
				"total_balance_after":  1001250,
				"interest_rate":        0.005,
			},
			"execution_metadata": map[string]interface{}{
				"server_id":      "worker-01",
				"execution_time": "2024-06-15T12:00:00Z",
				"version":        "1.0.0",
			},
		}

		err := repo.Create(ctx, run)
		require.NoError(t, err)
		assert.NotZero(t, run.ID)
	})
}

func TestInterestRunRepository_GetLatest(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)

	repo := NewInterestRunRepository(testDB.DB)
	ctx := context.Background()

	t.Run("no runs exist", func(t *testing.T) {
		run, err := repo.GetLatest(ctx)
		require.NoError(t, err)
		assert.Nil(t, run)
	})

	t.Run("single run", func(t *testing.T) {
		testDate := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
		run := testutil.CreateTestInterestRun(testDate)
		err := repo.Create(ctx, run)
		require.NoError(t, err)

		latest, err := repo.GetLatest(ctx)
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, run.ID, latest.ID)
	})

	t.Run("multiple runs returns latest", func(t *testing.T) {
		// Create runs in non-chronological order
		dates := []time.Time{
			time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 8, 10, 0, 0, 0, 0, time.UTC), // Latest
			time.Date(2024, 8, 3, 0, 0, 0, 0, time.UTC),
		}

		var latestRun *models.InterestRun
		latestDate := time.Time{}

		for i, date := range dates {
			run := testutil.CreateTestInterestRunWithDetails(date, int64(1000*(i+1)), i+5)
			err := repo.Create(ctx, run)
			require.NoError(t, err)

			if date.After(latestDate) {
				latestDate = date
				latestRun = run
			}
		}

		// Get latest run
		latest, err := repo.GetLatest(ctx)
		require.NoError(t, err)
		require.NotNil(t, latest)
		
		// Should be the run with the latest date (2024-8-10)
		assert.Equal(t, latestRun.TotalInterestDistributed, latest.TotalInterestDistributed)
		assert.Equal(t, latestRun.UsersAffected, latest.UsersAffected)
		assert.Equal(t, latestDate.Format("2006-01-02"), latest.RunDate.Format("2006-01-02"))
	})

	t.Run("metadata preservation in latest", func(t *testing.T) {
		testDate := time.Date(2024, 9, 20, 0, 0, 0, 0, time.UTC)
		originalSummary := map[string]interface{}{
			"test_mode":        true,
			"interest_rate":    0.005,
			"total_processed":  100,
			"execution_stats": map[string]interface{}{
				"duration_ms":    1500,
				"memory_usage":   "45MB",
				"cpu_usage":      "12%",
			},
		}

		run := testutil.CreateTestInterestRun(testDate)
		run.ExecutionSummary = originalSummary
		err := repo.Create(ctx, run)
		require.NoError(t, err)

		latest, err := repo.GetLatest(ctx)
		require.NoError(t, err)
		require.NotNil(t, latest)

		// Verify metadata was preserved
		assert.Equal(t, true, latest.ExecutionSummary["test_mode"])
		assert.Equal(t, 0.005, latest.ExecutionSummary["interest_rate"])
		assert.Equal(t, float64(100), latest.ExecutionSummary["total_processed"])
		
		execStats := latest.ExecutionSummary["execution_stats"].(map[string]interface{})
		assert.Equal(t, float64(1500), execStats["duration_ms"])
		assert.Equal(t, "45MB", execStats["memory_usage"])
		assert.Equal(t, "12%", execStats["cpu_usage"])
	})
}