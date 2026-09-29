package scim

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the SCIM plugin.
var PostgresMigrations = migrate.NewGroup("authsome-scim", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the SCIM plugin.
var SqliteMigrations = migrate.NewGroup("authsome-scim", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_scim_tables",
			Version: "20260307000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_scim_configs (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    org_id       TEXT,
    name         TEXT NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    auto_create  BOOLEAN NOT NULL DEFAULT TRUE,
    auto_suspend BOOLEAN NOT NULL DEFAULT TRUE,
    group_sync   BOOLEAN NOT NULL DEFAULT FALSE,
    default_role TEXT NOT NULL DEFAULT 'member',
    metadata     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_configs_app
    ON authsome_scim_configs (app_id);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_configs_org
    ON authsome_scim_configs (org_id);

CREATE TABLE IF NOT EXISTS authsome_scim_tokens (
    id           TEXT PRIMARY KEY,
    config_id    TEXT NOT NULL REFERENCES authsome_scim_configs(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_tokens_config
    ON authsome_scim_tokens (config_id);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_tokens_hash
    ON authsome_scim_tokens (token_hash);

CREATE TABLE IF NOT EXISTS authsome_scim_provision_logs (
    id            TEXT PRIMARY KEY,
    config_id     TEXT NOT NULL REFERENCES authsome_scim_configs(id) ON DELETE CASCADE,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    external_id   TEXT,
    internal_id   TEXT,
    status        TEXT NOT NULL,
    detail        TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_config
    ON authsome_scim_provision_logs (config_id);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_status
    ON authsome_scim_provision_logs (status);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_created
    ON authsome_scim_provision_logs (created_at DESC);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_scim_provision_logs;
DROP TABLE IF EXISTS authsome_scim_tokens;
DROP TABLE IF EXISTS authsome_scim_configs;
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
			Name:    "create_scim_tables",
			Version: "20260307000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_scim_configs (
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
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_configs_app
    ON authsome_scim_configs (app_id);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_configs_org
    ON authsome_scim_configs (org_id);

CREATE TABLE IF NOT EXISTS authsome_scim_tokens (
    id           TEXT PRIMARY KEY,
    config_id    TEXT NOT NULL REFERENCES authsome_scim_configs(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL,
    last_used_at TEXT,
    expires_at   TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_tokens_config
    ON authsome_scim_tokens (config_id);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_tokens_hash
    ON authsome_scim_tokens (token_hash);

CREATE TABLE IF NOT EXISTS authsome_scim_provision_logs (
    id            TEXT PRIMARY KEY,
    config_id     TEXT NOT NULL REFERENCES authsome_scim_configs(id) ON DELETE CASCADE,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    external_id   TEXT,
    internal_id   TEXT,
    status        TEXT NOT NULL,
    detail        TEXT,
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_config
    ON authsome_scim_provision_logs (config_id);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_status
    ON authsome_scim_provision_logs (status);
CREATE INDEX IF NOT EXISTS idx_authsome_scim_logs_created
    ON authsome_scim_provision_logs (created_at DESC);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_scim_provision_logs;
DROP TABLE IF EXISTS authsome_scim_tokens;
DROP TABLE IF EXISTS authsome_scim_configs;
`)
				return err
			},
		},
	)
}

func init() {
	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "scim_token_lookup",
			Version: "20260922000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				// The lookup digest lets a presented token be resolved by one
				// indexed equality; the bcrypt digest is then compared on that
				// row alone. NULL on rows written before, which are scanned and
				// upgraded on their next use.
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_scim_tokens ADD COLUMN IF NOT EXISTS token_lookup TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_scim_tokens_lookup
    ON authsome_scim_tokens (token_lookup);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_scim_tokens_lookup;
ALTER TABLE authsome_scim_tokens DROP COLUMN IF EXISTS token_lookup;
`)
				return err
			},
		},
	)
	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "scim_token_lookup",
			Version: "20260922000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				for _, stmt := range []string{
					`ALTER TABLE authsome_scim_tokens ADD COLUMN token_lookup TEXT`,
					`CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_scim_tokens_lookup ON authsome_scim_tokens (token_lookup)`,
				} {
					if _, err := exec.Exec(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP INDEX IF EXISTS idx_authsome_scim_tokens_lookup`)
				return err
			},
		},
	)
}
