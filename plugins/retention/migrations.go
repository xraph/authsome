package retention

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
// because both tables live in the same schema and the core group has to
// create it first. There is deliberately no foreign key from app_id to
// authsome_apps: see the design note for why the outbox does not carry
// one.
var (
	PostgresMigrations = migrate.NewGroup("authsome-retention", migrate.DependsOn("authsome"))
	SqliteMigrations   = migrate.NewGroup("authsome-retention", migrate.DependsOn("authsome"))
	MongoMigrations    = migrate.NewGroup("authsome-retention", migrate.DependsOn("authsome"))
)

// Mongo collection names.
const (
	colContactRef = "authsome_retention_contact_ref"
	colOutbox     = "authsome_retention_outbox"
)

const pgSchema = `
CREATE TABLE IF NOT EXISTS authsome_retention_contact_ref (
    id                 TEXT PRIMARY KEY,
    app_id             TEXT NOT NULL,
    env_id             TEXT NOT NULL DEFAULT '',
    user_id            TEXT NOT NULL,
    provider           TEXT NOT NULL,
    remote_object_type TEXT NOT NULL DEFAULT '',
    remote_id          TEXT NOT NULL,
    synced_at          TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_retention_ref
    ON authsome_retention_contact_ref (app_id, env_id, user_id, provider);

CREATE TABLE IF NOT EXISTS authsome_retention_outbox (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL,
    env_id           TEXT NOT NULL DEFAULT '',
    user_id          TEXT NOT NULL,
    provider         TEXT NOT NULL,
    kind             TEXT NOT NULL,
    payload          TEXT NOT NULL DEFAULT '{}',
    idempotency_key  TEXT NOT NULL DEFAULT '',
    state            TEXT NOT NULL DEFAULT 'pending',
    attempts         INTEGER NOT NULL DEFAULT 0,
    next_attempt_at  TIMESTAMPTZ NOT NULL,
    in_flight_until  TIMESTAMPTZ,
    last_error       TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_retention_outbox_due
    ON authsome_retention_outbox (state, next_attempt_at);
-- The idempotency key is what makes Enqueue safe to call twice for the same
-- signup/login hook firing again; empty keys are excluded so unkeyed jobs
-- (there are none today, but the column allows it) never collide with each
-- other.
CREATE UNIQUE INDEX IF NOT EXISTS ux_retention_outbox_key
    ON authsome_retention_outbox (idempotency_key)
    WHERE idempotency_key != '';
`

// sqliteSchema is the same shape with SQLite's type names. TIMESTAMPTZ and
// NOW() do not exist there.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS authsome_retention_contact_ref (
    id                 TEXT PRIMARY KEY,
    app_id             TEXT NOT NULL,
    env_id             TEXT NOT NULL DEFAULT '',
    user_id            TEXT NOT NULL,
    provider           TEXT NOT NULL,
    remote_object_type TEXT NOT NULL DEFAULT '',
    remote_id          TEXT NOT NULL,
    synced_at          DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_retention_ref
    ON authsome_retention_contact_ref (app_id, env_id, user_id, provider);

