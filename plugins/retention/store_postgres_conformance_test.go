//go:build integration

package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"
	_ "github.com/xraph/grove/drivers/pgdriver/pgmigrate"

	pgstore "github.com/xraph/authsome/store/postgres"
)

// TestPostgresStoreConformance runs the shared cross-backend suite against a
// real PostgreSQL instance via testcontainers. Requires Docker; run with
// `-tags integration`.
func TestPostgresStoreConformance(t *testing.T) {
	ctx := context.Background()

	container, err := pgmodule.Run(ctx, "postgres:16-alpine",
		pgmodule.WithDatabase("retention_test"),
		pgmodule.WithUsername("test"),
		pgmodule.WithPassword("test"),
		pgmodule.BasicWaitStrategies(),
		pgmodule.WithSQLDriver("pgx"),
	)
	require.NoError(t, err, "start postgres container")
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx), "terminate container") })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "get connection string")

	pgdb := pgdriver.New()
	require.NoError(t, pgdb.Open(ctx, connStr), "open grove pg connection")
	db, err := grove.Open(pgdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() { _ = db.Close() })

	// Core migrations satisfy the retention group's DependsOn("authsome");
	// the retention group then creates the outbox and contact-ref tables.
	// One container for the whole suite, but ClaimDue is deliberately global
	// (no app_id filter -- see claimSQL), so unlike stores whose methods are
	// all scoped by an id argument, leftover rows from an earlier subtest
	// are visible to a later subtest's unscoped claim. Truncating both
	// tables before every subtest gives each one the same clean slate the
	// sqlite factory gets for free by opening a brand new database file.
	require.NoError(t, pgstore.New(db).Migrate(ctx, PostgresMigrations))
	store := NewPostgresStore(db)

	runStoreConformance(t, func(t *testing.T) Store {
		t.Helper()
		_, err := pgdb.Exec(ctx, "TRUNCATE TABLE authsome_retention_outbox, authsome_retention_contact_ref")
		require.NoError(t, err, "truncate retention tables between subtests")
		return store
	})
}
