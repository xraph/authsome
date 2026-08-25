package oauth2provider_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// newSQLiteClientStore opens an embedded SQLite database, runs the core
// migrations plus the oauth2provider SQLite migration group (which is what
// actually creates the dpop_mode column), seeds a real authsome_apps row,
// and returns a real SQL-backed oauth2provider.Store.
//
// The Postgres and SQLite stores share the same oauth2ClientModel row struct
// and the fromOAuth2Client/toOAuth2Client converters in store_models.go (see
// store_postgres.go and store_sqlite.go: neither defines its own column list
// or scan code). So exercising the round trip through embedded SQLite here
// also proves out the mapping path Postgres uses, without needing a
// container. It does not cover store_mongo.go, which has its own separate
// oauth2ClientDoc struct and converters.
//
// authsome_oauth2_clients.app_id is a real foreign key into authsome_apps in
// this schema (unlike the sso plugin's connections table, which happens to
// share a name with a table the core migrations already create without an
// FK, so its own FK-bearing CREATE TABLE never actually runs). Foreign keys
// are verified ON here: an earlier version of this test that skipped seeding
// an app row failed with "FOREIGN KEY constraint failed" on every insert,
// which is how this was caught.
func newSQLiteClientStore(t *testing.T) (oauth2provider.Store, id.AppID) {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "oauth2-dpop.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	require.NoError(t, core.Migrate(ctx, oauth2provider.SqliteMigrations))

	appID := id.NewAppID()
	require.NoError(t, core.CreateApp(ctx, &app.App{
		ID:        appID,
		Name:      "DPoP test app",
		Slug:      "dpop-test-app-" + appID.String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	return oauth2provider.NewSqliteStore(db), appID
}

// fixedTimestamp works around a pre-existing, unrelated bug: modernc/sqlite
// serializes a time.Time write parameter with time.Time.String(), which
// appends a " m=+..." monotonic-clock reading whenever the value came from
// time.Now(). Grove's own TEXT-column scan parser (grove's
// scan/convert.go:parseTimeString) does not strip that suffix before
// matching it against its known layouts, so any TEXT-declared timestamp
// column (which is what every hand-written CREATE TABLE in this file uses
// for created_at/updated_at, including authsome_oauth2_clients) fails to
// scan back a row written with a fresh time.Now(). time.Date gives a
// monotonic-free time.Time, sidestepping that bug so this test can exercise
// the property it actually cares about: dpop_mode's own column mapping.
// See the test's report for a fuller writeup; this is not a DPoP-specific
// problem and is not fixed here.
var fixedTimestamp = time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

func TestClientDPoPMode_SQLiteRoundTrip(t *testing.T) {
	s, appID := newSQLiteClientStore(t)
	ctx := context.Background()

	client := &oauth2provider.OAuth2Client{
		ID:         id.NewOAuth2ClientID(),
		AppID:      appID,
		ClientID:   "client-dpop-sqlite",
		Name:       "DPoP SQLite client",
		DPoPMode:   "required",
		GrantTypes: []string{"authorization_code"},
		CreatedAt:  fixedTimestamp,
		UpdatedAt:  fixedTimestamp,
	}
	require.NoError(t, s.CreateClient(ctx, client))

	got, err := s.GetClient(ctx, "client-dpop-sqlite")
	require.NoError(t, err)
	assert.Equal(t, "required", got.DPoPMode)
}

func TestClientDPoPMode_SQLiteDefaultsToInherit(t *testing.T) {
	s, appID := newSQLiteClientStore(t)
	ctx := context.Background()

	client := &oauth2provider.OAuth2Client{
		ID:         id.NewOAuth2ClientID(),
		AppID:      appID,
		ClientID:   "client-plain-sqlite",
		Name:       "Plain SQLite client",
		GrantTypes: []string{"authorization_code"},
		CreatedAt:  fixedTimestamp,
		UpdatedAt:  fixedTimestamp,
	}
	require.NoError(t, s.CreateClient(ctx, client))

	got, err := s.GetClient(ctx, "client-plain-sqlite")
	require.NoError(t, err)
	assert.Empty(t, got.DPoPMode, "empty means inherit from the app, matching TokenFormat")
}
