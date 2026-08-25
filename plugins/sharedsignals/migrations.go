package sharedsignals

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
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

// fixEventDedupeIndexSQL replaces the received-events unique index with one
// keyed on (stream_id, jti, event_type) instead of just (stream_id, jti).
//
// RFC 8417 keys a SET's `events` object by event type URI, so a single
// delivery carries at most one event of a given type under one jti but may
// legitimately carry several different types. The two-column index made the
// second event of a multi-event SET collide with the dedupe row the first
// event had just inserted -- on the very first delivery, before any replay
// ever happened. This is syntax-compatible across Postgres and SQLite, so
// one string serves both migration groups.
//
// The prior migration has already shipped, so this is a NEW version that
// alters the index rather than an edit to the original -- an already-applied
// migration must not be rewritten.
const fixEventDedupeIndexSQL = `
DROP INDEX IF EXISTS idx_authsome_ssf_events_jti;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_events_jti_type
    ON authsome_ssf_received_events (stream_id, jti, event_type);
`

// revertEventDedupeIndexSQL is fixEventDedupeIndexSQL's Down: it restores
// the original two-column unique index. Any received-event rows written
// under the three-column index that share (stream_id, jti) across different
// event_type values will make the CREATE UNIQUE INDEX below fail, which is
// the correct outcome -- a downgrade cannot silently discard the rows that
// only the fixed key made possible to store.
const revertEventDedupeIndexSQL = `
DROP INDEX IF EXISTS idx_authsome_ssf_events_jti_type;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_events_jti
    ON authsome_ssf_received_events (stream_id, jti);
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

	MongoMigrations.MustRegister(&migrate.Migration{
		Name:    "create_sharedsignals_collections",
		Version: "20260824000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("sharedsignals: expected mongomigrate executor, got %T", exec)
			}
			for _, model := range []any{
				(*inboundStreamDoc)(nil), (*subjectLinkDoc)(nil),
				(*receivedEventDoc)(nil), (*signalDoc)(nil),
			} {
				if err := mexec.CreateCollection(ctx, model); err != nil {
					return err
				}
			}
			if err := mexec.CreateIndexes(ctx, colInboundStreams, []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "push_path_hash", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
			}); err != nil {
				return err
			}
			if err := mexec.CreateIndexes(ctx, colSubjectLinks, []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "app_id", Value: 1}, {Key: "env_id", Value: 1},
						{Key: "issuer", Value: 1}, {Key: "subject", Value: 1},
					},
					Options: options.Index().SetUnique(true),
				},
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
			}); err != nil {
				return err
			}
			// This unique index is the replay guard on mongo. Without it a
			// replayed SET revokes sessions a second time.
			if err := mexec.CreateIndexes(ctx, colReceivedEvents, []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "stream_id", Value: 1}, {Key: "jti", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{Keys: bson.D{{Key: "stream_id", Value: 1}, {Key: "received_at", Value: -1}}},
			}); err != nil {
				return err
			}
			return mexec.CreateIndexes(ctx, colSignals, []mongo.IndexModel{
				{Keys: bson.D{
					{Key: "app_id", Value: 1}, {Key: "user_id", Value: 1},
					{Key: "expires_at", Value: -1},
				}},
			})
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("sharedsignals: expected mongomigrate executor, got %T", exec)
			}
			for _, model := range []any{
				(*signalDoc)(nil), (*receivedEventDoc)(nil),
				(*subjectLinkDoc)(nil), (*inboundStreamDoc)(nil),
			} {
				if err := mexec.DropCollection(ctx, model); err != nil {
					return err
				}
			}
			return nil
		},
	})

	// Fixes the dedupe key from (stream_id, jti) to (stream_id, jti,
	// event_type). See fixEventDedupeIndexSQL for why. The prior migration
	// has already shipped, so this alters the index in a new version rather
	// than editing "create_sharedsignals_tables" in place.
	PostgresMigrations.MustRegister(&migrate.Migration{
		Name:    "fix_event_dedupe_key",
		Version: "20260824000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, fixEventDedupeIndexSQL)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, revertEventDedupeIndexSQL)
			return err
		},
	})

	SqliteMigrations.MustRegister(&migrate.Migration{
		Name:    "fix_event_dedupe_key",
		Version: "20260824000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, fixEventDedupeIndexSQL)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, revertEventDedupeIndexSQL)
			return err
		},
	})

	MongoMigrations.MustRegister(&migrate.Migration{
		Name:    "fix_event_dedupe_key",
		Version: "20260824000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("sharedsignals: expected mongomigrate executor, got %T", exec)
			}
			coll := mexec.DB().Collection(colReceivedEvents)
			// Mongo auto-names an index from its keys when no name is
			// given, so the original unique index from version 1 is
			// "stream_id_1_jti_1".
			if err := dropIndexIfExists(ctx, coll, "stream_id_1_jti_1"); err != nil {
				return err
			}
			return mexec.CreateIndexes(ctx, colReceivedEvents, []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "stream_id", Value: 1}, {Key: "jti", Value: 1},
						{Key: "event_type", Value: 1},
					},
					Options: options.Index().SetUnique(true),
				},
			})
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("sharedsignals: expected mongomigrate executor, got %T", exec)
			}
			coll := mexec.DB().Collection(colReceivedEvents)
			if err := dropIndexIfExists(ctx, coll, "stream_id_1_jti_1_event_type_1"); err != nil {
				return err
			}
			return mexec.CreateIndexes(ctx, colReceivedEvents, []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "stream_id", Value: 1}, {Key: "jti", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
			})
		},
	})
}

// dropIndexIfExists drops a Mongo index by name, tolerating the case where
// it is already gone (IndexNotFound, server error code 27) so the migration
// stays safe to reason about even if it is ever re-applied against a
// database that was partially migrated by hand.
func dropIndexIfExists(ctx context.Context, coll *mongo.Collection, name string) error {
	err := coll.Indexes().DropOne(ctx, name)
	if err == nil {
		return nil
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 27 { // IndexNotFound
		return nil
	}
	return err
}
