package sharedsignals

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// Migration groups, one per driver. All depend on the core authsome group
// because the stream tables reference authsome_apps.
var (
	PostgresMigrations = migrate.NewGroup("authsome-sharedsignals", migrate.DependsOn("authsome"))
	SqliteMigrations   = migrate.NewGroup("authsome-sharedsignals", migrate.DependsOn("authsome"))
	MongoMigrations    = migrate.NewGroup("authsome-sharedsignals", migrate.DependsOn("authsome"))
)

// Mongo collection names.
const (
	colInboundStreams = "authsome_ssf_inbound_streams"
	colSubjectLinks   = "authsome_ssf_subject_links"
	colReceivedEvents = "authsome_ssf_received_events"
	colSignals        = "authsome_ssf_signals"
)

const pgSchema = `
CREATE TABLE IF NOT EXISTS authsome_ssf_inbound_streams (
    id                      TEXT PRIMARY KEY,
    app_id                  TEXT NOT NULL REFERENCES authsome_apps(id),
    env_id                  TEXT NOT NULL DEFAULT '',
    name                    TEXT NOT NULL DEFAULT '',
    issuer                  TEXT NOT NULL,
    audience                TEXT NOT NULL,
    jwks_uri                TEXT NOT NULL,
    push_path_hash          TEXT NOT NULL,
    push_token_hash         TEXT NOT NULL,
    allowed_event_types     TEXT NOT NULL DEFAULT '[]',
    allowed_subject_formats TEXT NOT NULL DEFAULT '[]',
    verified_domains        TEXT NOT NULL DEFAULT '[]',
    action_overrides        TEXT NOT NULL DEFAULT '{}',
    enforcement_mode        TEXT NOT NULL DEFAULT 'enforce',
    status                  TEXT NOT NULL DEFAULT 'enabled',
    max_actions_per_hour    INTEGER NOT NULL DEFAULT 100,
    pending_verify_state    TEXT NOT NULL DEFAULT '',
    last_verified_at        TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The push path is how an inbound request finds its stream, so the lookup
-- must be unique and indexed.
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_streams_push_path
    ON authsome_ssf_inbound_streams (push_path_hash);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_streams_app
    ON authsome_ssf_inbound_streams (app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_subject_links (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    env_id       TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL,
    subject      TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'sso',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_links_tuple
    ON authsome_ssf_subject_links (app_id, env_id, issuer, subject);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_links_user
    ON authsome_ssf_subject_links (user_id);

CREATE TABLE IF NOT EXISTS authsome_ssf_received_events (
    id               TEXT PRIMARY KEY,
    stream_id        TEXT NOT NULL,
    jti              TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    subject_json     TEXT NOT NULL DEFAULT '',
    resolved_user_id TEXT NOT NULL DEFAULT '',
    outcome          TEXT NOT NULL DEFAULT 'pending',
    action_taken     TEXT NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- This unique constraint IS the replay guard. Without it a replayed SET
-- revokes sessions twice.
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_events_jti
    ON authsome_ssf_received_events (stream_id, jti);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_events_stream_time
    ON authsome_ssf_received_events (stream_id, received_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_signals (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    env_id     TEXT NOT NULL DEFAULT '',
    user_id    TEXT NOT NULL,
    stream_id  TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL,
    severity   INTEGER NOT NULL DEFAULT 0,
    reason     TEXT NOT NULL DEFAULT '',
    event_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_ssf_signals_lookup
    ON authsome_ssf_signals (app_id, user_id, expires_at DESC);
`

// sqliteSchema is the same shape with SQLite's type names. TIMESTAMPTZ and
// NOW() do not exist there.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS authsome_ssf_inbound_streams (
    id                      TEXT PRIMARY KEY,
    app_id                  TEXT NOT NULL,
    env_id                  TEXT NOT NULL DEFAULT '',
    name                    TEXT NOT NULL DEFAULT '',
    issuer                  TEXT NOT NULL,
    audience                TEXT NOT NULL,
    jwks_uri                TEXT NOT NULL,
    push_path_hash          TEXT NOT NULL,
    push_token_hash         TEXT NOT NULL,
    allowed_event_types     TEXT NOT NULL DEFAULT '[]',
    allowed_subject_formats TEXT NOT NULL DEFAULT '[]',
    verified_domains        TEXT NOT NULL DEFAULT '[]',
    action_overrides        TEXT NOT NULL DEFAULT '{}',
    enforcement_mode        TEXT NOT NULL DEFAULT 'enforce',
    status                  TEXT NOT NULL DEFAULT 'enabled',
    max_actions_per_hour    INTEGER NOT NULL DEFAULT 100,
    pending_verify_state    TEXT NOT NULL DEFAULT '',
    last_verified_at        DATETIME,
    created_at              DATETIME NOT NULL,
    updated_at              DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_streams_push_path
    ON authsome_ssf_inbound_streams (push_path_hash);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_streams_app
    ON authsome_ssf_inbound_streams (app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_subject_links (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    env_id       TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL,
    subject      TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'sso',
    created_at   DATETIME NOT NULL,
    last_seen_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_links_tuple
    ON authsome_ssf_subject_links (app_id, env_id, issuer, subject);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_links_user
    ON authsome_ssf_subject_links (user_id);

CREATE TABLE IF NOT EXISTS authsome_ssf_received_events (
    id               TEXT PRIMARY KEY,
    stream_id        TEXT NOT NULL,
    jti              TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    subject_json     TEXT NOT NULL DEFAULT '',
    resolved_user_id TEXT NOT NULL DEFAULT '',
    outcome          TEXT NOT NULL DEFAULT 'pending',
    action_taken     TEXT NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    received_at      DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_events_jti
    ON authsome_ssf_received_events (stream_id, jti);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_events_stream_time
    ON authsome_ssf_received_events (stream_id, received_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_signals (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    env_id     TEXT NOT NULL DEFAULT '',
    user_id    TEXT NOT NULL,
    stream_id  TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL,
    severity   INTEGER NOT NULL DEFAULT 0,
    reason     TEXT NOT NULL DEFAULT '',
    event_at   DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_authsome_ssf_signals_lookup
    ON authsome_ssf_signals (app_id, user_id, expires_at DESC);
`

const dropSchema = `
DROP TABLE IF EXISTS authsome_ssf_signals;
DROP TABLE IF EXISTS authsome_ssf_received_events;
DROP TABLE IF EXISTS authsome_ssf_subject_links;
DROP TABLE IF EXISTS authsome_ssf_inbound_streams;
`

func init() {
	PostgresMigrations.MustRegister(&migrate.Migration{
		Name:    "create_sharedsignals_tables",
		Version: "20260824000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, pgSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropSchema)
			return err
		},
	})

	SqliteMigrations.MustRegister(&migrate.Migration{
		Name:    "create_sharedsignals_tables",
		Version: "20260824000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, sqliteSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropSchema)
			return err
		},
	})

}
