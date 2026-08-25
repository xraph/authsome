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

	// Core migrations satisfy the agentauth group's DependsOn("authsome");
	// the agentauth group creates the client_id uniqueness index (and a
	// couple of query-support indexes) via mongomigrate.
	require.NoError(t, mongostore.New(db).Migrate(ctx, agentauth.MongoMigrations))
	store := agentauth.NewMongoStore(db)

	runStoreConformance(t, func(_ *testing.T) agentauth.Store { return store })
}
