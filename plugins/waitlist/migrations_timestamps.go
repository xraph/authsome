package waitlist

import (
	"context"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/internal/sqliteschema"
)

// authsome_waitlist_entries declared created_at and updated_at as TEXT, so any
// entry written with an ordinary time.Now() could not be read back. See the
// sqliteschema package doc for the full mechanism, and
// store/sqlite/migrations_timestamps.go for the core store's equivalent fix.
var timestampRebuilds = []sqliteschema.TableRebuild{
	{
		Table: "authsome_waitlist_entries",
		Create: `CREATE TABLE authsome_waitlist_entries_new (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    email       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    user_id     TEXT,
    ip_address  TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    UNIQUE(app_id, email)
);`,
		Columns: "id, app_id, email, name, status, user_id, ip_address, note, created_at, updated_at",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_waitlist_entries_app_status
    ON authsome_waitlist_entries (app_id, status);`,
	},
}

func init() {
	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "convert_text_timestamps_to_timestamp",
			Version: "20260824000050",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				return sqliteschema.RebuildTables(ctx, exec, timestampRebuilds)
			},
			// Forward-only: the TEXT declaration is the bug, so rebuilding back
			// to it has no value. Down is a documented no-op.
			Down: func(_ context.Context, _ migrate.Executor) error { return nil },
		},
	)
}
