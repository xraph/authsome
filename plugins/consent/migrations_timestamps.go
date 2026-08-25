package consent

import (
	"context"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/internal/sqliteschema"
)

// authsome_consents declared granted_at, revoked_at, created_at and updated_at
// as TEXT, so any consent written with an ordinary time.Now() could not be read
// back. See the sqliteschema package doc for the full mechanism, and
// store/sqlite/migrations_timestamps.go for the core store's equivalent fix.
var timestampRebuilds = []sqliteschema.TableRebuild{
	{
		Table: "authsome_consents",
		Create: `CREATE TABLE authsome_consents_new (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES authsome_users(id),
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    purpose     TEXT NOT NULL,
    granted     INTEGER NOT NULL DEFAULT 1,
    version     TEXT NOT NULL DEFAULT '',
    ip_address  TEXT NOT NULL DEFAULT '',
    granted_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    revoked_at  TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		Columns: "id, user_id, app_id, purpose, granted, version, ip_address, granted_at, revoked_at, created_at, updated_at",
		Indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_consents_user_app_purpose
    ON authsome_consents (user_id, app_id, purpose);

CREATE INDEX IF NOT EXISTS idx_authsome_consents_user
    ON authsome_consents (user_id);`,
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
