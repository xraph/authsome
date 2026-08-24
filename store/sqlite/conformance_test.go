//go:build integration

package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/store"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
	"github.com/xraph/authsome/store/storetest"
)

// TestConformance runs the shared cross-backend store contract suite against
// the SQLite backend. SQLite itself is embedded and needs no Docker, but this
// is gated behind the integration tag anyway (matching postgres and mongo) so
// that shared conformance cases added ahead of a backend's own implementation
// task, such as the principal/delegation cases added in the non-human-
// principals work, don't turn normal CI red before that backend's task lands.
func TestConformance(t *testing.T) {
	storetest.RunConformance(t, func(t *testing.T) store.Store {
		ctx := context.Background()
		dsn := "file:" + filepath.Join(t.TempDir(), "conf.db") + "?cache=shared&_pragma=foreign_keys(1)"
		sdb := sqlitedriver.New()
		require.NoError(t, sdb.Open(ctx, dsn))
		db, err := grove.Open(sdb)
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		s := sqlitestore.New(db)
		require.NoError(t, s.Migrate(ctx))
		return s
	})
}
