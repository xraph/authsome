// Package sqliteschema provides the SQLite table-rebuild procedure used by
// migrations that must change a column's declared type.
//
// SQLite has no ALTER COLUMN TYPE, so changing a declared type means rebuilding
// the table: create a replacement, copy every row, drop the original, rename the
// replacement into place, recreate the indexes.
//
// # Why a declared type matters in a dynamically typed database
//
// SQLite stores values, not column types, so a declared type looks cosmetic. It
// is not. The modernc.org/sqlite driver decides whether to convert a TEXT value
// back into a time.Time by looking at the column's DECLARED type: only DATE,
// DATETIME and TIMESTAMP get the conversion (see rows.go, ColumnTypeDatabaseTypeName).
// A column declared TEXT hands the raw string to the caller.
//
// That is a problem because of how the same driver WRITES a time.Time. Unless
// the DSN carries _time_format, conn.formatTime falls back to time.Time.String(),
// which appends the monotonic clock reading:
//
//	2026-08-24 18:13:13.095282 -0500 CDT m=+0.241018501
//
// modernc knows how to strip that " m=+…" suffix on read, but only on the
// DATE/DATETIME/TIMESTAMP path. For a TEXT column the raw string reaches grove's
// scan/convert.go, whose parseTimeString layouts do not tolerate the suffix, and
// the read fails:
//
//	sql: Scan error on column index 10, name "created_at":
//	scan: cannot parse "… -0500 CDT m=+0.016649834" as time.Time
//
// So any row written with an ordinary time.Now() into a TEXT-declared timestamp
// column cannot be read back. Declaring the column TIMESTAMP fixes both halves:
// the driver strips the suffix and returns a time.Time directly.
//
// The core store hit this first and fixed it in
// store/sqlite/migrations_timestamps.go (migration 20260601000002). This package
// generalises that procedure for the plugin schemas, which — unlike the core
// schema — declare foreign keys.
//
// # Foreign keys
//
// The rebuild drops the original table. If foreign key enforcement is on at that
// moment, SQLite treats the drop as an implicit DELETE FROM and fires every
// ON DELETE CASCADE pointing at it. Rebuilding authsome_scim_configs that way
// would silently empty authsome_scim_tokens and authsome_scim_provision_logs.
//
// So the procedure follows https://sqlite.org/lang_altertable.html#otheralter and
// turns foreign keys off for the duration. Turning them off also settles the
// other half of the rename semantics: ALTER TABLE … RENAME rewrites REFERENCES
// clauses in other tables only when foreign keys are ON, so with them OFF each
// replacement table must spell out its own REFERENCES clauses using the final
// table names — which is exactly what copying the original CREATE and changing
// only the timestamp types gives you.
package sqliteschema

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/xraph/grove/migrate"
)

// TableRebuild describes the replacement of one table.
type TableRebuild struct {
	// Table is the existing table name.
	Table string
	// Create is a CREATE TABLE <Table>_new (…) statement: the original
	// definition with the timestamp columns declared TIMESTAMP, every other
	// column and every REFERENCES clause left exactly as it was.
	Create string
	// Columns is the comma-separated column list used as both the INSERT
	// target and the SELECT source, so the copy is order-independent.
	Columns string
	// Indexes holds the CREATE INDEX statements recreated after the rename.
	// Indexes implied by the CREATE (PRIMARY KEY, inline UNIQUE) come back on
	// their own and must not be listed here.
	Indexes string
}

// RebuildTables runs the rebuild procedure for every table in rebuilds, with
// foreign key enforcement disabled for the duration and restored afterwards.
//
// Each rebuild is self-checked: the replacement's column set must match both the
// live table and the copy list, and the table's index set must be unchanged once
// the rename is done. A mismatch aborts before any data moves rather than
// silently dropping a column added by a later migration.
func RebuildTables(ctx context.Context, exec migrate.Executor, rebuilds []TableRebuild) error {
	restore, err := disableForeignKeys(ctx, exec)
	if err != nil {
		return err
	}
	defer restore()

	for _, r := range rebuilds {
		if err := rebuildTable(ctx, exec, r); err != nil {
			return fmt.Errorf("rebuild %s: %w", r.Table, err)
		}
	}

	return checkForeignKeys(ctx, exec)
}

// disableForeignKeys turns foreign key enforcement off and returns a function
// that puts the previous setting back.
//
// PRAGMA foreign_keys is per-connection and grove hands migrations a pooled
// *sql.DB, so there is no way to pin these statements to one connection. A
// sequential migration run reuses a single idle connection in practice, but
// "in practice" is not good enough when the failure mode is a silent cascade
// delete — so the setting is read back and a still-enabled connection aborts
// the migration instead of proceeding.
func disableForeignKeys(ctx context.Context, exec migrate.Executor) (func(), error) {
	was, err := foreignKeysEnabled(ctx, exec)
	if err != nil {
		return nil, err
	}
	if _, execErr := exec.Exec(ctx, "PRAGMA foreign_keys = OFF;"); execErr != nil {
		return nil, fmt.Errorf("disable foreign keys: %w", execErr)
	}
	off, err := foreignKeysEnabled(ctx, exec)
	if err != nil {
		return nil, err
	}
	if off {
		return nil, fmt.Errorf(
			"foreign keys are still enabled after PRAGMA foreign_keys = OFF; " +
				"dropping a parent table now would cascade-delete its children, so the rebuild is aborted")
	}

	// Best-effort restore: it runs in a defer, so the rebuild's own error is
	// the one worth returning, and foreign_key_check has already run by then.
	return func() {
		if was {
			_, _ = exec.Exec(ctx, "PRAGMA foreign_keys = ON;") //nolint:errcheck // see above
		}
	}, nil
}

