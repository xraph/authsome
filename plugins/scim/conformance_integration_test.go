//go:build integration

package scim_test

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

	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/plugins/scim/scimtest"
	mongostore "github.com/xraph/authsome/store/mongo"
	pgstore "github.com/xraph/authsome/store/postgres"
)

func init() {
	extraBackends = append(extraBackends,
		backend{"Postgres", postgresSetup},
		backend{"Mongo", mongoSetup},
	)
}

func postgresSetup(t *testing.T) scimtest.Factory {
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
	require.NoError(t, pgdb.Open(ctx, connStr))
	db, err := grove.Open(pgdb)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, pgstore.New(db).Migrate(ctx, scim.PostgresMigrations))
	return fixtureFor(scim.NewPostgresStore(db))
}

func mongoSetup(t *testing.T) scimtest.Factory {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo conformance run")
	}
	ctx := context.Background()
	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri))
	db, err := grove.Open(mdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, mongostore.New(db).Migrate(ctx))
	return fixtureFor(scim.NewMongoStore(db))
}