CREATE TABLE IF NOT EXISTS authsome_retention_outbox (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL,
    env_id           TEXT NOT NULL DEFAULT '',
    user_id          TEXT NOT NULL,
    provider         TEXT NOT NULL,
    kind             TEXT NOT NULL,
    payload          TEXT NOT NULL DEFAULT '{}',
    idempotency_key  TEXT NOT NULL DEFAULT '',
    state            TEXT NOT NULL DEFAULT 'pending',
    attempts         INTEGER NOT NULL DEFAULT 0,
    next_attempt_at  DATETIME NOT NULL,
    in_flight_until  DATETIME,
    last_error       TEXT NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_retention_outbox_due
    ON authsome_retention_outbox (state, next_attempt_at);
CREATE UNIQUE INDEX IF NOT EXISTS ux_retention_outbox_key
    ON authsome_retention_outbox (idempotency_key)
    WHERE idempotency_key != '';
`

const dropSchema = `
DROP TABLE IF EXISTS authsome_retention_outbox;
DROP TABLE IF EXISTS authsome_retention_contact_ref;
`

// purgeIndexSchema backs PurgeTerminal's `WHERE state = ? AND created_at <
// ?` predicate. ix_retention_outbox_due (state, next_attempt_at) does not
// serve that query: created_at is not a prefix or a covered column, so
// Postgres and SQLite fall back to scanning every row in the matching
// state and testing created_at row by row. That scan runs once per
// terminal class, every PurgeInterval tick, for the life of the table --
// not once while the backlog is large, because purge deletes what is due
// and leaves everything not yet due sitting in the same state for the scan
// to pass over again next hour.
const purgeIndexSchema = `
CREATE INDEX IF NOT EXISTS ix_retention_outbox_purge
    ON authsome_retention_outbox (state, created_at);
`

const dropPurgeIndexSchema = `
DROP INDEX IF EXISTS ix_retention_outbox_purge;
`

func init() {
	PostgresMigrations.MustRegister(&migrate.Migration{
		Name:    "create_retention_tables",
		Version: "20260903000001",
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
		Name:    "create_retention_tables",
		Version: "20260903000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, sqliteSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropSchema)
			return err
		},
	})

	PostgresMigrations.MustRegister(&migrate.Migration{
		Name:    "add_retention_outbox_purge_index",
		Version: "20260903000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, purgeIndexSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropPurgeIndexSchema)
			return err
		},
	})

	SqliteMigrations.MustRegister(&migrate.Migration{
		Name:    "add_retention_outbox_purge_index",
		Version: "20260903000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, purgeIndexSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropPurgeIndexSchema)
			return err
		},
	})

	MongoMigrations.MustRegister(&migrate.Migration{
		Name:    "create_retention_indexes",
		Version: "20260903000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("retention: expected mongomigrate executor, got %T", exec)
			}
			if err := mexec.CreateIndexes(ctx, colContactRef, []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "app_id", Value: 1}, {Key: "env_id", Value: 1},
						{Key: "user_id", Value: 1}, {Key: "provider", Value: 1},
					},
					Options: options.Index().SetUnique(true),
				},
			}); err != nil {
				return err
			}
			// Same partial-uniqueness guard as the SQL schemas: an empty key
			// must never collide with another empty key.
			//
			// $ne is rejected by Mongo's partial-filter validator (it
			// desugars to $not, which the validator refuses outright), so
			// this uses $gt: "" instead. That is NOT equivalent to $ne: "":
			// $gt only matches documents where idempotency_key exists and is
			// a string, so it silently excludes missing/null, which $ne
			// would have caught. This is safe only because every write goes
			// through jobToDoc (store_mongo.go), which always sets the
			// field -- mirroring the SQL side's idempotency_key TEXT NOT
			// NULL DEFAULT ''. See the warning on outboxDoc.IdempotencyKey.
			return mexec.CreateIndexes(ctx, colOutbox, []mongo.IndexModel{
				{
					Keys: bson.D{{Key: "idempotency_key", Value: 1}},
					Options: options.Index().SetUnique(true).
						SetPartialFilterExpression(bson.D{
							{Key: "idempotency_key", Value: bson.D{{Key: "$gt", Value: ""}}},
						}),
				},
			})
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("retention: expected mongomigrate executor, got %T", exec)
			}
			if err := dropIndexIfExists(ctx, mexec.DB().Collection(colContactRef),
				"app_id_1_env_id_1_user_id_1_provider_1"); err != nil {
				return err
			}
			return dropIndexIfExists(ctx, mexec.DB().Collection(colOutbox), "idempotency_key_1")
		},
	})

	// Mongo has no counterpart to ix_retention_outbox_due at all, so
	// PurgeTerminal's {state, created_at} filter (store_mongo.go) runs as an
	// unindexed COLLSCAN today: three full collection scans every
	// PurgeInterval tick, for as long as the collection exists. Named to
	// match the SQL side's ix_retention_outbox_purge rather than left on
	// Mongo's default key-spec name, so Down (and a human reading the index
	// list) does not have to reconstruct it from the key spec.
	MongoMigrations.MustRegister(&migrate.Migration{
		Name:    "add_retention_outbox_purge_index",
		Version: "20260903000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("retention: expected mongomigrate executor, got %T", exec)
			}
			return mexec.CreateIndexes(ctx, colOutbox, []mongo.IndexModel{
				{
					Keys: bson.D{{Key: "state", Value: 1}, {Key: "created_at", Value: 1}},
					Options: options.Index().
						SetName("ix_retention_outbox_purge"),
				},
			})
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("retention: expected mongomigrate executor, got %T", exec)
			}
			return dropIndexIfExists(ctx, mexec.DB().Collection(colOutbox), "ix_retention_outbox_purge")
		},
	})
}

// dropIndexIfExists drops a Mongo index by name, tolerating the case where
// it is already gone (IndexNotFound, server error code 27) so the Down
// migration stays safe to run against a partially migrated database.
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
