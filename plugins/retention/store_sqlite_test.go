package retention

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

func TestSqliteStoreConformance(t *testing.T) {
	runStoreConformance(t, func(t *testing.T) Store {
		t.Helper()
		ctx := context.Background()
		dsn := "file:" + filepath.Join(t.TempDir(), "retention-conformance.db") + "?cache=shared"
		sdb := sqlitedriver.New()
		require.NoError(t, sdb.Open(ctx, dsn))
		db, err := grove.Open(sdb)
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		// The core migrations satisfy the group's DependsOn("authsome").
		require.NoError(t, sqlitestore.New(db).Migrate(ctx, SqliteMigrations))
		return NewSqliteStore(db)
	})
}
