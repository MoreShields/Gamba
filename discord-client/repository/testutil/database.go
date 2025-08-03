package testutil

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gambler/discord-client/database"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestDatabase represents a test database instance
type TestDatabase struct {
	Container *postgres.PostgresContainer
	DB        *database.DB
	URL       string
}

// SetupTestDatabase creates a new PostgreSQL test container and runs migrations
func SetupTestDatabase(t *testing.T) *TestDatabase {
	ctx := context.Background()

	// Generate unique labels for this test container
	testName := t.Name()
	timestamp := time.Now().Format("20060102-150405")
	labels := map[string]string{
		"test":      "gambler-repository",
		"test-name": testName,
		"timestamp": timestamp,
		"cleanup":   "auto",
	}

	// Generate a container name based on the test name
	// Replace any characters that aren't valid in container names
	sanitizedTestName := strings.ReplaceAll(testName, "/", "-")
	sanitizedTestName = strings.ReplaceAll(sanitizedTestName, " ", "_")
	containerName := fmt.Sprintf("gambler-test-pg-%s-%s", sanitizedTestName, timestamp)

	// Create PostgreSQL container with custom name and labels
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("gambler_test"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		postgres.BasicWaitStrategies(),
		testcontainers.WithLabels(labels),
		testcontainers.WithName(containerName),
	)
	require.NoError(t, err)

	// Log container information for debugging
	t.Logf("Created PostgreSQL container '%s' for test '%s'", containerName, testName)

	// Register cleanup immediately after successful container creation
	testDB := &TestDatabase{
		Container: postgresContainer,
	}

	// Use t.Cleanup for better test integration and guaranteed execution
	t.Cleanup(func() {
		testDB.robustCleanup(t)
	})

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run migrations first (before creating the connection)
	err = database.RunMigrationsWithURL(connStr)
	require.NoError(t, err)

	// Create database connection after migrations
	db, err := database.NewConnection(ctx, connStr)
	require.NoError(t, err)

	// Complete the test database setup
	testDB.DB = db
	testDB.URL = connStr

	return testDB
}

// Cleanup closes the database connection and terminates the container
// Deprecated: Use robustCleanup instead, which is automatically registered
func (td *TestDatabase) Cleanup(t *testing.T) {
	td.robustCleanup(t)
}

// robustCleanup provides robust container cleanup with panic recovery
func (td *TestDatabase) robustCleanup(t *testing.T) {
	// Recover from any panics during cleanup
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Panic during container cleanup (recovered): %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close database connection (non-critical if it fails)
	if td.DB != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Panic closing database connection (recovered): %v", r)
				}
			}()
			td.DB.Close()
		}()
	}

	// Terminate container with timeout
	if td.Container != nil {
		err := td.Container.Terminate(ctx)
		if err != nil {
			t.Logf("Warning: Failed to terminate test container: %v", err)
			// Don't fail the test on cleanup errors
		} else {
			t.Logf("Successfully cleaned up test container")
		}
	}
}