// checkForeignKeys fails if the rebuilt schema left any dangling reference.
func checkForeignKeys(ctx context.Context, exec migrate.Executor) error {
	rows, err := exec.Query(ctx, "PRAGMA foreign_key_check;")
	if err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rows.Close()

	var violations []string
	for rows.Next() {
		var (
			table, parent string
			rowid, fkid   any
		)
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return fmt.Errorf("foreign_key_check: scan: %w", err)
		}
		violations = append(violations, table+" -> "+parent)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	if len(violations) > 0 {
		return fmt.Errorf("rebuild left %d foreign key violation(s): %s",
			len(violations), strings.Join(violations, ", "))
	}
	return nil
}

// rebuildTable performs create → verify → copy → drop → rename → reindex for a
// single table.
func rebuildTable(ctx context.Context, exec migrate.Executor, r TableRebuild) error {
	live, err := tableColumns(ctx, exec, r.Table)
	if err != nil {
		return err
	}
	if len(live) == 0 {
		return fmt.Errorf("table does not exist")
	}
	indexesBefore, err := tableIndexes(ctx, exec, r.Table)
	if err != nil {
		return err
	}

	tmp := r.Table + "_new"
	if _, execErr := exec.Exec(ctx, r.Create); execErr != nil {
		return fmt.Errorf("create %s: %w", tmp, execErr)
	}

	replacement, err := tableColumns(ctx, exec, tmp)
	if err != nil {
		return err
	}
	copyCols := splitColumns(r.Columns)
	if !sameStrings(live, replacement) {
		return fmt.Errorf("replacement columns %v do not match live columns %v", replacement, live)
	}
	if !sameStrings(live, copyCols) {
		return fmt.Errorf("copy list %v does not match live columns %v", copyCols, live)
	}

	if _, execErr := exec.Exec(ctx, fmt.Sprintf(
		"INSERT INTO %s (%s) SELECT %s FROM %s;", tmp, r.Columns, r.Columns, r.Table,
	)); execErr != nil {
		return fmt.Errorf("copy rows into %s: %w", tmp, execErr)
	}
	if _, execErr := exec.Exec(ctx, fmt.Sprintf("DROP TABLE %s;", r.Table)); execErr != nil {
		return fmt.Errorf("drop old %s: %w", r.Table, execErr)
	}
	if _, execErr := exec.Exec(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tmp, r.Table)); execErr != nil {
		return fmt.Errorf("rename %s: %w", tmp, execErr)
	}
	if r.Indexes != "" {
		if _, execErr := exec.Exec(ctx, r.Indexes); execErr != nil {
			return fmt.Errorf("recreate indexes: %w", execErr)
		}
	}

	indexesAfter, err := tableIndexes(ctx, exec, r.Table)
	if err != nil {
		return err
	}
	if !sameStrings(indexesBefore, indexesAfter) {
		return fmt.Errorf("index set changed: had %v, now %v — a lost unique index is a lost constraint",
			indexesBefore, indexesAfter)
	}
	return nil
}

// tableColumns returns the sorted column names of table, or an empty slice if
// the table does not exist.
func tableColumns(ctx context.Context, exec migrate.Executor, table string) ([]string, error) {
	rows, err := exec.Query(ctx, "PRAGMA table_info("+table+");")
	if err != nil {
		return nil, fmt.Errorf("table_info(%s): %w", table, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var (
			cid, notnull, pk int64
			name, typ        string
			dflt             any
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("table_info(%s): scan: %w", table, err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("table_info(%s): %w", table, err)
	}
	sort.Strings(names)
	return names, nil
}

// tableIndexes returns the sorted names of every index on table, including the
// ones SQLite creates for PRIMARY KEY and inline UNIQUE constraints.
func tableIndexes(ctx context.Context, exec migrate.Executor, table string) ([]string, error) {
	rows, err := exec.Query(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ?;", table)
	if err != nil {
		return nil, fmt.Errorf("list indexes of %s: %w", table, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("list indexes of %s: scan: %w", table, err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list indexes of %s: %w", table, err)
	}
	sort.Strings(names)
	return names, nil
}

func foreignKeysEnabled(ctx context.Context, exec migrate.Executor) (bool, error) {
	rows, err := exec.Query(ctx, "PRAGMA foreign_keys;")
	if err != nil {
		return false, fmt.Errorf("read foreign_keys pragma: %w", err)
	}
	defer rows.Close()

	var on int64
	if rows.Next() {
		if err := rows.Scan(&on); err != nil {
			return false, fmt.Errorf("read foreign_keys pragma: scan: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("read foreign_keys pragma: %w", err)
	}
	return on != 0, nil
}

func splitColumns(cols string) []string {
	parts := strings.Split(cols, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
