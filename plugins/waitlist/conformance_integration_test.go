//go:build integration

package waitlist_test

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

	"github.com/xraph/authsome/plugins/waitlist"
	"github.com/xraph/authsome/plugins/waitlist/waitlisttest"
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
func postgresSetup(t *testing.T) waitlisttest.Factory {
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
	require.NoError(t, core.Migrate(ctx, waitlist.PostgresMigrations), "run migrations")

	plugin := waitlist.NewPostgresStore(db)
	const enforcesUniqueEmail = true
	return func(t *testing.T) waitlisttest.Fixture {
		t.Helper()
		return seedFixture(t, core, plugin, enforcesUniqueEmail)
	}
}

// mongoSetup reuses the CI mongo. There is no mongo migration group for this
// plugin: the collections are schemaless and the core Migrate call is what
// creates the indexes the fixtures rely on.
func mongoSetup(t *testing.T) waitlisttest.Factory {
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
	// The plugin's mongo group is what declares its collections and indexes,
	// so it has to run here too. Skipping it makes the fixture disagree with
	// production and turns a missing index into a passing test.
	require.NoError(t, core.Migrate(ctx, waitlist.MongoMigrations), "run migrations")

	plugin := waitlist.NewMongoStore(db)
	// waitlist.MongoMigrations exists and runs above, but its only migration
	// is a no-op, so nothing declares a unique index on (app_id, email) the
	// way the SQL migrations do.
	const enforcesUniqueEmail = false
	return func(t *testing.T) waitlisttest.Fixture {
		t.Helper()
		return seedFixture(t, core, plugin, enforcesUniqueEmail)
	}
}
