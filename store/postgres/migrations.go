package postgres

import (
	"context"
	"fmt"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/id"
)

// Migrations is the grove migration group for the AuthSome store.
// It can be registered with the grove extension for orchestrated migration
// management (locking, version tracking, rollback support).
var Migrations = migrate.NewGroup("authsome")

func init() {
	Migrations.MustRegister(
		// Migration 1: Initial schema (11 tables: apps, users, sessions,
		// verifications, password_resets, organizations, members, invitations,
		// teams, devices, webhooks, notifications)
		&migrate.Migration{
			Name:    "create_initial_schema",
			Version: "20240101000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_apps (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    logo        TEXT NOT NULL DEFAULT '',
    is_platform BOOLEAN NOT NULL DEFAULT FALSE,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_apps_slug
    ON authsome_apps (slug);

CREATE TABLE IF NOT EXISTS authsome_users (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    email            TEXT NOT NULL,
    email_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    name             TEXT NOT NULL DEFAULT '',
    image            TEXT NOT NULL DEFAULT '',
    username         TEXT NOT NULL DEFAULT '',
    display_username TEXT NOT NULL DEFAULT '',
    phone            TEXT NOT NULL DEFAULT '',
    phone_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash    TEXT NOT NULL DEFAULT '',
    banned           BOOLEAN NOT NULL DEFAULT FALSE,
    ban_reason       TEXT NOT NULL DEFAULT '',
    ban_expires      TIMESTAMPTZ,
    metadata         JSONB DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_users_email
    ON authsome_users (app_id, email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_users_username
    ON authsome_users (app_id, username) WHERE deleted_at IS NULL AND username != '';
CREATE INDEX IF NOT EXISTS idx_authsome_users_app
    ON authsome_users (app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS authsome_sessions (
    id                        TEXT PRIMARY KEY,
    app_id                    TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id                   TEXT NOT NULL REFERENCES authsome_users(id),
    org_id                    TEXT NOT NULL DEFAULT '',
    token                     TEXT NOT NULL,
    refresh_token             TEXT NOT NULL,
    ip_address                TEXT NOT NULL DEFAULT '',
    user_agent                TEXT NOT NULL DEFAULT '',
    device_id                 TEXT NOT NULL DEFAULT '',
    expires_at                TIMESTAMPTZ NOT NULL,
    refresh_token_expires_at  TIMESTAMPTZ NOT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sessions_token
    ON authsome_sessions (token);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sessions_refresh_token
    ON authsome_sessions (refresh_token);
CREATE INDEX IF NOT EXISTS idx_authsome_sessions_user
    ON authsome_sessions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_sessions_expires
    ON authsome_sessions (expires_at);

CREATE TABLE IF NOT EXISTS authsome_verifications (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id    TEXT NOT NULL REFERENCES authsome_users(id),
    token      TEXT NOT NULL,
    type       TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_verifications_token
    ON authsome_verifications (token);

CREATE TABLE IF NOT EXISTS authsome_password_resets (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id    TEXT NOT NULL REFERENCES authsome_users(id),
    token      TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_password_resets_token
    ON authsome_password_resets (token);

CREATE TABLE IF NOT EXISTS authsome_organizations (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES authsome_apps(id),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    logo       TEXT NOT NULL DEFAULT '',
    metadata   JSONB DEFAULT '{}',
    created_by TEXT NOT NULL REFERENCES authsome_users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_organizations_slug
    ON authsome_organizations (app_id, slug);
CREATE INDEX IF NOT EXISTS idx_authsome_organizations_app
    ON authsome_organizations (app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS authsome_members (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES authsome_organizations(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES authsome_users(id),
    role       TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_members_user_org
    ON authsome_members (user_id, org_id);
CREATE INDEX IF NOT EXISTS idx_authsome_members_org
    ON authsome_members (org_id);

CREATE TABLE IF NOT EXISTS authsome_invitations (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES authsome_organizations(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'member',
    inviter_id TEXT NOT NULL REFERENCES authsome_users(id),
    status     TEXT NOT NULL DEFAULT 'pending',
    token      TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_invitations_token
    ON authsome_invitations (token);
CREATE INDEX IF NOT EXISTS idx_authsome_invitations_org
    ON authsome_invitations (org_id, status);

CREATE TABLE IF NOT EXISTS authsome_teams (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES authsome_organizations(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_teams_slug
    ON authsome_teams (org_id, slug);
CREATE INDEX IF NOT EXISTS idx_authsome_teams_org
    ON authsome_teams (org_id);

CREATE TABLE IF NOT EXISTS authsome_devices (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES authsome_users(id),
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    name         TEXT NOT NULL DEFAULT '',
    type         TEXT NOT NULL DEFAULT '',
    browser      TEXT NOT NULL DEFAULT '',
    os           TEXT NOT NULL DEFAULT '',
    ip_address   TEXT NOT NULL DEFAULT '',
    fingerprint  TEXT NOT NULL DEFAULT '',
    trusted      BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_devices_user
    ON authsome_devices (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_devices_fingerprint
    ON authsome_devices (user_id, fingerprint) WHERE fingerprint != '';

CREATE TABLE IF NOT EXISTS authsome_webhooks (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES authsome_apps(id),
    url        TEXT NOT NULL,
    events     JSONB NOT NULL DEFAULT '[]',
    secret     TEXT NOT NULL DEFAULT '',
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_webhooks_app
    ON authsome_webhooks (app_id, active);

CREATE TABLE IF NOT EXISTS authsome_notifications (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id    TEXT NOT NULL REFERENCES authsome_users(id),
    type       TEXT NOT NULL,
    channel    TEXT NOT NULL,
    subject    TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',
    sent       BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_notifications_user
    ON authsome_notifications (user_id, created_at DESC);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_notifications;
DROP TABLE IF EXISTS authsome_webhooks;
DROP TABLE IF EXISTS authsome_devices;
DROP TABLE IF EXISTS authsome_teams;
DROP TABLE IF EXISTS authsome_invitations;
DROP TABLE IF EXISTS authsome_members;
DROP TABLE IF EXISTS authsome_organizations;
DROP TABLE IF EXISTS authsome_password_resets;
DROP TABLE IF EXISTS authsome_verifications;
DROP TABLE IF EXISTS authsome_sessions;
DROP TABLE IF EXISTS authsome_users;
DROP TABLE IF EXISTS authsome_apps;
`)
				return err
			},
		},

		// Migration 2: API keys table
		&migrate.Migration{
			Name:    "create_api_keys",
			Version: "20240101000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_api_keys (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id      TEXT NOT NULL REFERENCES authsome_users(id),
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    key_prefix   TEXT NOT NULL,
    scopes       TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_app
    ON authsome_api_keys (app_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_user
    ON authsome_api_keys (app_id, user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_api_keys_prefix
    ON authsome_api_keys (app_id, key_prefix);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_api_keys_hash
    ON authsome_api_keys (key_hash);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_api_keys`)
				return err
			},
		},

		// Migration 3: Plugin tables (MFA enrollments, Passkey credentials, OAuth connections)
		&migrate.Migration{
			Name:    "create_plugin_tables",
			Version: "20240101000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_mfa_enrollments (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES authsome_users(id),
    method     TEXT NOT NULL,
    secret     TEXT NOT NULL,
    verified   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_mfa_enrollments_user_method
    ON authsome_mfa_enrollments (user_id, method);
CREATE INDEX IF NOT EXISTS idx_authsome_mfa_enrollments_user
    ON authsome_mfa_enrollments (user_id);

CREATE TABLE IF NOT EXISTS authsome_passkey_credentials (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES authsome_users(id),
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    credential_id    BYTEA NOT NULL,
    public_key       BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    transport        TEXT NOT NULL DEFAULT '',
    sign_count       INTEGER NOT NULL DEFAULT 0,
    aaguid           BYTEA,
    display_name     TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_cred_id
    ON authsome_passkey_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_user
    ON authsome_passkey_credentials (user_id);

CREATE TABLE IF NOT EXISTS authsome_oauth_connections (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL REFERENCES authsome_apps(id),
    user_id          TEXT NOT NULL REFERENCES authsome_users(id),
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email            TEXT NOT NULL DEFAULT '',
    access_token     TEXT NOT NULL DEFAULT '',
    refresh_token    TEXT NOT NULL DEFAULT '',
    expires_at       TIMESTAMPTZ,
    metadata         JSONB DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_oauth_connections_provider
    ON authsome_oauth_connections (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_authsome_oauth_connections_user
    ON authsome_oauth_connections (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_oauth_connections;
DROP TABLE IF EXISTS authsome_passkey_credentials;
DROP TABLE IF EXISTS authsome_mfa_enrollments;
`)
				return err
			},
		},

		// Migration 4: MFA recovery codes table
		&migrate.Migration{
			Name:    "create_mfa_recovery_codes",
			Version: "20240101000004",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_mfa_recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES authsome_users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,
    used       BOOLEAN NOT NULL DEFAULT FALSE,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_mfa_recovery_codes_user
    ON authsome_mfa_recovery_codes (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_mfa_recovery_codes`)
				return err
			},
		},

		// Migration 5: Add impersonated_by column to sessions
		&migrate.Migration{
			Name:    "add_session_impersonation",
			Version: "20240101000005",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS impersonated_by TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions DROP COLUMN IF EXISTS impersonated_by;
`)
				return err
			},
		},

		// Migration 6: SSO connections table
		&migrate.Migration{
			Name:    "create_sso_connections",
			Version: "20240101000006",
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
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_sso_connections`)
				return err
			},
		},

		// Migration 7: Environments table + env_id columns on all scoped tables.
		// Creates a default "Production" environment per app and backfills env_id.
		&migrate.Migration{
			Name:    "create_environments",
			Version: "20240101000007",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				// Step 1: Create environments table.
				if _, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_environments (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('development', 'staging', 'production')),
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    color       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    settings    JSONB DEFAULT NULL,
    cloned_from TEXT NOT NULL DEFAULT '',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_environments_slug
    ON authsome_environments (app_id, slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_environments_default
    ON authsome_environments (app_id) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_authsome_environments_app
    ON authsome_environments (app_id, created_at DESC);
`); err != nil {
					return fmt.Errorf("create environments table: %w", err)
				}

				// Step 2: Create default "Production" environment for each existing app.
				rows, err := exec.Query(ctx, "SELECT id FROM authsome_apps")
				if err != nil {
					return fmt.Errorf("list apps for backfill: %w", err)
				}
				defer rows.Close()

				var appIDs []string
				for rows.Next() {
					var aID string
					if err := rows.Scan(&aID); err != nil {
						return fmt.Errorf("scan app row: %w", err)
					}
					appIDs = append(appIDs, aID)
				}
				if err := rows.Err(); err != nil {
					return fmt.Errorf("iterate apps: %w", err)
				}

				for _, aID := range appIDs {
					envID := id.NewEnvironmentID()
					if _, err := exec.Exec(ctx,
						`INSERT INTO authsome_environments (id, app_id, name, slug, type, is_default, color)
						 VALUES ($1, $2, 'Production', 'production', 'production', TRUE, '#ef4444')`,
						envID.String(), aID,
					); err != nil {
						return fmt.Errorf("create default env for app %s: %w", aID, err)
					}
				}

				// Step 3: Add nullable env_id columns to all scoped tables.
				tables := []string{
					"authsome_users",
					"authsome_sessions",
					"authsome_organizations",
					"authsome_webhooks",
					"authsome_api_keys",
					"authsome_notifications",
					"authsome_devices",
					"authsome_verifications",
					"authsome_password_resets",
					"authsome_sso_connections",
				}
				for _, t := range tables {
					if _, err := exec.Exec(ctx, fmt.Sprintf(
						`ALTER TABLE %s ADD COLUMN IF NOT EXISTS env_id TEXT`, t,
					)); err != nil {
						return fmt.Errorf("add env_id to %s: %w", t, err)
					}
				}

				// Step 4: Backfill env_id from app's default environment.
				for _, t := range tables {
					if _, err := exec.Exec(ctx, fmt.Sprintf(
						`UPDATE %s AS tbl SET env_id = (
							SELECT e.id FROM authsome_environments e
							WHERE e.app_id = tbl.app_id AND e.is_default = TRUE
						) WHERE tbl.env_id IS NULL`, t,
					)); err != nil {
						return fmt.Errorf("backfill env_id for %s: %w", t, err)
					}
				}

				// Step 5: Make env_id NOT NULL and add FK constraints.
				for _, t := range tables {
					if _, err := exec.Exec(ctx, fmt.Sprintf(
						`ALTER TABLE %s ALTER COLUMN env_id SET NOT NULL`, t,
					)); err != nil {
						return fmt.Errorf("set env_id NOT NULL on %s: %w", t, err)
					}
					if _, err := exec.Exec(ctx, fmt.Sprintf(
						`ALTER TABLE %s ADD CONSTRAINT fk_%s_env
						 FOREIGN KEY (env_id) REFERENCES authsome_environments(id)`,
						t, t,
					)); err != nil {
						return fmt.Errorf("add env FK on %s: %w", t, err)
					}
				}

				// Step 6: Drop old unique indexes and recreate with env_id.
				if _, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_users_email;
CREATE UNIQUE INDEX idx_authsome_users_email
    ON authsome_users (app_id, env_id, email) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_authsome_users_username;
CREATE UNIQUE INDEX idx_authsome_users_username
    ON authsome_users (app_id, env_id, username) WHERE deleted_at IS NULL AND username != '';

DROP INDEX IF EXISTS idx_authsome_organizations_slug;
CREATE UNIQUE INDEX idx_authsome_organizations_slug
    ON authsome_organizations (app_id, env_id, slug);

DROP INDEX IF EXISTS idx_authsome_sso_connections_domain;
CREATE UNIQUE INDEX idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, env_id, domain) WHERE active = TRUE;
`); err != nil {
					return fmt.Errorf("recreate unique indexes: %w", err)
				}

				// Step 7: Add env-scoped lookup indexes.
				if _, err := exec.Exec(ctx, `
CREATE INDEX IF NOT EXISTS idx_authsome_users_env
    ON authsome_users (env_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_sessions_env
    ON authsome_sessions (env_id);
CREATE INDEX IF NOT EXISTS idx_authsome_organizations_env
    ON authsome_organizations (env_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_webhooks_env
    ON authsome_webhooks (env_id, active);
CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_env
    ON authsome_api_keys (env_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_authsome_devices_env
    ON authsome_devices (env_id);
CREATE INDEX IF NOT EXISTS idx_authsome_notifications_env
    ON authsome_notifications (env_id);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_env
    ON authsome_sso_connections (env_id);
`); err != nil {
					return fmt.Errorf("create env indexes: %w", err)
				}

				return nil
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				// Remove env_id columns and indexes (best-effort).
				tables := []string{
					"authsome_users",
					"authsome_sessions",
					"authsome_organizations",
					"authsome_webhooks",
					"authsome_api_keys",
					"authsome_notifications",
					"authsome_devices",
					"authsome_verifications",
					"authsome_password_resets",
					"authsome_sso_connections",
				}
				for _, t := range tables {
					_, _ = exec.Exec(ctx, fmt.Sprintf( //nolint:errcheck // best-effort migration
						`ALTER TABLE %s DROP CONSTRAINT IF EXISTS fk_%s_env`, t, t,
					))
					_, _ = exec.Exec(ctx, fmt.Sprintf( //nolint:errcheck // best-effort migration
						`ALTER TABLE %s DROP COLUMN IF EXISTS env_id`, t,
					))
				}

				// Restore original unique indexes.
				//nolint:errcheck // best-effort migration
				_, _ = exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_users_email;
CREATE UNIQUE INDEX idx_authsome_users_email
    ON authsome_users (app_id, email) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_authsome_users_username;
CREATE UNIQUE INDEX idx_authsome_users_username
    ON authsome_users (app_id, username) WHERE deleted_at IS NULL AND username != '';

DROP INDEX IF EXISTS idx_authsome_organizations_slug;
CREATE UNIQUE INDEX idx_authsome_organizations_slug
    ON authsome_organizations (app_id, slug);

DROP INDEX IF EXISTS idx_authsome_sso_connections_domain;
CREATE UNIQUE INDEX idx_authsome_sso_connections_domain
    ON authsome_sso_connections (app_id, domain) WHERE active = TRUE;
`)

				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_environments`)
				return err
			},
		},
		// Migration 8: Form configs and branding configs tables.
		&migrate.Migration{
			Name:    "create_form_and_branding_configs",
			Version: "20240101000008",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_form_configs (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES authsome_apps(id),
    form_type  TEXT NOT NULL DEFAULT 'signup',
    fields     JSONB NOT NULL DEFAULT '[]',
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    version    INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_form_configs_app_type
    ON authsome_form_configs (app_id, form_type);

CREATE TABLE IF NOT EXISTS authsome_branding_configs (
    id               TEXT PRIMARY KEY,
    org_id           TEXT NOT NULL,
    app_id           TEXT NOT NULL,
    logo_url         TEXT NOT NULL DEFAULT '',
    primary_color    TEXT NOT NULL DEFAULT '',
    background_color TEXT NOT NULL DEFAULT '',
    accent_color     TEXT NOT NULL DEFAULT '',
    font_family      TEXT NOT NULL DEFAULT '',
    custom_css       TEXT NOT NULL DEFAULT '',
    company_name     TEXT NOT NULL DEFAULT '',
    tagline          TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_branding_configs_org
    ON authsome_branding_configs (org_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_branding_configs;
DROP TABLE IF EXISTS authsome_form_configs;
`)
				return err
			},
		},

		// Migration 9: Per-app session configuration
		&migrate.Migration{
			Name:    "create_app_session_configs",
			Version: "20240101000009",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_app_session_configs (
    id                        TEXT PRIMARY KEY,
    app_id                    TEXT NOT NULL,
    token_ttl_seconds         INTEGER,
    refresh_token_ttl_seconds INTEGER,
    max_active_sessions       INTEGER,
    rotate_refresh_token      BOOLEAN,
    bind_to_ip                BOOLEAN,
    bind_to_device            BOOLEAN,
    token_format              TEXT NOT NULL DEFAULT '',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_app_session_configs_app
    ON authsome_app_session_configs (app_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_app_session_configs;`)
				return err
			},
		},
		// Migration 10: Rename name to first_name + last_name
		&migrate.Migration{
			Name:    "rename_user_name_to_first_last",
			Version: "20240101000010",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_users RENAME COLUMN name TO first_name;
ALTER TABLE authsome_users ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_users DROP COLUMN IF EXISTS last_name;
ALTER TABLE authsome_users RENAME COLUMN first_name TO name;
`)
				return err
			},
		},
		// Migration 11: Add public_key and public_key_prefix columns to API keys
		&migrate.Migration{
			Name:    "add_api_key_public_key",
			Version: "20240101000011",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_api_keys ADD COLUMN IF NOT EXISTS public_key TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_api_keys ADD COLUMN IF NOT EXISTS public_key_prefix TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_public_key ON authsome_api_keys (app_id, public_key);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_api_keys_public_key;
ALTER TABLE authsome_api_keys DROP COLUMN IF EXISTS public_key_prefix;
ALTER TABLE authsome_api_keys DROP COLUMN IF EXISTS public_key;
`)
				return err
			},
		},

		// Migration 12: Add publishable_key column to apps table
		&migrate.Migration{
			Name:    "add_app_publishable_key",
			Version: "20240101000012",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_apps ADD COLUMN IF NOT EXISTS publishable_key TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_apps_publishable_key
    ON authsome_apps (publishable_key) WHERE publishable_key != '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_apps_publishable_key;
ALTER TABLE authsome_apps DROP COLUMN IF EXISTS publishable_key;
`)
				return err
			},
		},

		// Migration 13: Add last_activity_at column to sessions table for sliding session extension.
		&migrate.Migration{
			Name:    "add_session_last_activity_at",
			Version: "20240101000013",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMPTZ;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions DROP COLUMN IF EXISTS last_activity_at;
`)
				return err
			},
		},

		// Migration 14: Create settings and app_client_configs tables.
		&migrate.Migration{
			Name:    "create_settings_and_app_client_configs",
			Version: "20240101000014",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_settings (
    id          TEXT PRIMARY KEY,
    key         TEXT NOT NULL,
    value       JSONB DEFAULT '{}',
    scope       TEXT NOT NULL DEFAULT 'global',
    scope_id    TEXT NOT NULL DEFAULT '',
    app_id      TEXT NOT NULL DEFAULT '',
    org_id      TEXT NOT NULL DEFAULT '',
    enforced    BOOLEAN NOT NULL DEFAULT FALSE,
    namespace   TEXT NOT NULL DEFAULT '',
    version     BIGINT NOT NULL DEFAULT 1,
    updated_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_settings_key_scope
    ON authsome_settings (key, scope, scope_id);
CREATE INDEX IF NOT EXISTS idx_authsome_settings_namespace
    ON authsome_settings (namespace) WHERE namespace != '';

CREATE TABLE IF NOT EXISTS authsome_app_client_configs (
    id                  TEXT PRIMARY KEY,
    app_id              TEXT NOT NULL UNIQUE,
    password_enabled    BOOLEAN,
    passkey_enabled     BOOLEAN,
    magic_link_enabled  BOOLEAN,
    mfa_enabled         BOOLEAN,
    sso_enabled         BOOLEAN,
    social_enabled      BOOLEAN,
    social_providers    JSONB DEFAULT '[]',
    mfa_methods         JSONB DEFAULT '[]',
    app_name            TEXT,
    logo_url            TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_app_client_configs;
DROP TABLE IF EXISTS authsome_settings;
`)
				return err
			},
		},

		// Migration 15: Add waitlist_enabled column to app_client_configs.
		&migrate.Migration{
			Name:    "add_waitlist_enabled_to_client_config",
			Version: "20260322000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_app_client_configs ADD COLUMN IF NOT EXISTS waitlist_enabled BOOLEAN;`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_app_client_configs DROP COLUMN IF EXISTS waitlist_enabled;`)
				return err
			},
		},
		// Migration 16: Add require_email_verification column to app_client_configs.
		&migrate.Migration{
			Name:    "add_require_email_verification_to_client_config",
			Version: "20260324000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_app_client_configs ADD COLUMN IF NOT EXISTS require_email_verification BOOLEAN;`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_app_client_configs DROP COLUMN IF EXISTS require_email_verification;`)
				return err
			},
		},
		// Migration 17: Add signup_enabled column to app_client_configs.
		&migrate.Migration{
			Name:    "add_signup_enabled_to_client_config",
			Version: "20260330000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_app_client_configs ADD COLUMN IF NOT EXISTS signup_enabled BOOLEAN;`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `ALTER TABLE authsome_app_client_configs DROP COLUMN IF EXISTS signup_enabled;`)
				return err
			},
		},
	)
}
