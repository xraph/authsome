//go:build integration

package mongo

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/id"
)

// openMigratedExec returns a migrated store plus the mongomigrate executor the
// backfill runs against, on a database unique to this test.
func openMigratedExec(t *testing.T) *mongomigrate.Executor {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo integration test")
	}
	ctx := context.Background()

	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri))
	db, err := grove.Open(mdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	s := New(db)
	require.NoError(t, s.Migrate(ctx))

	exec, err := migrate.NewExecutorFor(mdb)
	require.NoError(t, err)
	mexec, ok := exec.(*mongomigrate.Executor)
	require.True(t, ok, "expected mongomigrate executor, got %T", exec)
	return mexec
}

// seedLegacyMongoApp writes what a pre-env deployment looks like: an app with
// no environment, and a user doc with no env_id field at all. Mongo is
// schemaless, so old docs simply lack the field rather than carrying "".
func seedLegacyMongoApp(ctx context.Context, t *testing.T, mexec *mongomigrate.Executor) (appID, userID string) {
	t.Helper()
	appID = id.NewAppID().String()
	userID = id.NewUserID().String()
	db := mexec.DB()

	_, err := db.Collection(colApps).InsertOne(ctx, bson.M{
		"_id": appID, "name": "Legacy", "slug": "legacy-" + appID[len(appID)-8:],
	})
	require.NoError(t, err)

	_, err = db.Collection(colUsers).InsertOne(ctx, bson.M{
		"_id": userID, "app_id": appID, "email": "legacy-" + userID[len(userID)-8:] + "@test.com",
	})
	require.NoError(t, err)

	return appID, userID
}

func mongoEnvIDOfUser(ctx context.Context, t *testing.T, mexec *mongomigrate.Executor, userID string) string {
	t.Helper()
	var doc struct {
		EnvID string `bson:"env_id"`
	}
	require.NoError(t, mexec.DB().Collection(colUsers).
		FindOne(ctx, bson.M{"_id": userID}).Decode(&doc))
	return doc.EnvID
}

func mongoDefaultEnvCount(ctx context.Context, t *testing.T, mexec *mongomigrate.Executor, appID string) int64 {
	t.Helper()
	n, err := mexec.DB().Collection(colEnvironments).
		CountDocuments(ctx, bson.M{"app_id": appID, "is_default": true})
	require.NoError(t, err)
	return n
}

// A doc predating the environments work carries no env_id, so an env-scoped
// lookup cannot find it. The backfill has to give it one.
func TestMongoBackfillDefaultEnvironments_StampsLegacyDocs(t *testing.T) {
	ctx := context.Background()
	mexec := openMigratedExec(t)

	appID, userID := seedLegacyMongoApp(ctx, t, mexec)
	require.Equal(t, "", mongoEnvIDOfUser(ctx, t, mexec, userID), "precondition: no env_id")
	require.Equal(t, int64(0), mongoDefaultEnvCount(ctx, t, mexec, appID))

	require.NoError(t, backfillDefaultEnvironments(ctx, mexec))

	assert.Equal(t, int64(1), mongoDefaultEnvCount(ctx, t, mexec, appID),
		"the app must gain exactly one default environment")
	got := mongoEnvIDOfUser(ctx, t, mexec, userID)
	_, parseErr := id.ParseEnvironmentID(got)
	assert.NoError(t, parseErr, "stamped value must be an environment id, got %q", got)
}

// The migration can be re-run. A second pass must not mint another default
// environment or move a user that already has one.
func TestMongoBackfillDefaultEnvironments_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	mexec := openMigratedExec(t)

	appID, userID := seedLegacyMongoApp(ctx, t, mexec)
	require.NoError(t, backfillDefaultEnvironments(ctx, mexec))
	first := mongoEnvIDOfUser(ctx, t, mexec, userID)

	require.NoError(t, backfillDefaultEnvironments(ctx, mexec))

	assert.Equal(t, int64(1), mongoDefaultEnvCount(ctx, t, mexec, appID))
	assert.Equal(t, first, mongoEnvIDOfUser(ctx, t, mexec, userID),
		"a second pass must not move a user that already has an environment")
}

// An app that already has a default environment keeps it, rather than having
// its users re-pointed at a freshly minted one.
func TestMongoBackfillDefaultEnvironments_ReusesExistingDefault(t *testing.T) {
	ctx := context.Background()
	mexec := openMigratedExec(t)

	appID, userID := seedLegacyMongoApp(ctx, t, mexec)
	existing := id.NewEnvironmentID().String()
	_, err := mexec.DB().Collection(colEnvironments).InsertOne(ctx, bson.M{
		"_id": existing, "app_id": appID, "name": "Production",
		"slug": "production", "type": "production", "is_default": true,
	})
	require.NoError(t, err)

	require.NoError(t, backfillDefaultEnvironments(ctx, mexec))

	assert.Equal(t, int64(1), mongoDefaultEnvCount(ctx, t, mexec, appID))
	assert.Equal(t, existing, mongoEnvIDOfUser(ctx, t, mexec, userID),
		"the app's existing default environment must be reused, not replaced")
}
