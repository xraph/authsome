//go:build integration

package waitlist_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/waitlist"
	mongostore "github.com/xraph/authsome/store/mongo"
	"github.com/xraph/authsome/store/storetest"
)

const waitlistColl = "authsome_waitlist_entries"

// openMongo brings up a store with the core schema only. The waitlist mongo
// group is deliberately left unapplied so a test can put rows in before the
// unique index exists.
func openMongo(t *testing.T, dbName string) (*mongodriver.MongoDB, *mongostore.Store) {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo integration test")
	}
	// Point at a database of this test's own, so seeded duplicates cannot
	// leak into another package's run.
	if i := strings.Index(uri, "?"); i >= 0 {
		uri = uri[:strings.LastIndex(uri[:i], "/")+1] + dbName + uri[i:]
	}
	ctx := context.Background()

	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri))
	db, err := grove.Open(mdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := mongostore.New(db)
	require.NoError(t, core.Migrate(ctx), "core migrations")
	return mdb, core
}

// TestMigration_WaitlistUniqueEmail_RefusesOnExistingDuplicates covers the guard
// on the way in. A collection that already holds two entries for one address
// must stop the migration with something an operator can act on, and must
// still hold both entries afterwards.
func TestMigration_WaitlistUniqueEmail_RefusesOnExistingDuplicates(t *testing.T) {
	ctx := context.Background()
	mdb, core := openMongo(t, "authsome_wl_dupes")

	tn := storetest.SeedTenant(t, core)
	coll := mdb.Collection(waitlistColl)
	_, err := coll.DeleteMany(ctx, bson.M{})
	require.NoError(t, err, "clear collection")

	const email = "twice@example.test"
	for range 2 {
		_, err := coll.InsertOne(ctx, bson.M{
			"_id": id.NewWaitlistID().String(), "app_id": tn.AppID.String(),
			"email": email, "name": "Test", "status": "pending",
			"ip_address": "203.0.113.1", "note": "",
			"created_at": time.Now().UTC(), "updated_at": time.Now().UTC(),
		})
		require.NoError(t, err, "seed duplicate")
	}

	err = core.Migrate(ctx, waitlist.MongoMigrations)
	require.Error(t, err, "the migration must refuse rather than build an index over duplicates")

	msg := err.Error()
	t.Logf("operator sees:\n%s", msg)
	require.Contains(t, msg, email, "the error must name the address so it can be found")
	require.Contains(t, msg, "2 entries", "the error must say how many entries collide")
	require.Contains(t, msg, "will not choose which entries to remove",
		"the error must say the migration declined to delete, not that it failed obscurely")

	// Nothing removed. A failed migration must not have eaten a signup.
	n, err := coll.CountDocuments(ctx, bson.M{"app_id": tn.AppID.String(), "email": email})
	require.NoError(t, err)
	require.EqualValues(t, 2, n, "the migration deleted rows while refusing to run")
}

// TestMigration_WaitlistUniqueEmail_AppliesOnCleanCollection is the other half:
// with nothing colliding, the migration runs and the index it creates is the
// one doing the work afterwards.
func TestMigration_WaitlistUniqueEmail_AppliesOnCleanCollection(t *testing.T) {
	ctx := context.Background()
	mdb, core := openMongo(t, "authsome_wl_clean")

	coll := mdb.Collection(waitlistColl)
	_, err := coll.DeleteMany(ctx, bson.M{})
	require.NoError(t, err)

	require.NoError(t, core.Migrate(ctx, waitlist.MongoMigrations),
		"a clean collection must migrate without complaint")

	cursor, err := coll.Indexes().List(ctx)
	require.NoError(t, err)
	defer func() { _ = cursor.Close(ctx) }()

	var found bool
	for cursor.Next(ctx) {
		var idx struct {
			Name   string `bson:"name"`
			Unique bool   `bson:"unique"`
		}
		require.NoError(t, cursor.Decode(&idx))
		if idx.Name == "app_id_1_email_1" {
			found = true
			require.True(t, idx.Unique, "the index exists but is not unique, so it enforces nothing")
		}
	}
	require.True(t, found, "the migration did not create the (app_id, email) index")
}
