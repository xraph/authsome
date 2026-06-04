package sqlite

import (
	"context"
	"fmt"

	"github.com/xraph/grove/migrate"
)

// Migration 23: convert every timestamp column declared as TEXT to TIMESTAMP.
//
// The initial schema (and several later migrations) declared timestamp
// columns — created_at, updated_at, expires_at, ban_expires, last_seen_at,
// deleted_at, sent_at, used_at, revoked_at, … — with SQLite type TEXT. The
// grove sqlite driver writes time.Time as an RFC3339 string and relies on the
// modernc.org/sqlite driver to convert it back to time.Time on read. modernc
// only performs that conversion when the column's DECLARED type (lowercased)
// is one of date/datetime/time/timestamp; for a TEXT column it returns the
// raw string and database/sql then fails with:
//
//	sql: Scan error on column ...: unsupported Scan, storing driver.Value
//	type string into type *time.Time
//
// SQLite has no ALTER COLUMN TYPE, so each affected table is rebuilt with the
// standard table-rebuild procedure: create a replacement table whose timestamp
// columns are declared TIMESTAMP, copy every row, drop the original, rename the
// replacement into place, then recreate the indexes. No table in this schema
// declares foreign keys, triggers, or views, so the rebuild needs neither the
// PRAGMA foreign_keys toggle nor any reference rewriting.
//
// authsome_settings and authsome_app_client_configs (migration 14) and
// authsome_user_emails (migration 22) already declare DATETIME/TIMESTAMP and
// are intentionally left untouched.
//
// The reference declaration is authsome_user_emails in migration 22, which
// uses TIMESTAMP and round-trips correctly.

// timestampTableRebuild describes a single table-rebuild step.
type timestampTableRebuild struct {
	table   string // original table name
	create  string // CREATE TABLE <table>_new (...) with TIMESTAMP columns
	cols    string // every column, used for both the INSERT target and SELECT source
	indexes string // CREATE INDEX statements recreated after the rename
}

// rebuildTimestampTable performs the create → copy → drop → rename → reindex
// rebuild for one table. Columns are copied by name (identical list on both
// sides), so the copy is order-independent and a misnamed column fails loudly
// rather than silently shifting data between columns.
func rebuildTimestampTable(ctx context.Context, exec migrate.Executor, r timestampTableRebuild) error {
	if _, err := exec.Exec(ctx, r.create); err != nil {
		return fmt.Errorf("create %s_new: %w", r.table, err)
	}
	if _, err := exec.Exec(ctx, fmt.Sprintf(
		"INSERT INTO %s_new (%s) SELECT %s FROM %s;", r.table, r.cols, r.cols, r.table,
	)); err != nil {
		return fmt.Errorf("copy rows into %s_new: %w", r.table, err)
	}
	if _, err := exec.Exec(ctx, fmt.Sprintf("DROP TABLE %s;", r.table)); err != nil {
		return fmt.Errorf("drop old %s: %w", r.table, err)
	}
	if _, err := exec.Exec(ctx, fmt.Sprintf("ALTER TABLE %s_new RENAME TO %s;", r.table, r.table)); err != nil {
		return fmt.Errorf("rename %s_new: %w", r.table, err)
	}
	if r.indexes != "" {
		if _, err := exec.Exec(ctx, r.indexes); err != nil {
			return fmt.Errorf("recreate indexes for %s: %w", r.table, err)
		}
	}
	return nil
}

