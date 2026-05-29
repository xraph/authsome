//go:build integration

package mongo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	_ "github.com/xraph/grove/drivers/mongodriver/mongomigrate"

	mongostore "github.com/xraph/authsome/store/mongo"
)

// openTestMongo opens a grove mongo connection or skips the test when
// AUTHSOME_MONGO_URI is unset.
func openTestMongo(t *testing.T) (*mongodriver.MongoDB, *grove.DB) {
	t.Helper()
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

	return mdb, db
}

// TestMigrate_BreaksStaleLock verifies that a Migrate call recovers from
// a stale lock document — the failure mode that occurs when a previous
// process holding the migration lock crashes before its defer can fire.
func TestMigrate_BreaksStaleLock(t *testing.T) {
	mdb, db := openTestMongo(t)
	ctx := context.Background()

	s := mongostore.New(db)
	// First Migrate establishes the lock collection in the unlocked state.
	require.NoError(t, s.Migrate(ctx), "initial migrate")

	// Forge a stale lock: locked=true with locked_at well past the
	// threshold (set in store.go to 5m). 1h is comfortably stale.
	staleAt := time.Now().Add(-1 * time.Hour)
	lockColl := mdb.Collection("grove_migration_locks")
	_, err := lockColl.UpdateOne(ctx,
		bson.M{"_id": "grove_migration_lock"},
		bson.M{"$set": bson.M{
			"locked":    true,
			"locked_at": staleAt,
			"locked_by": "fake-crashed-process",
		}},
	)
	require.NoError(t, err, "forge stale lock")

	// Migrate must succeed by detecting + breaking the stale lock.
	require.NoError(t, s.Migrate(ctx), "migrate should self-heal stale lock")

	// And the lock must be released after the run.
	var post struct {
		Locked bool `bson:"locked"`
	}
	require.NoError(t,
		lockColl.FindOne(ctx, bson.M{"_id": "grove_migration_lock"}).Decode(&post),
		"read post-migrate lock",
	)
	assert.False(t, post.Locked, "lock must be released after successful migrate")
}

// TestMigrate_DoesNotBreakFreshLock verifies that a fresh lock (one that
// could plausibly belong to a live migrator) is NOT broken. Migrate
// should fail rather than racing a concurrent process.
func TestMigrate_DoesNotBreakFreshLock(t *testing.T) {
	mdb, db := openTestMongo(t)
	ctx := context.Background()

	s := mongostore.New(db)
	require.NoError(t, s.Migrate(ctx), "initial migrate")

	freshAt := time.Now()
	lockColl := mdb.Collection("grove_migration_locks")
	_, err := lockColl.UpdateOne(ctx,
		bson.M{"_id": "grove_migration_lock"},
		bson.M{"$set": bson.M{
			"locked":    true,
			"locked_at": freshAt,
			"locked_by": "concurrent-process",
		}},
	)
	require.NoError(t, err, "forge fresh lock")

	t.Cleanup(func() {
		// Release the lock so subsequent tests aren't wedged.
		_, _ = lockColl.UpdateOne(ctx,
			bson.M{"_id": "grove_migration_lock"},
			bson.M{"$set": bson.M{"locked": false, "locked_at": nil, "locked_by": nil}},
		)
	})

	err = s.Migrate(ctx)
	require.Error(t, err, "migrate must not break a fresh lock")
	assert.Contains(t, err.Error(), "lock is held by another process")
}
