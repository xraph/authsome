package sqliteschema

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

// openExec opens a fresh temp-file sqlite database and returns a migrate.Executor
// for it, the same way the migration runner would.
func openExec(t *testing.T) migrate.Executor {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "rebuild.db") + "?cache=shared"

	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	t.Cleanup(func() { _ = sdb.Close() })

	exec, err := migrate.NewExecutorFor(sdb)
	require.NoError(t, err)
	return exec
}

func mustExec(t *testing.T, exec migrate.Executor, query string, args ...any) {
	t.Helper()
	_, err := exec.Exec(context.Background(), query, args...)
	require.NoErrorf(t, err, "exec: %s", query)
}

func countRows(t *testing.T, exec migrate.Executor, table string) int64 {
	t.Helper()
	rows, err := exec.Query(context.Background(), "SELECT COUNT(*) FROM "+table+";")
	require.NoError(t, err)
	defer rows.Close()

	var n int64
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&n))
	require.NoError(t, rows.Err())
	return n
}

// TestRebuildTables_CopiesRowsAndScansAsTime is the direct proof of the fix: a
// row written into a TEXT-declared timestamp column with the exact string the
// modernc driver produces for time.Now() — monotonic suffix and all — survives
// the rebuild and then scans straight into a time.Time. Scanning into a
// time.Time is the round-trip the TEXT declaration breaks, so a clean scan here
// is the fix itself, not a proxy for it.
func TestRebuildTables_CopiesRowsAndScansAsTime(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	mustExec(t, exec, `CREATE TABLE widgets (
        id         TEXT PRIMARY KEY,
        name       TEXT NOT NULL,
        created_at TEXT NOT NULL
    );`)
	mustExec(t, exec, `CREATE INDEX idx_widgets_name ON widgets (name);`)

	// Exactly what modernc.org/sqlite writes for a time.Now() value when the
	// DSN carries no _time_format: time.Time.String(), monotonic clock included.
	const stored = "2026-01-15 10:30:00.123456789 -0600 CST m=+0.016649834"
	want := time.Date(2026, 1, 15, 16, 30, 0, 123456789, time.UTC)
	mustExec(t, exec, `INSERT INTO widgets (id, name, created_at) VALUES ('w1', 'gear', ?);`, stored)

	require.NoError(t, RebuildTables(ctx, exec, []TableRebuild{{
		Table: "widgets",
		Create: `CREATE TABLE widgets_new (
            id         TEXT PRIMARY KEY,
            name       TEXT NOT NULL,
            created_at TIMESTAMP NOT NULL
        );`,
		Columns: "id, name, created_at",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_widgets_name ON widgets (name);`,
	}}))

	rows, err := exec.Query(ctx, `SELECT id, name, created_at FROM widgets;`)
	require.NoError(t, err)
	defer rows.Close()

	var (
		count    int
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

// TestRebuildTables_KeepsCascadeChildren covers the hazard that separates this
// helper from the core store's: dropping a parent table while foreign keys are
// enforced is an implicit DELETE, and every ON DELETE CASCADE pointing at it
// fires. Rebuilding a parent naively empties its children. Here the child rows
// must still be there afterwards, and the cascade must still work once the
// rebuild is done.
func TestRebuildTables_KeepsCascadeChildren(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	mustExec(t, exec, `CREATE TABLE configs (
        id         TEXT PRIMARY KEY,
        created_at TEXT NOT NULL
    );`)
	mustExec(t, exec, `CREATE TABLE tokens (
        id         TEXT PRIMARY KEY,
        config_id  TEXT NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
        created_at TEXT NOT NULL
    );`)
	mustExec(t, exec, `INSERT INTO configs (id, created_at) VALUES ('c1', '2026-01-15T10:30:00Z');`)
	mustExec(t, exec, `INSERT INTO tokens (id, config_id, created_at) VALUES ('t1', 'c1', '2026-01-15T10:30:00Z');`)

	require.NoError(t, RebuildTables(ctx, exec, []TableRebuild{
		{
			Table: "configs",
			Create: `CREATE TABLE configs_new (
                id         TEXT PRIMARY KEY,
                created_at TIMESTAMP NOT NULL
            );`,
			Columns: "id, created_at",
		},
		{
			Table: "tokens",
			Create: `CREATE TABLE tokens_new (
                id         TEXT PRIMARY KEY,
                config_id  TEXT NOT NULL REFERENCES configs(id) ON DELETE CASCADE,
                created_at TIMESTAMP NOT NULL
            );`,
			Columns: "id, config_id, created_at",
		},
	}))

	assert.EqualValues(t, 1, countRows(t, exec, "configs"), "parent row lost")
	assert.EqualValues(t, 1, countRows(t, exec, "tokens"),
		"child rows were cascade-deleted by the parent rebuild")

	// The rebuilt child still declares the cascade, and foreign keys are back on.
	mustExec(t, exec, `DELETE FROM configs WHERE id = 'c1';`)
	assert.EqualValues(t, 0, countRows(t, exec, "tokens"),
		"ON DELETE CASCADE was lost in the rebuild")
}

// TestRebuildTables_RestoresForeignKeyEnforcement guards the other half of the
// pragma dance: enforcement must be back on when the migration returns, or every
// later write in the process runs unchecked.
func TestRebuildTables_RestoresForeignKeyEnforcement(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	mustExec(t, exec, `CREATE TABLE parents (id TEXT PRIMARY KEY);`)
	mustExec(t, exec, `CREATE TABLE kids (
        id         TEXT PRIMARY KEY,
        parent_id  TEXT NOT NULL REFERENCES parents(id),
        created_at TEXT NOT NULL
    );`)

	require.NoError(t, RebuildTables(ctx, exec, []TableRebuild{{
		Table: "kids",
		Create: `CREATE TABLE kids_new (
            id         TEXT PRIMARY KEY,
            parent_id  TEXT NOT NULL REFERENCES parents(id),
            created_at TIMESTAMP NOT NULL
        );`,
		Columns: "id, parent_id, created_at",
	}}))

	on, err := foreignKeysEnabled(ctx, exec)
	require.NoError(t, err)
	assert.True(t, on, "foreign key enforcement was left disabled")

	_, err = exec.Exec(ctx,
		`INSERT INTO kids (id, parent_id, created_at) VALUES ('k1', 'missing', '2026-01-15T10:30:00Z');`)
	assert.Error(t, err, "orphan insert should be rejected once enforcement is restored")
}

// TestRebuildTables_RejectsColumnDrift is the guard the core store could only
// express as a test: if a later migration added a column that the replacement
// CREATE does not know about, copying would silently drop it. The rebuild must
// refuse before any data moves.
func TestRebuildTables_RejectsColumnDrift(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	mustExec(t, exec, `CREATE TABLE widgets (
        id         TEXT PRIMARY KEY,
        created_at TEXT NOT NULL,
        added_later TEXT NOT NULL DEFAULT ''
    );`)
	mustExec(t, exec, `INSERT INTO widgets (id, created_at, added_later) VALUES ('w1', '2026-01-15T10:30:00Z', 'keep me');`)

	err := RebuildTables(ctx, exec, []TableRebuild{{
		Table: "widgets",
		Create: `CREATE TABLE widgets_new (
            id         TEXT PRIMARY KEY,
            created_at TIMESTAMP NOT NULL
        );`,
		Columns: "id, created_at",
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not match live columns")

	// The original table is untouched, so a failed migration loses nothing.
	assert.EqualValues(t, 1, countRows(t, exec, "widgets"))
}

// TestRebuildTables_RejectsCopyListDrift catches the mirror-image mistake: the
// replacement CREATE names every column but the copy list forgets one, which
// would leave that column at its default for every existing row.
func TestRebuildTables_RejectsCopyListDrift(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	mustExec(t, exec, `CREATE TABLE widgets (
        id         TEXT PRIMARY KEY,
        note       TEXT NOT NULL DEFAULT '',
        created_at TEXT NOT NULL
    );`)

	err := RebuildTables(ctx, exec, []TableRebuild{{
		Table: "widgets",
		Create: `CREATE TABLE widgets_new (
            id         TEXT PRIMARY KEY,
            note       TEXT NOT NULL DEFAULT '',
            created_at TIMESTAMP NOT NULL
        );`,
		Columns: "id, created_at",
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copy list")
}

// TestRebuildTables_RejectsLostIndex catches an incomplete Indexes list, which
// would quietly drop a unique constraint along with the index.
func TestRebuildTables_RejectsLostIndex(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)

	mustExec(t, exec, `CREATE TABLE widgets (
        id         TEXT PRIMARY KEY,
        name       TEXT NOT NULL,
        created_at TEXT NOT NULL
    );`)
	mustExec(t, exec, `CREATE UNIQUE INDEX idx_widgets_name ON widgets (name);`)

	err := RebuildTables(ctx, exec, []TableRebuild{{
		Table: "widgets",
		Create: `CREATE TABLE widgets_new (
            id         TEXT PRIMARY KEY,
            name       TEXT NOT NULL,
            created_at TIMESTAMP NOT NULL
        );`,
		Columns: "id, name, created_at",
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index set changed")
}
