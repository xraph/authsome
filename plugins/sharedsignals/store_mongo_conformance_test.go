//go:build integration

package sharedsignals

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	_ "github.com/xraph/grove/drivers/mongodriver/mongomigrate"

	"github.com/xraph/authsome/id"
	mongostore "github.com/xraph/authsome/store/mongo"
)

// The mongo store was the one backend with no evidence beyond doc converters:
// store_mongo_test.go round-trips structs through BSON tags in memory, which
// proves the field names line up and nothing else. It cannot catch a filter
// that queries the wrong key, a sort that never applies, or a unique index the
// migration forgot -- and the replay guard IS a unique index.
//
// So this runs the same contract suite the SQL backends run, gated the way the
// rest of the repo gates mongo work: a build tag plus AUTHSOME_MONGO_URI, the
// pattern store/mongo/conformance_test.go already uses. It stays out of the
// default `go test ./...` because it needs a live server, and CI can turn it on
// by setting the variable and passing -tags integration.
//
//	AUTHSOME_MONGO_URI=mongodb://localhost:27017 go test -tags integration \
//	    ./plugins/sharedsignals/ -run TestStoreConformance_Mongo
func TestStoreConformance_Mongo(t *testing.T) {
	s := newMongoConformanceStore(t)
	runStoreConformance(t, func(_ *testing.T) Store { return s })
}

// newMongoConformanceStore opens one store for the whole suite. The sub-tests
// isolate on random IDs, so sharing is safe, with one exception worth naming:
// push_path_hash is unique and the suite uses fixed literals for it, so a
// second run against a database the first one left behind would collide. Each
// run therefore gets its own database, dropped on the way out.
func newMongoConformanceStore(t *testing.T) Store {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo integration test")
	}
	ctx := context.Background()

	// Mongo caps a database name at 63 bytes, so this takes the tail of a
	// fresh ID rather than the whole prefixed string.
	unique := strings.ReplaceAll(id.NewSSFStreamID().String(), "_", "")
	dbName := "ssf_conf_" + unique[len(unique)-16:]
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
	return NewMongoStore(db)
}
