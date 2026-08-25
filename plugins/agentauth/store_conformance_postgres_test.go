//go:build integration

package agentauth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"
	_ "github.com/xraph/grove/drivers/pgdriver/pgmigrate"

	"github.com/xraph/authsome/plugins/agentauth"
	pgstore "github.com/xraph/authsome/store/postgres"
)

// TestStoreConformance_Postgres runs the shared cross-backend suite against a
// real PostgreSQL instance via testcontainers. Requires Docker; run with
// `-tags integration`.
func TestStoreConformance_Postgres(t *testing.T) {
	ctx := context.Background()

	container, err := pgmodule.Run(ctx, "postgres:16-alpine",
		pgmodule.WithDatabase("agentauth_test"),
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

	// Core migrations satisfy the agentauth group's DependsOn("authsome");
	// the agentauth group creates the agents/agent_grants/agent_policies
	// tables with the full column set. One store, shared across every
	// subtest below — they isolate themselves with random ids — so this
	// only pays for a single container.
	require.NoError(t, pgstore.New(db).Migrate(ctx, agentauth.PostgresMigrations))
	store := agentauth.NewPostgresStore(db)

	runStoreConformance(t, func(_ *testing.T) agentauth.Store { return store })
}
