package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"rockets-backend/pkg"
	"testing"

	_ "github.com/lib/pq"
)

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use test database configuration
	host := pkg.GetEnv("TEST_DB_HOST", "localhost")
	port := pkg.GetEnv("TEST_DB_PORT", "5432")
	user := pkg.GetEnv("TEST_DB_USER", "postgres")
	password := pkg.GetEnv("TEST_DB_PASSWORD", "postgres")
	dbname := pkg.GetEnv("TEST_DB_NAME", "rockets_test")
	sslmode := pkg.GetEnv("TEST_DB_SSLMODE", "disable")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err = db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Initialize test schema
	setupTestSchema(t, db)

	return db
}

// CleanupTestDB cleans up test data
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clean up test data in reverse dependency order
	tables := []string{"rocket_events", "rockets"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Failed to clean up table %s: %v", table, err)
		}
	}
}

// setupTestSchema creates the test database schema
func setupTestSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	schema := `
	CREATE TABLE IF NOT EXISTS rockets (
		id UUID PRIMARY KEY,
		type VARCHAR(100) NOT NULL,
		current_speed INTEGER NOT NULL DEFAULT 0,
		mission VARCHAR(255) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		explosion_reason VARCHAR(255) NULL,
		launch_time TIMESTAMP NOT NULL,
		last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_message_number INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS rocket_events (
		id SERIAL PRIMARY KEY,
		channel UUID NOT NULL,
		message_number INTEGER NOT NULL,
		message_type VARCHAR(50) NOT NULL,
		message_data JSONB NOT NULL,
		received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		processed_at TIMESTAMP NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		error_message TEXT NULL,
		UNIQUE(channel, message_number)
	);

	CREATE INDEX IF NOT EXISTS idx_rockets_status ON rockets(status);
	CREATE INDEX IF NOT EXISTS idx_rockets_last_updated ON rockets(last_updated);
	CREATE INDEX IF NOT EXISTS idx_rockets_type ON rockets(type);
	CREATE INDEX IF NOT EXISTS idx_rocket_events_status ON rocket_events(status);
	CREATE INDEX IF NOT EXISTS idx_rocket_events_channel ON rocket_events(channel);
	CREATE INDEX IF NOT EXISTS idx_rocket_events_received_at ON rocket_events(received_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
}

// SkipIfNoTestDB skips the test if test database is not available
func SkipIfNoTestDB(t *testing.T) {
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database tests (SKIP_DB_TESTS=true)")
	}
}
