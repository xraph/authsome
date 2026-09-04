//go:build integration

package retention

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	_ "github.com/xraph/grove/drivers/mongodriver/mongomigrate"

	"github.com/xraph/authsome/id"
	mongostore "github.com/xraph/authsome/store/mongo"
)

// TestMongoStoreConformance runs the shared cross-backend suite against a
// real MongoDB instance. Gated the way the rest of the repo gates mongo
// work: a build tag plus AUTHSOME_MONGO_URI. It stays out of the default
// `go test ./...` because it needs a live server.
//
//	AUTHSOME_MONGO_URI=mongodb://localhost:27017 go test -tags integration \
//	    ./plugins/retention/ -run TestMongoStoreConformance
func TestMongoStoreConformance(t *testing.T) {
	s, mdb := newMongoConformanceStore(t)
	ctx := context.Background()

	// One database for the whole suite, but ClaimDue is deliberately global
	// (no app_id filter), so unlike methods that are all scoped by an id
	// argument, leftover documents from an earlier subtest are visible to a
	// later subtest's unscoped claim. Clearing both collections before every
	// subtest gives each one the same clean slate the sqlite factory gets
	// for free by opening a brand new database file.
	runStoreConformance(t, func(t *testing.T) Store {
		t.Helper()
		_, err := mdb.Collection(colOutbox).DeleteMany(ctx, bson.M{})
		require.NoError(t, err, "clear outbox collection between subtests")
		_, err = mdb.Collection(colContactRef).DeleteMany(ctx, bson.M{})
		require.NoError(t, err, "clear contact ref collection between subtests")
		return s
	})
}

// newMongoConformanceStore opens one store for the whole suite, isolated in
// its own database so a run never collides with a database an earlier run
// left behind. It also returns the underlying *mongodriver.MongoDB so the
// caller can clear collections between subtests.
func newMongoConformanceStore(t *testing.T) (Store, *mongodriver.MongoDB) {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo integration test")
	}
	ctx := context.Background()

	// Mongo caps a database name at 63 bytes, so this takes the tail of a
	// fresh ID rather than the whole prefixed string.
	unique := strings.ReplaceAll(id.NewRetentionJobID().String(), "_", "")
	dbName := "retention_conf_" + unique[len(unique)-16:]
	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri, mongodriver.WithDatabase(dbName)),
		"open grove mongo connection")

	db, err := grove.Open(mdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() {
		_ = mdb.Database().Drop(context.Background())
		_ = db.Close()
	})

	// The core migrations satisfy the group's DependsOn("authsome").
	require.NoError(t, mongostore.New(db).Migrate(ctx, MongoMigrations), "run migrations")
	return NewMongoStore(db), mdb
}
