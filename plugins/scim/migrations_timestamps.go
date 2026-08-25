package scim

import (
	"context"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/internal/sqliteschema"
)

// The three SCIM tables declared their timestamp columns as TEXT, so a config,
// token or provisioning log written with an ordinary time.Now() could not be
// read back. See the sqliteschema package doc for the full mechanism, and
// store/sqlite/migrations_timestamps.go for the core store's equivalent fix.
//
// authsome_scim_tokens and authsome_scim_provision_logs both reference
// authsome_scim_configs ON DELETE CASCADE, which is why RebuildTables turns
// foreign keys off: dropping the configs table with enforcement on would empty
// both children.
var timestampRebuilds = []sqliteschema.TableRebuild{
	{
		Table: "authsome_scim_configs",
		Create: `CREATE TABLE authsome_scim_configs_new (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    org_id       TEXT,
    name         TEXT NOT NULL,
    enabled      INTEGER NOT NULL DEFAULT 1,
    auto_create  INTEGER NOT NULL DEFAULT 1,
    auto_suspend INTEGER NOT NULL DEFAULT 1,
    group_sync   INTEGER NOT NULL DEFAULT 0,
    default_role TEXT NOT NULL DEFAULT 'member',
    metadata     TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at   TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		Columns: "id, app_id, org_id, name, enabled, auto_create, auto_suspend, group_sync, default_role, metadata, created_at, updated_at",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_scim_configs_app
    ON authsome_scim_configs (app_id);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_configs_org
    ON authsome_scim_configs (org_id);`,
	},
	{
		Table: "authsome_scim_tokens",
		Create: `CREATE TABLE authsome_scim_tokens_new (
    id           TEXT PRIMARY KEY,
    config_id    TEXT NOT NULL REFERENCES authsome_scim_configs(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL,
    last_used_at TIMESTAMP,
    expires_at   TIMESTAMP,
    created_at   TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		Columns: "id, config_id, name, token_hash, last_used_at, expires_at, created_at",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_scim_tokens_config
    ON authsome_scim_tokens (config_id);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_tokens_hash
    ON authsome_scim_tokens (token_hash);`,
	},
	{
		Table: "authsome_scim_provision_logs",
		Create: `CREATE TABLE authsome_scim_provision_logs_new (
    id            TEXT PRIMARY KEY,
    config_id     TEXT NOT NULL REFERENCES authsome_scim_configs(id) ON DELETE CASCADE,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    external_id   TEXT,
    internal_id   TEXT,
    status        TEXT NOT NULL,
    detail        TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		Columns: "id, config_id, action, resource_type, external_id, internal_id, status, detail, created_at",
		Indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_config
    ON authsome_scim_provision_logs (config_id);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_status
    ON authsome_scim_provision_logs (status);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_created
    ON authsome_scim_provision_logs (created_at DESC);`,
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
