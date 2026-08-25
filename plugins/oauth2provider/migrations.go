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

	// Recreate the oauth2 app_id foreign keys ON DELETE CASCADE so deleting an
	// app removes its oauth2 clients, authorization codes and device codes
	// along with the core children (see the authsome group's
	// "cascade_app_id_foreign_keys" migration). PostgreSQL only — the sqlite
	// schema does not enforce these foreign keys, and the core DeleteApp does
	// not reach the plugin's separate sqlite/memory stores.
	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "cascade_oauth2_app_id_foreign_keys",
			Version: "20260615000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients DROP CONSTRAINT IF EXISTS authsome_oauth2_clients_app_id_fkey;
ALTER TABLE authsome_oauth2_clients ADD CONSTRAINT authsome_oauth2_clients_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES authsome_apps(id) ON DELETE CASCADE;

ALTER TABLE authsome_oauth2_auth_codes DROP CONSTRAINT IF EXISTS authsome_oauth2_auth_codes_app_id_fkey;
ALTER TABLE authsome_oauth2_auth_codes ADD CONSTRAINT authsome_oauth2_auth_codes_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES authsome_apps(id) ON DELETE CASCADE;

ALTER TABLE authsome_oauth2_device_codes DROP CONSTRAINT IF EXISTS authsome_oauth2_device_codes_app_id_fkey;
ALTER TABLE authsome_oauth2_device_codes ADD CONSTRAINT authsome_oauth2_device_codes_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES authsome_apps(id) ON DELETE CASCADE;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients DROP CONSTRAINT IF EXISTS authsome_oauth2_clients_app_id_fkey;
ALTER TABLE authsome_oauth2_clients ADD CONSTRAINT authsome_oauth2_clients_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES authsome_apps(id);

ALTER TABLE authsome_oauth2_auth_codes DROP CONSTRAINT IF EXISTS authsome_oauth2_auth_codes_app_id_fkey;
ALTER TABLE authsome_oauth2_auth_codes ADD CONSTRAINT authsome_oauth2_auth_codes_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES authsome_apps(id);

ALTER TABLE authsome_oauth2_device_codes DROP CONSTRAINT IF EXISTS authsome_oauth2_device_codes_app_id_fkey;
ALTER TABLE authsome_oauth2_device_codes ADD CONSTRAINT authsome_oauth2_device_codes_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES authsome_apps(id);
`)
				return err
			},
		},
	)

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_dynamic_registration_columns",
			Version: "20260824000030",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients
    ADD COLUMN IF NOT EXISTS token_endpoint_auth_method TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS registration_token_hash    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS dynamically_registered     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS client_secret_expires_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS metadata                   JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_clients_dynamic
    ON authsome_oauth2_clients (app_id)
    WHERE dynamically_registered;

UPDATE authsome_oauth2_clients
   SET token_endpoint_auth_method = CASE WHEN public THEN 'none' ELSE 'client_secret_basic' END
 WHERE token_endpoint_auth_method = '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_oauth2_clients_dynamic;
ALTER TABLE authsome_oauth2_clients
    DROP COLUMN IF EXISTS token_endpoint_auth_method,
    DROP COLUMN IF EXISTS registration_token_hash,
    DROP COLUMN IF EXISTS dynamically_registered,
    DROP COLUMN IF EXISTS client_secret_expires_at,
    DROP COLUMN IF EXISTS metadata;
`)
				return err
			},
		},
	)

	// Add the RFC 8707 resource indicator allowlist to clients, and the
	// per-grant resource audience to authorization codes and device codes.
	// Every existing row defaults to an empty array, so existing clients,
	// codes and device codes keep working exactly as before.
	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_oauth2_resources",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients
    ADD COLUMN IF NOT EXISTS resources JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE authsome_oauth2_auth_codes
    ADD COLUMN IF NOT EXISTS resources JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE authsome_oauth2_device_codes
    ADD COLUMN IF NOT EXISTS resources JSONB NOT NULL DEFAULT '[]'::jsonb;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients DROP COLUMN IF EXISTS resources;
ALTER TABLE authsome_oauth2_auth_codes DROP COLUMN IF EXISTS resources;
ALTER TABLE authsome_oauth2_device_codes DROP COLUMN IF EXISTS resources;
`)
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

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_dynamic_registration_columns",
			Version: "20260824000030",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients ADD COLUMN token_endpoint_auth_method TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_oauth2_clients ADD COLUMN registration_token_hash    TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_oauth2_clients ADD COLUMN dynamically_registered     BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE authsome_oauth2_clients ADD COLUMN client_secret_expires_at   TIMESTAMP;
ALTER TABLE authsome_oauth2_clients ADD COLUMN metadata                   TEXT NOT NULL DEFAULT '{}';

UPDATE authsome_oauth2_clients
   SET token_endpoint_auth_method = CASE WHEN public = 1 THEN 'none' ELSE 'client_secret_basic' END
 WHERE token_endpoint_auth_method = '';
`)
				return err
			},
			// Older SQLite cannot drop columns, so Down is a no-op. Rolling
			// back this migration leaves the columns in place, harmlessly.
			Down: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
		},
	)

	// The SQLite counterpart of add_oauth2_resources. SQLite takes one
	// ADD COLUMN per statement, so these run as three separate execs.
	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_oauth2_resources",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				for _, stmt := range []string{
					`ALTER TABLE authsome_oauth2_clients ADD COLUMN resources TEXT NOT NULL DEFAULT '[]';`,
					`ALTER TABLE authsome_oauth2_auth_codes ADD COLUMN resources TEXT NOT NULL DEFAULT '[]';`,
					`ALTER TABLE authsome_oauth2_device_codes ADD COLUMN resources TEXT NOT NULL DEFAULT '[]';`,
				} {
					if _, err := exec.Exec(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			},
			// Older SQLite cannot drop columns, so Down is a no-op, matching
			// the migration above.
			Down: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
		},
	)
}
