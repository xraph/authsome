package passkey

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the passkey plugin.
var PostgresMigrations = migrate.NewGroup("authsome-passkey", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the passkey plugin.
var SqliteMigrations = migrate.NewGroup("authsome-passkey", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_passkey_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_passkey_credentials (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES authsome_users(id),
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    credential_id    BYTEA NOT NULL,
    public_key       BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    transport        TEXT NOT NULL DEFAULT '',
    sign_count       INTEGER NOT NULL DEFAULT 0,
    aaguid           BYTEA,
    display_name     TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_cred_id
    ON authsome_passkey_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_user
    ON authsome_passkey_credentials (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_passkey_credentials;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_passkey_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_passkey_credentials (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES authsome_users(id),
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    credential_id    BLOB NOT NULL,
    public_key       BLOB NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    transport        TEXT NOT NULL DEFAULT '',
    sign_count       INTEGER NOT NULL DEFAULT 0,
    aaguid           BLOB,
    display_name     TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_cred_id
    ON authsome_passkey_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_user
    ON authsome_passkey_credentials (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_passkey_credentials;`)
				return err
			},
		},
	)
}
