package agentauth

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the agentauth plugin.
var PostgresMigrations = migrate.NewGroup("authsome-agentauth", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the agentauth plugin.
var SqliteMigrations = migrate.NewGroup("authsome-agentauth", migrate.DependsOn("authsome"))

// MongoMigrations is the MongoDB migration group for the agentauth plugin.
// MongoDB is schemaless, so this exists to create the client_id uniqueness
// index — the same atomicity guarantee CreateAgent needs on SQL, since a
// check-then-insert at the application layer races the same way there.
var MongoMigrations = migrate.NewGroup("authsome-agentauth", migrate.DependsOn("authsome"))

// Deliberate deviation from a plain port of the brief's sketch schema: these
// tables carry no REFERENCES to authsome_apps/authsome_agents. The Store
// interface (see MemoryStore) never validates that an AppID or AgentID
// names a row that actually exists — CreateAgentGrant just inserts — so an
// FK enforced only on the SQL backends would make an identical call succeed
// on memory/mongo and fail on postgres/sqlite: exactly the cross-backend
// drift this task exists to close. The one uniqueness invariant the Store
// interface documents (ErrConflict on a duplicate ClientID) is enforced
// everywhere via a real unique index instead.
func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_agentauth_tables",
			Version: "20260824000061",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_agents (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL,
    org_id      TEXT NOT NULL DEFAULT '',
    client_id   TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_uri    TEXT NOT NULL DEFAULT '',
    origin      TEXT NOT NULL,
    status      TEXT NOT NULL,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial: MemoryStore.CreateAgent only checks for a ClientID collision
-- when a.ClientID != "", so a bare UNIQUE index here would reject a second
-- agent with an unset ClientID even though the in-memory backend allows it.
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_agents_client_id
    ON authsome_agents (client_id) WHERE client_id <> '';

CREATE INDEX IF NOT EXISTS idx_authsome_agents_app_id
    ON authsome_agents (app_id);

CREATE TABLE IF NOT EXISTS authsome_agent_grants (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    agent_id     TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    org_id       TEXT NOT NULL DEFAULT '',
    scopes       TEXT NOT NULL DEFAULT '[]',
    consent_id   TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Every grant names the human who authorized it. The intersection model
    -- has no meaning without one, so this is enforced rather than assumed.
    CONSTRAINT authsome_agent_grants_user_required CHECK (user_id <> '')
);

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_user
    ON authsome_agent_grants (user_id);

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_agent
    ON authsome_agent_grants (agent_id);

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_org
    ON authsome_agent_grants (org_id);

CREATE TABLE IF NOT EXISTS authsome_agent_policies (
    org_id         TEXT PRIMARY KEY,
    mode           TEXT NOT NULL,
    max_grant_ttl  BIGINT NOT NULL DEFAULT 0,
    allowed_scopes TEXT NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_agent_policies;
DROP TABLE IF EXISTS authsome_agent_grants;
DROP TABLE IF EXISTS authsome_agents;
`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────
	//
	// Mirrors the Postgres group: TIMESTAMP-declared timestamp columns,
	// INTEGER for max_grant_ttl, and SQLite's own partial-index syntax
	// (identical WHERE clause; SQLite has supported partial indexes since
	// 3.8.0).
	//
	// Timestamp columns are declared TIMESTAMP, not TEXT, deliberately:
	// store/sqlite/migrations_timestamps.go (migration 23 in the core
	// store) rebuilt every core table for exactly this reason.
	// modernc.org/sqlite only converts a stored value back into time.Time
	// when the column's DECLARED type is one of date/datetime/time/
	// timestamp; a TEXT column returns the raw string and scanning it into
	// a time.Time field fails. This plugin's schema is new, so unlike the
	// core store there is no rebuild to do — declaring the columns
	// correctly from the start avoids ever needing the workaround. (The
	// store layer also normalizes every timestamp through .UTC() before
	// writing regardless, since that closes a second, unrelated failure —
	// see the utc() helper in store_models.go — so the store code is
	// unaffected by this column-type fix either way.)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_agentauth_tables",
			Version: "20260824000061",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_agents (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL,
    org_id      TEXT NOT NULL DEFAULT '',
    client_id   TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_uri    TEXT NOT NULL DEFAULT '',
    origin      TEXT NOT NULL,
    status      TEXT NOT NULL,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_agents_client_id
    ON authsome_agents (client_id) WHERE client_id <> '';

CREATE INDEX IF NOT EXISTS idx_authsome_agents_app_id
    ON authsome_agents (app_id);

CREATE TABLE IF NOT EXISTS authsome_agent_grants (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    agent_id     TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    org_id       TEXT NOT NULL DEFAULT '',
    scopes       TEXT NOT NULL DEFAULT '[]',
    consent_id   TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP,
    revoked_at   TIMESTAMP,
    created_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    CHECK (user_id <> '')
);

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_user
    ON authsome_agent_grants (user_id);

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_agent
    ON authsome_agent_grants (agent_id);

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_org
    ON authsome_agent_grants (org_id);

CREATE TABLE IF NOT EXISTS authsome_agent_policies (
    org_id         TEXT PRIMARY KEY,
    mode           TEXT NOT NULL,
    max_grant_ttl  INTEGER NOT NULL DEFAULT 0,
    allowed_scopes TEXT NOT NULL DEFAULT '[]',
    created_at     TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at     TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_agent_policies;
DROP TABLE IF EXISTS authsome_agent_grants;
DROP TABLE IF EXISTS authsome_agents;
`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// MongoDB migrations
	// ──────────────────────────────────────────────────
	//
	// No CreateCollection call (and so no $jsonSchema validator): plugin
	// stores in this codebase (consent, waitlist) don't validate schema at
	// the mongo layer the way the core store does, and CreateCollection
	// requires the model to carry grove struct tags the agentauth doc types
	// don't have. CreateIndexes alone is enough — mongo creates a collection
	// implicitly on first index creation or first insert — and it is the
	// part that actually matters here: the unique index is what makes
	// CreateAgent's ErrConflict atomic instead of a racy check-then-insert.
	MongoMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_agentauth_indexes",
			Version: "20260824000061",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateIndexes(ctx, agentsColl, []mongo.IndexModel{
					{
						// Partial for the same reason as the Postgres index:
						// MemoryStore only enforces uniqueness when
						// ClientID != "".
						Keys: bson.D{{Key: "client_id", Value: 1}},
						Options: options.Index().SetUnique(true).
							SetPartialFilterExpression(bson.M{"client_id": bson.M{"$gt": ""}}),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}}},
				}); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, agentGrantsColl, []mongo.IndexModel{
					{Keys: bson.D{{Key: "user_id", Value: 1}}},
					{Keys: bson.D{{Key: "agent_id", Value: 1}}},
					{Keys: bson.D{{Key: "org_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				_ = mexec.DB().Collection(agentsColl).Drop(ctx)        //nolint:errcheck // best-effort rollback
				_ = mexec.DB().Collection(agentGrantsColl).Drop(ctx)   //nolint:errcheck // best-effort rollback
				_ = mexec.DB().Collection(agentPoliciesColl).Drop(ctx) //nolint:errcheck // best-effort rollback
				return nil
			},
		},
	)
}
