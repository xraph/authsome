package oauth2provider_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/plugins/oauth2provider/oauth2test"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
	"github.com/xraph/authsome/store/storetest"
)

// backend is one store implementation under test. setup runs once per
// backend, receiving that backend's own *testing.T, and returns the factory
// used for each case. Backends backed by a container build it in setup so the
// cost is paid once rather than per case.
type backend struct {
	name  string
	setup func(t *testing.T) oauth2test.Factory
}

// extraBackends holds the backends that need external infrastructure. The
// integration-tagged file appends postgres and mongo to it, so this package
// keeps a single TestConformance entry point regardless of build tags.
var extraBackends []backend

// TestConformance runs the OAuth2 provider store contract against every
// backend. Memory and SQLite always run; postgres and mongo join under
// `-tags integration`.
func TestConformance(t *testing.T) {
	backends := append([]backend{
		{"Memory", func(*testing.T) oauth2test.Factory { return memoryFixture }},
		{"SQLite", func(*testing.T) oauth2test.Factory { return sqliteFixture }},
	}, extraBackends...)

	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			oauth2test.RunConformance(t, b.setup(t))
		})
	}
}

// seedFixture creates the two tenants and the user that the plugin rows point
// at, then pairs them with the plugin store under test.
func seedFixture(t *testing.T, core store.Store, plugin oauth2provider.Store) oauth2test.Fixture {
	t.Helper()
	tn := storetest.SeedTenant(t, core)
	other := storetest.SeedTenant(t, core)
	return oauth2test.Fixture{
		Store:      plugin,
		AppID:      tn.AppID,
		UserID:     storetest.SeedUser(t, core, tn, "oauth2-conformance@example.test"),
		OtherAppID: other.AppID,
	}
}

func memoryFixture(t *testing.T) oauth2test.Fixture {
	t.Helper()
	return seedFixture(t, memory.New(), oauth2provider.NewMemoryStore())
}

func sqliteFixture(t *testing.T) oauth2test.Fixture {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "oauth2conf.db") + "?cache=shared&_pragma=foreign_keys(1)"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	// The plugin's tables carry foreign keys into the core schema, so both
	// migration groups have to run before anything can be written.
	require.NoError(t, core.Migrate(ctx, oauth2provider.SqliteMigrations))

	return seedFixture(t, core, oauth2provider.NewSqliteStore(db))
}
