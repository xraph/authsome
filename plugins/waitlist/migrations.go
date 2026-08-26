package waitlist

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the waitlist plugin.
var PostgresMigrations = migrate.NewGroup("authsome-waitlist", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the waitlist plugin.
var SqliteMigrations = migrate.NewGroup("authsome-waitlist", migrate.DependsOn("authsome"))

// MongoMigrations is the MongoDB migration group for the waitlist plugin.
// MongoDB is schemaless so no actual migrations are needed.
var MongoMigrations = migrate.NewGroup("authsome-waitlist", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_waitlist_tables",
			Version: "20240601000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_waitlist_entries (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    email       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    user_id     TEXT,
    ip_address  TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, email)
);

CREATE INDEX IF NOT EXISTS idx_authsome_waitlist_entries_app_status
    ON authsome_waitlist_entries (app_id, status);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_waitlist_entries;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_waitlist_tables",
			Version: "20240601000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_waitlist_entries (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    email       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    user_id     TEXT,
    ip_address  TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(app_id, email)
);

CREATE INDEX IF NOT EXISTS idx_authsome_waitlist_entries_app_status
    ON authsome_waitlist_entries (app_id, status);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_waitlist_entries;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// MongoDB migrations (no-op — schemaless)
	// ──────────────────────────────────────────────────

	MongoMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_waitlist_collections",
			Version: "20240601000001",
			Up: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
		},
		&migrate.Migration{
			// The SQL schemas have carried UNIQUE(app_id, email) since the
			// table was created. Mongo had no equivalent, so the same address
			// could join one app's waitlist any number of times, and
			// CreateEntry's duplicate-key mapping onto ErrDuplicateEmail was
			// unreachable because nothing ever raised the error.
			Name:    "unique_index_on_app_and_email",
			Version: "20260826000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("waitlist: expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.CreateCollection(ctx, (*waitlistDoc)(nil)); err != nil {
					return err
				}
				coll := mexec.DB().Collection(waitlistColl)
				if err := waitlistRefuseOnDuplicates(ctx, coll); err != nil {
					return err
				}
				return mexec.CreateIndexes(ctx, waitlistColl, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "email", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("waitlist: expected mongomigrate executor, got %T", exec)
				}
				err := mexec.DB().Collection(waitlistColl).
					Indexes().DropOne(ctx, "app_id_1_email_1")
				if err != nil && !strings.Contains(err.Error(), "index not found") {
					return err
				}
				return nil
			},
		},
	)
}

// waitlistRefuseOnDuplicates fails the migration, without touching anything,
// when the collection already holds two entries for one address on one app.
//
// Creating the unique index over existing duplicates would fail anyway, with
// a bare E11000 naming a single offending key and nothing about the scale of
// the problem. This reports every affected pair instead, because the fix is a
// judgement call about somebody's waitlist position and not one a migration
// should make on its own: deleting the newer row is usually right, but only
// the operator knows whether the older or the newer record is the one that
// carries the note, the referral or the approval.
func waitlistRefuseOnDuplicates(ctx context.Context, coll *mongo.Collection) error {
	cursor, err := coll.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "app_id", Value: "$app_id"},
				{Key: "email", Value: "$email"},
			}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "count", Value: bson.D{{Key: "$gt", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
	})
	if err != nil {
		return fmt.Errorf("waitlist: scan for duplicate entries: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var dupes []struct {
		ID struct {
			AppID string `bson:"app_id"`
			Email string `bson:"email"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}
	if err := cursor.All(ctx, &dupes); err != nil {
		return fmt.Errorf("waitlist: read duplicate entries: %w", err)
	}
	if len(dupes) == 0 {
		return nil
	}

	// Name a bounded sample. A collection with thousands of duplicates should
	// not produce an error message nobody can read.
	const sample = 10
	var b strings.Builder
	var extra int
	for i, d := range dupes {
		if i >= sample {
			extra += d.Count - 1
			continue
		}
		fmt.Fprintf(&b, "\n  app %s, %s: %d entries", d.ID.AppID, d.ID.Email, d.Count)
	}
	if extra > 0 {
		fmt.Fprintf(&b, "\n  ...and %d more duplicate entries across %d further addresses",
			extra, len(dupes)-sample)
	}
	noun, verb := "addresses", "appear"
	if len(dupes) == 1 {
		noun, verb = "address", "appears"
	}
	return fmt.Errorf(
		"waitlist: cannot add the unique index on (app_id, email): %d %s already %s "+
			"more than once, and the migration will not choose which entries to remove.%s\n"+
			"Resolve them first, keeping one entry per address per app, then run the migration again",
		len(dupes), noun, verb, b.String())
}
