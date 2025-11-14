package testing

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"srmt-admin/internal/storage"
	postgresDriver "srmt-admin/internal/storage/driver/postgres"
	"srmt-admin/internal/storage/repo"
)

// TestDB wraps a test database with cleanup capabilities
type TestDB struct {
	Container *postgres.PostgresContainer
	DB        *sql.DB
	ConnStr   string
}

// SetupTestDB creates a PostgreSQL testcontainer and runs all migrations
// This provides a clean, isolated database for each test run
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container with postgres:16-alpine image
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute)),
	)
	require.NoError(t, err, "failed to start postgres container")

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	// Open database connection
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "failed to open database connection")
	require.NoError(t, db.Ping(), "failed to ping database")

	// Run migrations
	// Note: Path is relative to the test file location
	// From internal/storage/repo/testing -> migrations/postgres is ../../../../migrations/postgres
	migrationsPath := "file://" + filepath.Join("..", "..", "..", "..", "migrations", "postgres")

	m, err := migrate.New(migrationsPath, connStr)
	require.NoError(t, err, "failed to create migrator")
	defer m.Close()

	err = m.Up()
	require.NoError(t, err, "failed to apply migrations")

	return &TestDB{
		Container: pgContainer,
		DB:        db,
		ConnStr:   connStr,
	}
}

// Cleanup terminates the container and closes all connections
// Should be called with defer immediately after SetupTestDB
func (tdb *TestDB) Cleanup(t *testing.T) {
	t.Helper()

	if tdb.DB != nil {
		require.NoError(t, tdb.DB.Close(), "failed to close database connection")
	}
	if tdb.Container != nil {
		require.NoError(t, tdb.Container.Terminate(context.Background()), "failed to terminate container")
	}
}

// NewRepo creates a new Repo instance configured for testing
func (tdb *TestDB) NewRepo() *repo.Repo {
	return repo.New(&storage.Driver{
		DB:         tdb.DB,
		Translator: &postgresDriver.Translator{},
	})
}

// TruncateTable removes all data from a table (useful for cleanup between subtests)
func (tdb *TestDB) TruncateTable(t *testing.T, tableName string) {
	t.Helper()

	query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)
	_, err := tdb.DB.Exec(query)
	require.NoError(t, err, "failed to truncate table %s", tableName)
}

// TruncateAll removes all data from all tables (complete database reset)
func (tdb *TestDB) TruncateAll(t *testing.T) {
	t.Helper()

	tables := []string{
		"event_file_links",
		"events",
		"event_status",
		"event_type",
		"visits",
		"incidents",
		"shutdowns",
		"idle_water_discharges",
		"files",
		"categories",
		"users_roles",
		"users",
		"roles",
		"contacts",
		"departments",
		"positions",
		"organization_type_links",
		"organizations",
		"organization_types",
		"data",
		"level_volume",
		"indicator_height",
		"reservoirs",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)
		_, err := tdb.DB.Exec(query)
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}
