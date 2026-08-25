package sharedsignals_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	ssf "github.com/xraph/authsome/plugins/sharedsignals"
	"github.com/xraph/authsome/plugins/sharedsignals/ssftest"
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
	setup func(t *testing.T) ssftest.Factory
}

// extraBackends is appended to by the integration-tagged file, keeping one
// TestConformance entry point regardless of build tags.
var extraBackends []backend

// TestConformance runs the shared signals store contract against every backend. Memory
// and SQLite always run; postgres and mongo join under `-tags integration`.
func TestConformance(t *testing.T) {
	backends := append([]backend{
		{"Memory", func(*testing.T) ssftest.Factory { return memoryFixture }},
		{"SQLite", func(*testing.T) ssftest.Factory { return sqliteFixture }},
	}, extraBackends...)

	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			ssftest.RunConformance(t, b.setup(t))
		})
	}
}

// seedFixture creates the two tenants the scoping cases compare against.
func seedFixture(t *testing.T, core store.Store, plugin ssf.Store) ssftest.Fixture {
	t.Helper()
	tn := storetest.SeedTenant(t, core)
	other := storetest.SeedTenant(t, core)
	return ssftest.Fixture{
		Store:      plugin,
		AppID:      tn.AppID,
		EnvID:      tn.EnvID,
		OtherAppID: other.AppID,
		OtherEnvID: other.EnvID,
		UserID:     storetest.SeedUser(t, core, tn, "ssf-conformance@example.test"),
	}
}

func memoryFixture(t *testing.T) ssftest.Fixture {
	t.Helper()
	return seedFixture(t, memory.New(), ssf.NewMemoryStore())
}

func sqliteFixture(t *testing.T) ssftest.Fixture {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "ssfconf.db") + "?cache=shared&_pragma=foreign_keys(1)"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	// The plugin table carries foreign keys into the core schema, so both
	// migration groups have to run before anything can be written.
	require.NoError(t, core.Migrate(ctx, ssf.SqliteMigrations))

	return seedFixture(t, core, ssf.NewSqliteStore(db))
}
