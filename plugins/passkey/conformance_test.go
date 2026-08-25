package passkey_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/passkey/passkeytest"
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
	setup func(t *testing.T) passkeytest.Factory
}

// extraBackends is appended to by the integration-tagged file, keeping one
// TestConformance entry point regardless of build tags.
var extraBackends []backend

// TestConformance runs the passkey store contract against every backend.
// Memory and SQLite always run; postgres and mongo join under
// `-tags integration`.
func TestConformance(t *testing.T) {
	backends := append([]backend{
		{"Memory", func(*testing.T) passkeytest.Factory { return memoryFixture }},
		{"SQLite", func(*testing.T) passkeytest.Factory { return sqliteFixture }},
	}, extraBackends...)

	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			passkeytest.RunConformance(t, b.setup(t))
		})
	}
}

// seedFixture creates the tenant and the two users the credentials hang off.
func seedFixture(t *testing.T, core store.Store, plugin passkey.Store) passkeytest.Fixture {
	t.Helper()
	tn := storetest.SeedTenant(t, core)
	return passkeytest.Fixture{
		Store:       plugin,
		AppID:       tn.AppID,
		UserID:      storetest.SeedUser(t, core, tn, "passkey-a@example.test"),
		OtherUserID: storetest.SeedUser(t, core, tn, "passkey-b@example.test"),
	}
}

func memoryFixture(t *testing.T) passkeytest.Fixture {
	t.Helper()
	return seedFixture(t, memory.New(), passkey.NewMemoryStore())
}

func sqliteFixture(t *testing.T) passkeytest.Fixture {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "passkeyconf.db") + "?cache=shared&_pragma=foreign_keys(1)"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	// The plugin table carries foreign keys into the core schema, so both
	// migration groups have to run before anything can be written.
	require.NoError(t, core.Migrate(ctx, passkey.SqliteMigrations))

	return seedFixture(t, core, passkey.NewSqliteStore(db))
}
