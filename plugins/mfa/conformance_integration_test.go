//go:build integration

package mfa_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	"github.com/xraph/grove/drivers/pgdriver"
	_ "github.com/xraph/grove/drivers/pgdriver/pgmigrate"

	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/mfa/mfatest"
	mongostore "github.com/xraph/authsome/store/mongo"
	pgstore "github.com/xraph/authsome/store/postgres"
)

// The container-backed backends join the suite only under `-tags integration`,
// matching how store/postgres and store/mongo gate their own conformance runs.
func init() {
	extraBackends = append(extraBackends,
		backend{"Postgres", postgresSetup},
		backend{"Mongo", mongoSetup},
	)
}

// postgresSetup starts one container for the whole backend. Each case still
// seeds its own tenants, so cases stay isolated without a container apiece.
func postgresSetup(t *testing.T) mfatest.Factory {
	t.Helper()
	ctx := context.Background()

	container, err := pgmodule.Run(ctx, "postgres:16-alpine",
		pgmodule.WithDatabase("authsome_test"),
		pgmodule.WithUsername("test"),
		pgmodule.WithPassword("test"),
		pgmodule.BasicWaitStrategies(),
		pgmodule.WithSQLDriver("pgx"),
	)
	require.NoError(t, err, "start postgres container")
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pgdb := pgdriver.New()
	require.NoError(t, pgdb.Open(ctx, connStr), "open grove pg connection")
	db, err := grove.Open(pgdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	core := pgstore.New(db)
	require.NoError(t, core.Migrate(ctx, mfa.PostgresMigrations), "run migrations")

	plugin := mfa.NewPostgresStore(db)
	return func(t *testing.T) mfatest.Fixture {
		t.Helper()
		return seedFixture(t, core, plugin)
	}
}

// mongoSetup reuses the CI mongo. There is no mongo migration group for this
// plugin: the collections are schemaless and the core Migrate call is what
// creates the indexes the fixtures rely on.
func mongoSetup(t *testing.T) mfatest.Factory {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo conformance run")
	}
	ctx := context.Background()

	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri), "open grove mongo connection")
	db, err := grove.Open(mdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() { _ = db.Close() })

	core := mongostore.New(db)
	require.NoError(t, core.Migrate(ctx), "run migrations")

	plugin := mfa.NewMongoStore(db)
	return func(t *testing.T) mfatest.Fixture {
		t.Helper()
		return seedFixture(t, core, plugin)
	}
}
