package sso

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the SSO plugin.
var PostgresMigrations = migrate.NewGroup("authsome-sso", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the SSO plugin.
var SqliteMigrations = migrate.NewGroup("authsome-sso", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_sso_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_sso_connections (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    org_id       TEXT NOT NULL DEFAULT '',
    provider     TEXT NOT NULL,
    protocol     TEXT NOT NULL,
    domain       TEXT NOT NULL,
    metadata_url TEXT NOT NULL DEFAULT '',
    client_id    TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL DEFAULT '',
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_app
    ON authsome_sso_connections (app_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_provider
    ON authsome_sso_connections (app_id, provider) WHERE active = TRUE;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_sso_connections;`)
				return err
			},
		},
	)

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_client_secret",
			Version: "20240201000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS client_secret TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS client_secret;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_sso_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_sso_connections (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    org_id       TEXT NOT NULL DEFAULT '',
    provider     TEXT NOT NULL,
    protocol     TEXT NOT NULL,
    domain       TEXT NOT NULL,
    metadata_url TEXT NOT NULL DEFAULT '',
    client_id    TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL DEFAULT '',
    active       INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_app
    ON authsome_sso_connections (app_id, created_at);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_provider
    ON authsome_sso_connections (app_id, provider);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_sso_connections;`)
				return err
			},
		},
	)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_client_secret",
			Version: "20240201000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections ADD COLUMN client_secret TEXT NOT NULL DEFAULT '';`)
				return err
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				// SQLite does not support DROP COLUMN in older versions; best-effort.
				return nil
			},
		},
	)
}
