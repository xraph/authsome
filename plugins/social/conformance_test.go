package social_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/social/socialtest"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
	"github.com/xraph/authsome/store/storetest"
)

// backend is one store implementation under test. setup runs once per backend
// and returns the factory used for each case, so a container is built once
// rather than per case.
type backend struct {
	name  string
	setup func(t *testing.T) socialtest.Factory
}

// extraBackends is appended to by the integration-tagged file, keeping one
// TestConformance entry point regardless of build tags.
var extraBackends []backend

// TestConformance runs the social store contract against every backend. Memory
// and SQLite always run; postgres and mongo join under `-tags integration`.
func TestConformance(t *testing.T) {
	backends := append([]backend{
		{"Memory", func(*testing.T) socialtest.Factory { return memoryFixture }},
		{"SQLite", func(*testing.T) socialtest.Factory { return sqliteFixture }},
	}, extraBackends...)

	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			socialtest.RunConformance(t, b.setup(t))
		})
	}
}

// seedFixture creates the two tenants the scoping cases compare against.
func seedFixture(t *testing.T, core store.Store, plugin social.Store) socialtest.Fixture {
	t.Helper()
	tn := storetest.SeedTenant(t, core)
	other := storetest.SeedTenant(t, core)
	return socialtest.Fixture{
		Store:        plugin,
		AppID:        tn.AppID,
		UserID:       storetest.SeedUser(t, core, tn, "social-a@example.test"),
		OtherUserID:  storetest.SeedUser(t, core, tn, "social-b@example.test"),
		OtherAppID:   other.AppID,
		OtherAppUser: storetest.SeedUser(t, core, other, "social-c@example.test"),
	}
}

func memoryFixture(t *testing.T) socialtest.Fixture {
	t.Helper()
	return seedFixture(t, memory.New(), social.NewMemoryStore())
}

func sqliteFixture(t *testing.T) socialtest.Fixture {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "socialconf.db") + "?cache=shared&_pragma=foreign_keys(1)"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	// The plugin table carries foreign keys into the core schema, so both
	// migration groups have to run before anything can be written.
	require.NoError(t, core.Migrate(ctx, social.SqliteMigrations))

	return seedFixture(t, core, social.NewSqliteStore(db))
}
