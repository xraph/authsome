//go:build integration

package agentauth_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	_ "github.com/xraph/grove/drivers/mongodriver/mongomigrate"

	"github.com/xraph/authsome/plugins/agentauth"
	mongostore "github.com/xraph/authsome/store/mongo"
)

// TestStoreConformance_Mongo runs the shared cross-backend suite against a
// real MongoDB instance. Skips unless AUTHSOME_MONGO_URI is set (mirrors
// store/mongo's own integration tests, which use the same env var rather
// than testcontainers — there is no mongo testcontainers module in go.mod).
// Run with `-tags integration`.
func TestStoreConformance_Mongo(t *testing.T) {
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo integration test")
	}
	ctx := context.Background()

	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri), "open grove mongo connection")
	db, err := grove.Open(mdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() { _ = db.Close() })

	// Drop the database at setup, not just leave it for teardown: a prior
	// run against the same AUTHSOME_MONGO_URI (this test has no
	// testcontainers isolation, unlike Postgres) leaves the client_id
	// unique index populated with e.g. "dup-client-...", and the shared
	// store this test builds means every subtest's writes accumulate in
	// the same database across runs. Dropping at setup makes the test
	// re-runnable against a long-lived mongo instance; doing it at
	// teardown only would leave a failed run's leftovers for the next one.
	require.NoError(t, mongodriver.Unwrap(db).Database().Drop(ctx), "drop database before run")

	// Core migrations satisfy the agentauth group's DependsOn("authsome");
	// the agentauth group creates the client_id uniqueness index (and a
	// couple of query-support indexes) via mongomigrate.
	require.NoError(t, mongostore.New(db).Migrate(ctx, agentauth.MongoMigrations))
	store := agentauth.NewMongoStore(db)

	runStoreConformance(t, func(_ *testing.T) agentauth.Store { return store })
}
