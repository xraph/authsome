package oauth2provider

import (
	"context"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/internal/sqliteschema"
)

// The three OAuth2 tables declared their timestamp columns as TEXT, so a client,
// authorization code or device code written with an ordinary time.Now() could
// not be read back — CreateClient followed by GetClient failed outright. See the
// sqliteschema package doc for the full mechanism, and
// store/sqlite/migrations_timestamps.go for the core store's equivalent fix.
//
// last_polled_at keeps its DEFAULT ” rather than becoming NULL: the store reads
// it through a nullable path, and changing the default is a behaviour change
// that does not belong in a type fix.
var timestampRebuilds = []sqliteschema.TableRebuild{
	{
		Table: "authsome_oauth2_clients",
		Create: `CREATE TABLE authsome_oauth2_clients_new (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    name             TEXT NOT NULL,
    client_id        TEXT NOT NULL UNIQUE,
    client_secret    TEXT NOT NULL DEFAULT '',
    redirect_uris    TEXT NOT NULL DEFAULT '[]',
    scopes           TEXT NOT NULL DEFAULT '[]',
    grant_types      TEXT NOT NULL DEFAULT '[]',
    public           INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),

    -- Added by 20260824000030 (dynamic client registration).
    -- client_secret_expires_at was already declared TIMESTAMP there, so it is
    -- carried across unchanged.
    token_endpoint_auth_method TEXT NOT NULL DEFAULT '',
    registration_token_hash    TEXT NOT NULL DEFAULT '',
    dynamically_registered     BOOLEAN NOT NULL DEFAULT 0,
    client_secret_expires_at   TIMESTAMP,
    metadata                   TEXT NOT NULL DEFAULT '{}',
    -- Added by 20260824000001 (RFC 8707 resource indicators), which runs
    -- before this rebuild, so the live table already has it.
    resources                  TEXT NOT NULL DEFAULT '[]',
    -- Added by 20260824000041 (RFC 9449 DPoP), which also runs before this
    -- rebuild. Leave it out and the rebuild aborts on a column mismatch.
    dpop_mode                  TEXT NOT NULL DEFAULT ''
);`,
		Columns: "id, app_id, name, client_id, client_secret, redirect_uris, scopes, grant_types, public, created_at, updated_at, " +
			"token_endpoint_auth_method, registration_token_hash, dynamically_registered, client_secret_expires_at, metadata, resources, dpop_mode",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_clients_app
    ON authsome_oauth2_clients (app_id);`,
	},
	{
		Table: "authsome_oauth2_auth_codes",
		Create: `CREATE TABLE authsome_oauth2_auth_codes_new (
    id                    TEXT PRIMARY KEY,
    code                  TEXT NOT NULL UNIQUE,
    client_id             TEXT NOT NULL,
    user_id               TEXT NOT NULL REFERENCES authsome_users(id),
    app_id                TEXT NOT NULL REFERENCES authsome_apps(id),
    redirect_uri          TEXT NOT NULL DEFAULT '',
    scopes                TEXT NOT NULL DEFAULT '[]',
    code_challenge        TEXT NOT NULL DEFAULT '',
    code_challenge_method TEXT NOT NULL DEFAULT '',
    expires_at            TIMESTAMP NOT NULL,
    consumed              INTEGER NOT NULL DEFAULT 0,
    created_at            TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    -- Added by 20260824000001 (RFC 8707), which runs before this rebuild.
    resources             TEXT NOT NULL DEFAULT '[]'
);`,
		Columns: "id, code, client_id, user_id, app_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at, consumed, created_at, resources",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_auth_codes_code
    ON authsome_oauth2_auth_codes (code);`,
	},
	{
		Table: "authsome_oauth2_device_codes",
		Create: `CREATE TABLE authsome_oauth2_device_codes_new (
    id               TEXT PRIMARY KEY,
    device_code      TEXT NOT NULL UNIQUE,
    user_code        TEXT NOT NULL,
    client_id        TEXT NOT NULL,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    scopes           TEXT NOT NULL DEFAULT '[]',
    verification_uri TEXT NOT NULL DEFAULT '',
    expires_at       TIMESTAMP NOT NULL,
    interval         INTEGER NOT NULL DEFAULT 5,
    status           TEXT NOT NULL DEFAULT 'pending',
    user_id          TEXT DEFAULT '',
    last_polled_at   TIMESTAMP DEFAULT '',
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    -- Added by 20260824000001 (RFC 8707), which runs before this rebuild.
    resources        TEXT NOT NULL DEFAULT '[]'
);`,
		Columns: "id, device_code, user_code, client_id, app_id, scopes, verification_uri, expires_at, interval, status, user_id, last_polled_at, created_at, resources",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_device_codes_device_code
    ON authsome_oauth2_device_codes (device_code);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_device_codes_user_code
    ON authsome_oauth2_device_codes (user_code);`,
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
