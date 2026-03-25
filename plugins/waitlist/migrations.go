package waitlist

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the waitlist plugin.
var PostgresMigrations = migrate.NewGroup("authsome-waitlist", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the waitlist plugin.
var SqliteMigrations = migrate.NewGroup("authsome-waitlist", migrate.DependsOn("authsome"))

// MongoMigrations is the MongoDB migration group for the waitlist plugin.
// MongoDB is schemaless so no actual migrations are needed.
var MongoMigrations = migrate.NewGroup("authsome-waitlist", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_waitlist_tables",
			Version: "20240601000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_waitlist_entries (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    email       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    user_id     TEXT,
    ip_address  TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, email)
);

CREATE INDEX IF NOT EXISTS idx_authsome_waitlist_entries_app_status
    ON authsome_waitlist_entries (app_id, status);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_waitlist_entries;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_waitlist_tables",
			Version: "20240601000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_waitlist_entries (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    email       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    user_id     TEXT,
    ip_address  TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(app_id, email)
);

CREATE INDEX IF NOT EXISTS idx_authsome_waitlist_entries_app_status
    ON authsome_waitlist_entries (app_id, status);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_waitlist_entries;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// MongoDB migrations (no-op — schemaless)
	// ──────────────────────────────────────────────────

	MongoMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_waitlist_collections",
			Version: "20240601000001",
			Up: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
		},
	)
}
