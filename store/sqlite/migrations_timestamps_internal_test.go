//go:build integration

package sqlite

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	"github.com/xraph/grove/migrate"
)

// openExec opens a fresh temp-file sqlite db and returns a migrate.Executor
// for it. The shared cache + file backing mirror setupTestStore.
func openExec(t *testing.T) migrate.Executor {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "rebuild.db") + "?cache=shared&_pragma=foreign_keys(1)"

	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	t.Cleanup(func() { _ = sdb.Close() })

	exec, err := migrate.NewExecutorFor(sdb)
	require.NoError(t, err)
	return exec
}

// TestTimestampRebuild_CopiesExistingRows proves the rebuild helper preserves
// pre-existing rows: a row written to a TEXT timestamp column (the buggy
// shape) survives the rebuild with its value intact, and — because the column
// is now declared TIMESTAMP — it scans straight into a time.Time. Scanning the
// value into a time.Time is itself the exact round-trip the bug breaks, so a
// successful scan here is direct proof of the fix on copied data.
func TestTimestampRebuild_CopiesExistingRows(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	// Buggy v1 shape: created_at declared TEXT.
	_, err := exec.Exec(ctx, `CREATE TABLE widgets (
        id         TEXT PRIMARY KEY,
        name       TEXT NOT NULL,
        created_at TEXT NOT NULL
    );`)
	require.NoError(t, err)

	want := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	_, err = exec.Exec(ctx,
		`INSERT INTO widgets (id, name, created_at) VALUES ('w1', 'gear', ?);`,
		want.Format(time.RFC3339))
	require.NoError(t, err)

	require.NoError(t, rebuildTimestampTable(ctx, exec, timestampTableRebuild{
		table: "widgets",
		create: `CREATE TABLE widgets_new (
            id         TEXT PRIMARY KEY,
            name       TEXT NOT NULL,
            created_at TIMESTAMP NOT NULL
        );`,
		cols:    "id, name, created_at",
		indexes: ``,
	}))

	rows, err := exec.Query(ctx, `SELECT id, name, created_at FROM widgets;`)
	require.NoError(t, err)
	defer rows.Close()

	var count int
	var (
		id, name string
		got      time.Time
	)
	for rows.Next() {
		count++
		require.NoError(t, rows.Scan(&id, &name, &got))
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, 1, count, "rebuild must preserve the existing row")
	assert.Equal(t, "w1", id)
	assert.Equal(t, "gear", name)
	assert.WithinDuration(t, want, got, time.Second, "copied timestamp value lost")
}

// TestTimestampRebuild_SpecColumnsMatchSchema guards against the one mistake
// the migration cannot catch at run time on an empty database: a column
// present in a rebuild's CREATE statement but omitted from its copy list. Such
// an omission would silently drop that column's data during the rebuild. For
// every rebuilt table it asserts the live column set (PRAGMA table_info, which
// reflects the CREATE) equals the set named in the copy list.
func TestTimestampRebuild_SpecColumnsMatchSchema(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "schema.db") + "?cache=shared&_pragma=foreign_keys(1)"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	t.Cleanup(func() { _ = sdb.Close() })

	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, New(db).Migrate(ctx))

	exec, err := migrate.NewExecutorFor(sdb)
	require.NoError(t, err)

	for _, r := range timestampRebuilds {
		t.Run(r.table, func(t *testing.T) {
			live := tableColumns(t, ctx, exec, r.table)
			copyCols := splitCols(r.cols)
			assert.ElementsMatchf(t, live, copyCols,
				"table %s: copy list and live schema disagree — a missing copy column drops data", r.table)
		})
	}
}

// TestTimestampRebuild_PreservesAllColumns runs every registered migration up
// to (but not including) the rebuild, snapshots each rebuilt table's columns,
// then runs the rebuild and asserts no table's column set changed. This is the
// decisive guard against the rebuild accidentally dropping (or adding) a
// column relative to the real prior schema — something the empty-database copy
// cannot detect on its own.
func TestTimestampRebuild_PreservesAllColumns(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	const rebuildVersion = "20260601000002"

	var rebuild *migrate.Migration
	for _, m := range Migrations.Migrations() {
		if m.Version == rebuildVersion {
			rebuild = m
			break
		}
		require.NoErrorf(t, m.Up(ctx, exec), "pre-rebuild migration %s (%s)", m.Version, m.Name)
	}
	require.NotNilf(t, rebuild, "rebuild migration %s not registered", rebuildVersion)

	colsBefore := make(map[string][]string, len(timestampRebuilds))
	idxBefore := make(map[string][]string, len(timestampRebuilds))
	for _, r := range timestampRebuilds {
		colsBefore[r.table] = tableColumns(t, ctx, exec, r.table)
		idxBefore[r.table] = tableIndexes(t, ctx, exec, r.table)
	}

	require.NoError(t, rebuild.Up(ctx, exec))

	for _, r := range timestampRebuilds {
		t.Run(r.table, func(t *testing.T) {
			assert.ElementsMatchf(t, colsBefore[r.table], tableColumns(t, ctx, exec, r.table),
				"rebuild changed the column set of %s", r.table)
			assert.ElementsMatchf(t, idxBefore[r.table], tableIndexes(t, ctx, exec, r.table),
				"rebuild changed the index set of %s — a dropped unique index silently loses a constraint", r.table)
		})
	}
}

// tableColumns returns the column names of table via PRAGMA table_info.
func tableColumns(t *testing.T, ctx context.Context, exec migrate.Executor, table string) []string {
	t.Helper()
	rows, err := exec.Query(ctx, "PRAGMA table_info("+table+");")
	require.NoError(t, err)
	defer rows.Close()

	// table_info columns: cid, name, type, notnull, dflt_value, pk.
	var names []string
	for rows.Next() {
		var (
			cid, notnull, pk int64
			name, typ        string
			dflt             any
		)
		require.NoError(t, rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	require.NotEmptyf(t, names, "table %s has no columns (does it exist?)", table)
	sort.Strings(names)
	return names
}

// tableIndexes returns the names of all indexes on table (both explicitly
// created idx_* indexes and any auto-created PK/UNIQUE indexes).
func tableIndexes(t *testing.T, ctx context.Context, exec migrate.Executor, table string) []string {
	t.Helper()
	rows, err := exec.Query(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ?;", table)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	sort.Strings(names)
	return names
}

func splitCols(cols string) []string {
	parts := strings.Split(cols, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	sort.Strings(out)
	return out
}
