package oauth2provider

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the OAuth2 provider plugin.
var PostgresMigrations = migrate.NewGroup("authsome-oauth2", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the OAuth2 provider plugin.
var SqliteMigrations = migrate.NewGroup("authsome-oauth2", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_oauth2_tables",
			Version: "20240301000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth2_clients (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    name             TEXT NOT NULL,
    client_id        TEXT NOT NULL UNIQUE,
    client_secret    TEXT NOT NULL DEFAULT '',
    redirect_uris    JSONB NOT NULL DEFAULT '[]',
    scopes           JSONB NOT NULL DEFAULT '[]',
    grant_types      JSONB NOT NULL DEFAULT '[]',
    public           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_clients_app
    ON authsome_oauth2_clients (app_id);

CREATE TABLE IF NOT EXISTS authsome_oauth2_auth_codes (
    id                    TEXT PRIMARY KEY,
    code                  TEXT NOT NULL UNIQUE,
    client_id             TEXT NOT NULL,
    user_id               TEXT NOT NULL REFERENCES authsome_users(id),
    app_id                TEXT NOT NULL REFERENCES authsome_apps(id),
    redirect_uri          TEXT NOT NULL DEFAULT '',
    scopes                JSONB NOT NULL DEFAULT '[]',
    code_challenge        TEXT NOT NULL DEFAULT '',
    code_challenge_method TEXT NOT NULL DEFAULT '',
    expires_at            TIMESTAMPTZ NOT NULL,
    consumed              BOOLEAN NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_auth_codes_code
    ON authsome_oauth2_auth_codes (code);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_oauth2_auth_codes;
DROP TABLE IF EXISTS authsome_oauth2_clients;
`)
				return err
			},
		},
	)

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_device_codes_table",
			Version: "20240301000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth2_device_codes (
    id               TEXT PRIMARY KEY,
    device_code      TEXT NOT NULL UNIQUE,
    user_code        TEXT NOT NULL,
    client_id        TEXT NOT NULL,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    scopes           JSONB NOT NULL DEFAULT '[]',
    verification_uri TEXT NOT NULL DEFAULT '',
    expires_at       TIMESTAMPTZ NOT NULL,
    interval         INTEGER NOT NULL DEFAULT 5,
    status           TEXT NOT NULL DEFAULT 'pending',
    user_id          TEXT DEFAULT '',
    last_polled_at   TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_device_codes_device_code
    ON authsome_oauth2_device_codes (device_code);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_device_codes_user_code
    ON authsome_oauth2_device_codes (user_code)
    WHERE status = 'pending';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_oauth2_device_codes;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_oauth2_tables",
			Version: "20240301000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth2_clients (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    name             TEXT NOT NULL,
    client_id        TEXT NOT NULL UNIQUE,
    client_secret    TEXT NOT NULL DEFAULT '',
    redirect_uris    TEXT NOT NULL DEFAULT '[]',
    scopes           TEXT NOT NULL DEFAULT '[]',
    grant_types      TEXT NOT NULL DEFAULT '[]',
    public           INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_clients_app
    ON authsome_oauth2_clients (app_id);

CREATE TABLE IF NOT EXISTS authsome_oauth2_auth_codes (
    id                    TEXT PRIMARY KEY,
    code                  TEXT NOT NULL UNIQUE,
    client_id             TEXT NOT NULL,
    user_id               TEXT NOT NULL REFERENCES authsome_users(id),
    app_id                TEXT NOT NULL REFERENCES authsome_apps(id),
    redirect_uri          TEXT NOT NULL DEFAULT '',
    scopes                TEXT NOT NULL DEFAULT '[]',
    code_challenge        TEXT NOT NULL DEFAULT '',
    code_challenge_method TEXT NOT NULL DEFAULT '',
    expires_at            TEXT NOT NULL,
    consumed              INTEGER NOT NULL DEFAULT 0,
    created_at            TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_auth_codes_code
    ON authsome_oauth2_auth_codes (code);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_oauth2_auth_codes;
DROP TABLE IF EXISTS authsome_oauth2_clients;
`)
				return err
			},
		},
	)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_device_codes_table",
			Version: "20240301000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth2_device_codes (
    id               TEXT PRIMARY KEY,
    device_code      TEXT NOT NULL UNIQUE,
    user_code        TEXT NOT NULL,
    client_id        TEXT NOT NULL,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    scopes           TEXT NOT NULL DEFAULT '[]',
    verification_uri TEXT NOT NULL DEFAULT '',
    expires_at       TEXT NOT NULL,
    interval         INTEGER NOT NULL DEFAULT 5,
    status           TEXT NOT NULL DEFAULT 'pending',
    user_id          TEXT DEFAULT '',
    last_polled_at   TEXT DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_device_codes_device_code
    ON authsome_oauth2_device_codes (device_code);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_device_codes_user_code
    ON authsome_oauth2_device_codes (user_code);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_oauth2_device_codes;`)
				return err
			},
		},
	)
}
