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
// because the outbox and contact-ref tables reference authsome_apps.
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
			return mexec.CreateIndexes(ctx, colOutbox, []mongo.IndexModel{
				{
					Keys: bson.D{{Key: "idempotency_key", Value: 1}},
					Options: options.Index().SetUnique(true).
						SetPartialFilterExpression(bson.D{
							{Key: "idempotency_key", Value: bson.D{{Key: "$ne", Value: ""}}},
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