// timestampRebuilds lists every table whose timestamp columns were declared
// TEXT, with its replacement schema (timestamp columns as TIMESTAMP, all other
// columns unchanged including those added by later ALTERs) and its indexes.
var timestampRebuilds = []timestampTableRebuild{
	{
		table: "authsome_apps",
		create: `CREATE TABLE authsome_apps_new (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    logo            TEXT NOT NULL DEFAULT '',
    is_platform     INTEGER NOT NULL DEFAULT 0,
    metadata        TEXT DEFAULT '{}',
    created_at      TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at      TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    publishable_key TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, name, slug, logo, is_platform, metadata, created_at, updated_at, publishable_key",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_apps_slug ON authsome_apps (slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_apps_publishable_key ON authsome_apps (publishable_key) WHERE publishable_key != '';`,
	},
	{
		table: "authsome_users",
		create: `CREATE TABLE authsome_users_new (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL,
    email            TEXT NOT NULL,
    email_verified   INTEGER NOT NULL DEFAULT 0,
    first_name       TEXT NOT NULL DEFAULT '',
    image            TEXT NOT NULL DEFAULT '',
    username         TEXT NOT NULL DEFAULT '',
    display_username TEXT NOT NULL DEFAULT '',
    phone            TEXT NOT NULL DEFAULT '',
    phone_verified   INTEGER NOT NULL DEFAULT 0,
    password_hash    TEXT NOT NULL DEFAULT '',
    banned           INTEGER NOT NULL DEFAULT 0,
    ban_reason       TEXT NOT NULL DEFAULT '',
    ban_expires      TIMESTAMP,
    metadata         TEXT DEFAULT '{}',
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    deleted_at       TIMESTAMP,
    env_id           TEXT NOT NULL DEFAULT '',
    last_name        TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, app_id, email, email_verified, first_name, image, username, display_username, phone, phone_verified, password_hash, banned, ban_reason, ban_expires, metadata, created_at, updated_at, deleted_at, env_id, last_name",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_users_email ON authsome_users (app_id, email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_users_username ON authsome_users (app_id, username) WHERE username != '';
CREATE INDEX IF NOT EXISTS idx_authsome_users_app ON authsome_users (app_id, created_at);`,
	},
	{
		table: "authsome_sessions",
		create: `CREATE TABLE authsome_sessions_new (
    id                       TEXT PRIMARY KEY,
    app_id                   TEXT NOT NULL,
    user_id                  TEXT NOT NULL,
    org_id                   TEXT NOT NULL DEFAULT '',
    token                    TEXT NOT NULL,
    refresh_token            TEXT NOT NULL,
    ip_address               TEXT NOT NULL DEFAULT '',
    user_agent               TEXT NOT NULL DEFAULT '',
    device_id                TEXT NOT NULL DEFAULT '',
    expires_at               TIMESTAMP NOT NULL,
    refresh_token_expires_at TIMESTAMP NOT NULL,
    created_at               TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at               TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    impersonated_by          TEXT NOT NULL DEFAULT '',
    env_id                   TEXT NOT NULL DEFAULT '',
    last_activity_at         TIMESTAMP,
    family_id                TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, app_id, user_id, org_id, token, refresh_token, ip_address, user_agent, device_id, expires_at, refresh_token_expires_at, created_at, updated_at, impersonated_by, env_id, last_activity_at, family_id",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sessions_token ON authsome_sessions (token);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sessions_refresh_token ON authsome_sessions (refresh_token);
CREATE INDEX IF NOT EXISTS idx_authsome_sessions_user ON authsome_sessions (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_authsome_sessions_expires ON authsome_sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_authsome_sessions_family_id ON authsome_sessions (family_id);`,
	},
	{
		table: "authsome_verifications",
		create: `CREATE TABLE authsome_verifications_new (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    token      TEXT NOT NULL,
    type       TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    consumed   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id     TEXT NOT NULL DEFAULT ''
);`,
		cols:    "id, app_id, user_id, token, type, expires_at, consumed, created_at, env_id",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_verifications_token ON authsome_verifications (token);`,
	},
	{
		table: "authsome_password_resets",
		create: `CREATE TABLE authsome_password_resets_new (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    token      TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    consumed   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id     TEXT NOT NULL DEFAULT ''
);`,
		cols:    "id, app_id, user_id, token, expires_at, consumed, created_at, env_id",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_password_resets_token ON authsome_password_resets (token);`,
	},
	{
		table: "authsome_organizations",
		create: `CREATE TABLE authsome_organizations_new (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    logo       TEXT NOT NULL DEFAULT '',
    metadata   TEXT DEFAULT '{}',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id     TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, app_id, name, slug, logo, metadata, created_by, created_at, updated_at, env_id",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_organizations_slug ON authsome_organizations (app_id, slug);
CREATE INDEX IF NOT EXISTS idx_authsome_organizations_app ON authsome_organizations (app_id, created_at);`,
	},
	{
		table: "authsome_members",
		create: `CREATE TABLE authsome_members_new (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, org_id, user_id, role, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_members_user_org ON authsome_members (user_id, org_id);
CREATE INDEX IF NOT EXISTS idx_authsome_members_org ON authsome_members (org_id);`,
	},
	{
		table: "authsome_invitations",
		create: `CREATE TABLE authsome_invitations_new (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'member',
    inviter_id TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    token      TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, org_id, email, role, inviter_id, status, token, expires_at, created_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_invitations_token ON authsome_invitations (token);
CREATE INDEX IF NOT EXISTS idx_authsome_invitations_org ON authsome_invitations (org_id, status);`,
	},
	{
		table: "authsome_teams",
		create: `CREATE TABLE authsome_teams_new (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, org_id, name, slug, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_teams_slug ON authsome_teams (org_id, slug);
CREATE INDEX IF NOT EXISTS idx_authsome_teams_org ON authsome_teams (org_id);`,
	},
	{
		table: "authsome_devices",
		create: `CREATE TABLE authsome_devices_new (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    app_id       TEXT NOT NULL,
    name         TEXT NOT NULL DEFAULT '',
    type         TEXT NOT NULL DEFAULT '',
    browser      TEXT NOT NULL DEFAULT '',
    os           TEXT NOT NULL DEFAULT '',
    ip_address   TEXT NOT NULL DEFAULT '',
    fingerprint  TEXT NOT NULL DEFAULT '',
    trusted      INTEGER NOT NULL DEFAULT 0,
    last_seen_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    created_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id       TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, user_id, app_id, name, type, browser, os, ip_address, fingerprint, trusted, last_seen_at, created_at, updated_at, env_id",
		indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_devices_user ON authsome_devices (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_devices_fingerprint ON authsome_devices (user_id, fingerprint);`,
	},
	{
		table: "authsome_webhooks",
		create: `CREATE TABLE authsome_webhooks_new (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    url        TEXT NOT NULL,
    events     TEXT NOT NULL DEFAULT '[]',
    secret     TEXT NOT NULL DEFAULT '',
    active     INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id     TEXT NOT NULL DEFAULT ''
);`,
		cols:    "id, app_id, url, events, secret, active, created_at, updated_at, env_id",
		indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_webhooks_app ON authsome_webhooks (app_id, active);`,
	},
	{
		table: "authsome_notifications",
		create: `CREATE TABLE authsome_notifications_new (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    type       TEXT NOT NULL,
    channel    TEXT NOT NULL,
    subject    TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',
    sent       INTEGER NOT NULL DEFAULT 0,
    sent_at    TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id     TEXT NOT NULL DEFAULT ''
);`,
		cols:    "id, app_id, user_id, type, channel, subject, body, sent, sent_at, created_at, env_id",
		indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_notifications_user ON authsome_notifications (user_id, created_at);`,
	},
	{
		table: "authsome_api_keys",
		create: `CREATE TABLE authsome_api_keys_new (
    id                TEXT PRIMARY KEY,
    app_id            TEXT NOT NULL,
    user_id           TEXT NOT NULL,
    name              TEXT NOT NULL,
    key_hash          TEXT NOT NULL,
    key_prefix        TEXT NOT NULL,
    scopes            TEXT NOT NULL DEFAULT '',
    expires_at        TIMESTAMP,
    last_used_at      TIMESTAMP,
    revoked           INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at        TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id            TEXT NOT NULL DEFAULT '',
    public_key        TEXT NOT NULL DEFAULT '',
    public_key_prefix TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, app_id, user_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, revoked, created_at, updated_at, env_id, public_key, public_key_prefix",
		indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_app ON authsome_api_keys (app_id, created_at);
CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_user ON authsome_api_keys (app_id, user_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_api_keys_prefix ON authsome_api_keys (app_id, key_prefix);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_api_keys_hash ON authsome_api_keys (key_hash);
CREATE INDEX IF NOT EXISTS idx_authsome_api_keys_public_key ON authsome_api_keys (app_id, public_key);`,
	},
	{
		table: "authsome_mfa_enrollments",
		create: `CREATE TABLE authsome_mfa_enrollments_new (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    method     TEXT NOT NULL,
    secret     TEXT NOT NULL,
    verified   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, user_id, method, secret, verified, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_mfa_enrollments_user_method ON authsome_mfa_enrollments (user_id, method);
CREATE INDEX IF NOT EXISTS idx_authsome_mfa_enrollments_user ON authsome_mfa_enrollments (user_id);`,
	},
	{
		table: "authsome_passkey_credentials",
		create: `CREATE TABLE authsome_passkey_credentials_new (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    app_id           TEXT NOT NULL,
    credential_id    BLOB NOT NULL,
    public_key       BLOB NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    transport        TEXT NOT NULL DEFAULT '',
    sign_count       INTEGER NOT NULL DEFAULT 0,
    aaguid           BLOB,
    display_name     TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, user_id, app_id, credential_id, public_key, attestation_type, transport, sign_count, aaguid, display_name, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_cred_id ON authsome_passkey_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_authsome_passkey_credentials_user ON authsome_passkey_credentials (user_id);`,
	},
	{
		table: "authsome_oauth_connections",
		create: `CREATE TABLE authsome_oauth_connections_new (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL,
    user_id          TEXT NOT NULL,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email            TEXT NOT NULL DEFAULT '',
    access_token     TEXT NOT NULL DEFAULT '',
    refresh_token    TEXT NOT NULL DEFAULT '',
    expires_at       TIMESTAMP,
    metadata         TEXT DEFAULT '{}',
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, app_id, user_id, provider, provider_user_id, email, access_token, refresh_token, expires_at, metadata, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_oauth_connections_provider ON authsome_oauth_connections (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_authsome_oauth_connections_user ON authsome_oauth_connections (user_id);`,
	},
	{
		table: "authsome_mfa_recovery_codes",
		create: `CREATE TABLE authsome_mfa_recovery_codes_new (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    used       INTEGER NOT NULL DEFAULT 0,
    used_at    TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols:    "id, user_id, code_hash, used, used_at, created_at",
		indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_mfa_recovery_codes_user ON authsome_mfa_recovery_codes (user_id);`,
	},
	{
		table: "authsome_sso_connections",
		create: `CREATE TABLE authsome_sso_connections_new (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    org_id       TEXT NOT NULL DEFAULT '',
    provider     TEXT NOT NULL,
    protocol     TEXT NOT NULL,
    domain       TEXT NOT NULL,
    metadata_url TEXT NOT NULL DEFAULT '',
    client_id    TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL DEFAULT '',
    active       INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at   TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    env_id       TEXT NOT NULL DEFAULT ''
);`,
		cols: "id, app_id, org_id, provider, protocol, domain, metadata_url, client_id, issuer, active, created_at, updated_at, env_id",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sso_connections_domain ON authsome_sso_connections (app_id, domain);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_app ON authsome_sso_connections (app_id, created_at);
CREATE INDEX IF NOT EXISTS idx_authsome_sso_connections_provider ON authsome_sso_connections (app_id, provider);`,
	},
	{
		table: "authsome_environments",
		create: `CREATE TABLE authsome_environments_new (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    type        TEXT NOT NULL,
    is_default  INTEGER NOT NULL DEFAULT 0,
    color       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    settings    TEXT DEFAULT NULL,
    cloned_from TEXT NOT NULL DEFAULT '',
    metadata    TEXT DEFAULT '{}',
    created_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols: "id, app_id, name, slug, type, is_default, color, description, settings, cloned_from, metadata, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_environments_slug ON authsome_environments (app_id, slug);
CREATE INDEX IF NOT EXISTS idx_authsome_environments_app ON authsome_environments (app_id, created_at);`,
	},
	{
		table: "authsome_form_configs",
		create: `CREATE TABLE authsome_form_configs_new (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    form_type  TEXT NOT NULL DEFAULT 'signup',
    fields     TEXT NOT NULL DEFAULT '[]',
    active     INTEGER NOT NULL DEFAULT 1,
    version    INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols:    "id, app_id, form_type, fields, active, version, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_form_configs_app_type ON authsome_form_configs (app_id, form_type);`,
	},
	{
		table: "authsome_branding_configs",
		create: `CREATE TABLE authsome_branding_configs_new (
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
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols:    "id, org_id, app_id, logo_url, primary_color, background_color, accent_color, font_family, custom_css, company_name, tagline, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_branding_configs_org ON authsome_branding_configs (org_id);`,
	},
	{
		table: "authsome_app_session_configs",
		create: `CREATE TABLE authsome_app_session_configs_new (
    id                        TEXT PRIMARY KEY,
    app_id                    TEXT NOT NULL,
    token_ttl_seconds         INTEGER,
    refresh_token_ttl_seconds INTEGER,
    max_active_sessions       INTEGER,
    rotate_refresh_token      INTEGER,
    bind_to_ip                INTEGER,
    bind_to_device            INTEGER,
    token_format              TEXT NOT NULL DEFAULT '',
    created_at                TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at                TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);`,
		cols:    "id, app_id, token_ttl_seconds, refresh_token_ttl_seconds, max_active_sessions, rotate_refresh_token, bind_to_ip, bind_to_device, token_format, created_at, updated_at",
		indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_app_session_configs_app ON authsome_app_session_configs (app_id);`,
	},
	{
		table: "authsome_revoked_refresh_tokens",
		create: `CREATE TABLE authsome_revoked_refresh_tokens_new (
    token_hash TEXT PRIMARY KEY,
    family_id  TEXT NOT NULL,
    revoked_at TIMESTAMP NOT NULL,
    reason     TEXT NOT NULL DEFAULT ''
);`,
		cols:    "token_hash, family_id, revoked_at, reason",
		indexes: `CREATE INDEX IF NOT EXISTS idx_authsome_revoked_refresh_tokens_family_id ON authsome_revoked_refresh_tokens (family_id);`,
	},
}

func init() {
	Migrations.MustRegister(
		&migrate.Migration{
			Name:    "convert_text_timestamps_to_timestamp",
			Version: "20260601000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				for _, r := range timestampRebuilds {
					if err := rebuildTimestampTable(ctx, exec, r); err != nil {
						return fmt.Errorf("rebuild %s: %w", r.table, err)
					}
				}
				return nil
			},
			// Forward-only: the previous TEXT declaration is the source of the
			// scan bug this migration fixes, so there is no value in rebuilding
			// the tables back to TEXT. Down is a documented no-op.
			Down: func(_ context.Context, _ migrate.Executor) error {
				return nil
			},
		},
	)
}
