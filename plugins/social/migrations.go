package social

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the social plugin.
var PostgresMigrations = migrate.NewGroup("authsome-social", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the social plugin.
var SqliteMigrations = migrate.NewGroup("authsome-social", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_social_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth_connections (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id          TEXT NOT NULL REFERENCES authsome_users(id),
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email            TEXT NOT NULL DEFAULT '',
    access_token     TEXT NOT NULL DEFAULT '',
    refresh_token    TEXT NOT NULL DEFAULT '',
    expires_at       TIMESTAMPTZ,
    metadata         JSONB DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_oauth_connections_provider
    ON authsome_oauth_connections (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_authsome_oauth_connections_user
    ON authsome_oauth_connections (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_oauth_connections;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_social_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth_connections (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id          TEXT NOT NULL REFERENCES authsome_users(id),
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email            TEXT NOT NULL DEFAULT '',
    access_token     TEXT NOT NULL DEFAULT '',
    refresh_token    TEXT NOT NULL DEFAULT '',
    expires_at       TEXT,
    metadata         TEXT DEFAULT '{}',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_oauth_connections_provider
    ON authsome_oauth_connections (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_authsome_oauth_connections_user
    ON authsome_oauth_connections (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_oauth_connections;`)
				return err
			},
		},
	)
}
