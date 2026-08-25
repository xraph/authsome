package scim

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"
	"github.com/xraph/grove/migrate"
)

const timestampMigrationVersion = "20260824000050"

// The SCIM plugin has no SQLite store yet, so there is no CreateX/GetX pair to
// round-trip. Instead this drives the migration directly: bring a database up to
// the state right before the rebuild, write rows the way the modernc driver
// writes a time.Now() value, then run the rebuild and read them back.
//
// authsome_scim_tokens references authsome_scim_configs ON DELETE CASCADE, which
// makes this the schema where a naive rebuild does real damage: dropping the
// configs table with foreign keys enforced would take every token with it.
func TestTimestampRebuild_ScimTables(t *testing.T) {
	ctx := context.Background()
	exec := openMigrationExec(t)

	rebuild := runMigrationsBefore(ctx, t, exec, timestampMigrationVersion)

	// Exactly what modernc.org/sqlite writes for time.Now() with no _time_format
	// in the DSN: time.Time.String(), monotonic clock reading included.
	const stored = "2026-01-15 10:30:00.123456789 -0600 CST m=+0.016649834"
	want := time.Date(2026, 1, 15, 16, 30, 0, 123456789, time.UTC)

	mustExec(ctx, t, exec, `INSERT INTO authsome_scim_configs (id, app_id, name, created_at, updated_at)
        VALUES ('cfg1', 'app1', 'Okta', ?, ?);`, stored, stored)
	mustExec(ctx, t, exec, `INSERT INTO authsome_scim_tokens (id, config_id, name, token_hash, created_at)
        VALUES ('tok1', 'cfg1', 'primary', 'hash', ?);`, stored)
	mustExec(ctx, t, exec, `INSERT INTO authsome_scim_provision_logs
        (id, config_id, action, resource_type, status, created_at)
        VALUES ('log1', 'cfg1', 'create', 'User', 'ok', ?);`, stored)

	require.NoError(t, rebuild.Up(ctx, exec))

	// The child rows are the cascade canary: a rebuild that drops the configs
	// table with foreign keys still enforced empties both of these.
	assert.EqualValues(t, 1, countRows(ctx, t, exec, "authsome_scim_configs"), "config row lost")
	assert.EqualValues(t, 1, countRows(ctx, t, exec, "authsome_scim_tokens"),
		"token rows were cascade-deleted by the configs rebuild")
	assert.EqualValues(t, 1, countRows(ctx, t, exec, "authsome_scim_provision_logs"),
		"provision log rows were cascade-deleted by the configs rebuild")

	// Scanning into a time.Time is the round-trip the TEXT declaration breaks,
	// so a clean scan here is the fix itself.
	for _, table := range []string{
		"authsome_scim_configs",
		"authsome_scim_tokens",
		"authsome_scim_provision_logs",
	} {
		t.Run(table, func(t *testing.T) {
			rows, err := exec.Query(ctx, "SELECT created_at FROM "+table+";")
			require.NoError(t, err)
			defer rows.Close()

			require.True(t, rows.Next())
			var got time.Time
			require.NoError(t, rows.Scan(&got))
			require.NoError(t, rows.Err())
			assert.WithinDuration(t, want, got, time.Second, "created_at did not survive the rebuild")
		})
	}
}

// TestTimestampRebuild_ScimCascadeStillWorks proves the rebuild kept the
// ON DELETE CASCADE clauses rather than quietly dropping them.
func TestTimestampRebuild_ScimCascadeStillWorks(t *testing.T) {
	ctx := context.Background()
	exec := openMigrationExec(t)

	rebuild := runMigrationsBefore(ctx, t, exec, timestampMigrationVersion)
	require.NoError(t, rebuild.Up(ctx, exec))

	now := time.Now()
	mustExec(ctx, t, exec, `INSERT INTO authsome_scim_configs (id, app_id, name, created_at, updated_at)
        VALUES ('cfg1', 'app1', 'Okta', ?, ?);`, now, now)
	mustExec(ctx, t, exec, `INSERT INTO authsome_scim_tokens (id, config_id, name, token_hash, created_at)
        VALUES ('tok1', 'cfg1', 'primary', 'hash', ?);`, now)

	mustExec(ctx, t, exec, `DELETE FROM authsome_scim_configs WHERE id = 'cfg1';`)
	assert.EqualValues(t, 0, countRows(ctx, t, exec, "authsome_scim_tokens"),
		"ON DELETE CASCADE was lost in the rebuild")
}

func openMigrationExec(t *testing.T) migrate.Executor {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "scim-timestamps.db") + "?cache=shared"

	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	t.Cleanup(func() { _ = sdb.Close() })

	exec, err := migrate.NewExecutorFor(sdb)
	require.NoError(t, err)
	return exec
}

// runMigrationsBefore applies every SQLite migration ordered before version and
// returns the one at version, left un-applied for the caller to drive.
func runMigrationsBefore(ctx context.Context, t *testing.T, exec migrate.Executor, version string) *migrate.Migration {
	t.Helper()
	for _, m := range SqliteMigrations.Migrations() {
		if m.Version == version {
			return m
		}
		require.NoErrorf(t, m.Up(ctx, exec), "migration %s (%s)", m.Version, m.Name)
	}
	t.Fatalf("migration %s is not registered", version)
	return nil
}

func mustExec(ctx context.Context, t *testing.T, exec migrate.Executor, query string, args ...any) {
	t.Helper()
	_, err := exec.Exec(ctx, query, args...)
	require.NoErrorf(t, err, "exec: %s", query)
}

func countRows(ctx context.Context, t *testing.T, exec migrate.Executor, table string) int64 {
	t.Helper()
	rows, err := exec.Query(ctx, "SELECT COUNT(*) FROM "+table+";")
	require.NoError(t, err)
	defer rows.Close()

	var n int64
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&n))
	require.NoError(t, rows.Err())
	return n
}
