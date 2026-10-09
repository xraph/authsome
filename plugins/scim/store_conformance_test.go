package scim_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/plugins/scim/scimtest"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// backend is one store implementation under test. The integration-tagged
// file appends postgres and mongo, so one TestConformance covers all four.
type backend struct {
	name  string
	setup func(t *testing.T) scimtest.Factory
}

var extraBackends []backend

func fixtureFor(s scim.Store) scimtest.Factory {
	return func(*testing.T) scimtest.Fixture {
		return scimtest.Fixture{Store: s, AppID: id.NewAppID(), OrgID: id.NewOrgID()}
	}
}

func sqliteSetup(t *testing.T) scimtest.Factory {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "scim-conformance.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlitestore.New(db).Migrate(ctx, scim.SqliteMigrations))
	return fixtureFor(scim.NewSqliteStore(db))
}

func TestConformance(t *testing.T) {
	backends := append([]backend{
		{"Memory", func(*testing.T) scimtest.Factory { return fixtureFor(scim.NewMemoryStore()) }},
		{"SQLite", sqliteSetup},
	}, extraBackends...)
	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			scimtest.RunConformance(t, b.setup(t))
		})
	}
}
