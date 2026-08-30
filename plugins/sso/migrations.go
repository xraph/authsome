package sso

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the SSO plugin.
var PostgresMigrations = migrate.NewGroup("authsome-sso", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the SSO plugin.
var SqliteMigrations = migrate.NewGroup("authsome-sso", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_sso_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_sso_connections (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    org_id       TEXT NOT NULL DEFAULT '',
    provider     TEXT NOT NULL,
    protocol     TEXT NOT NULL,
    domain       TEXT NOT NULL,
    metadata_url TEXT NOT NULL DEFAULT '',
    client_id    TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL DEFAULT '',
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_app
    ON authsome_sso_connections (app_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_provider
    ON authsome_sso_connections (app_id, provider) WHERE active = TRUE;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_sso_connections;`)
				return err
			},
		},
	)

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_client_secret",
			Version: "20240201000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS client_secret TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS client_secret;`)
				return err
			},
		},
	)

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_saml_fields",
			Version: "20240201000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS idp_metadata_xml    TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS idp_sso_url         TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS idp_certificate     TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS entity_id           TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS acs_url             TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS sp_certificate      TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS sp_private_key      TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS sign_requests       BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS attribute_mappings  TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS idp_metadata_xml;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS idp_sso_url;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS idp_certificate;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS entity_id;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS acs_url;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS sp_certificate;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS sp_private_key;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS sign_requests;
ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS attribute_mappings;
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
			Name:    "create_sso_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_sso_connections (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    org_id       TEXT NOT NULL DEFAULT '',
    provider     TEXT NOT NULL,
    protocol     TEXT NOT NULL,
    domain       TEXT NOT NULL,
    metadata_url TEXT NOT NULL DEFAULT '',
    client_id    TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL DEFAULT '',
    active       INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_app
    ON authsome_sso_connections (app_id, created_at);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_provider
    ON authsome_sso_connections (app_id, provider);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_sso_connections;`)
				return err
			},
		},
	)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_client_secret",
			Version: "20240201000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections ADD COLUMN client_secret TEXT NOT NULL DEFAULT '';`)
				return err
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				// SQLite does not support DROP COLUMN in older versions; best-effort.
				return nil
			},
		},
	)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_saml_fields",
			Version: "20240201000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				// SQLite requires one ADD COLUMN per statement.
				cols := []string{
					`ALTER TABLE authsome_sso_connections ADD COLUMN idp_metadata_xml   TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN idp_sso_url        TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN idp_certificate    TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN entity_id          TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN acs_url            TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN sp_certificate     TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN sp_private_key     TEXT NOT NULL DEFAULT '';`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN sign_requests      INTEGER NOT NULL DEFAULT 0;`,
					`ALTER TABLE authsome_sso_connections ADD COLUMN attribute_mappings TEXT NOT NULL DEFAULT '';`,
				}
				for _, stmt := range cols {
					if _, err := exec.Exec(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				// SQLite does not support DROP COLUMN in older versions; best-effort.
				return nil
			},
		},
	)

	// ──────────────────────────────────────────────────
	// Multi-tenant: scope the domain uniqueness by org
	// ──────────────────────────────────────────────────
	//
	// The same email domain may now be configured for SSO in several orgs
	// (workspaces), so uniqueness moves from (app_id, domain) to
	// (app_id, org_id, domain). The drop clears whichever prior form exists —
	// the core store used (app_id, env_id, domain); the plugin used
	// (app_id, domain). env_id is intentionally omitted (the plugin schema has
	// no env_id column, and org is the tenant boundary for SSO, not env).

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "scope_domain_index_by_org",
			Version: "20240201000004",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_sso_connections_domain;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, org_id, domain) WHERE active = TRUE;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_sso_connections_domain;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain) WHERE active = TRUE;
`)
				return err
			},
		},
	)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "scope_domain_index_by_org",
			Version: "20240201000004",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_sso_connections_domain;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, org_id, domain) WHERE active = 1;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_sso_connections_domain;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain);
`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SSO enforcement: require SSO for a connection's domain
	// ──────────────────────────────────────────────────
	// When enforced, the plugin's BeforeSignIn vetoes password login for users on
	// the connection's domain (owners/admins excepted). Defaults false.

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_enforced",
			Version: "20240201000005",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections ADD COLUMN IF NOT EXISTS enforced BOOLEAN NOT NULL DEFAULT FALSE;`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections DROP COLUMN IF EXISTS enforced;`)
				return err
			},
		},
	)

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_enforced",
			Version: "20240201000005",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_sso_connections ADD COLUMN enforced INTEGER NOT NULL DEFAULT 0;`)
				return err
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				return nil // SQLite lacks DROP COLUMN on older versions; best-effort.
			},
		},
	)
}
