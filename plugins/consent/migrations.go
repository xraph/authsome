package consent

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the consent plugin.
var PostgresMigrations = migrate.NewGroup("authsome-consent", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the consent plugin.
var SqliteMigrations = migrate.NewGroup("authsome-consent", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_consent_tables",
			Version: "20240401000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_consents (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES authsome_users(id),
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    purpose     TEXT NOT NULL,
    granted     BOOLEAN NOT NULL DEFAULT TRUE,
    version     TEXT NOT NULL DEFAULT '',
    ip_address  TEXT NOT NULL DEFAULT '',
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_consents_user_app_purpose
    ON authsome_consents (user_id, app_id, purpose);

CREATE INDEX IF NOT EXISTS idx_authsome_consents_user
    ON authsome_consents (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_consents;`)
				return err
			},
		},
	)

	// ──────────────────────────────────────────────────
	// SQLite migrations
	// ──────────────────────────────────────────────────

	SqliteMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_consent_tables",
			Version: "20240401000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_consents (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES authsome_users(id),
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    purpose     TEXT NOT NULL,
    granted     INTEGER NOT NULL DEFAULT 1,
    version     TEXT NOT NULL DEFAULT '',
    ip_address  TEXT NOT NULL DEFAULT '',
    granted_at  TEXT NOT NULL DEFAULT (datetime('now')),
    revoked_at  TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_consents_user_app_purpose
    ON authsome_consents (user_id, app_id, purpose);

CREATE INDEX IF NOT EXISTS idx_authsome_consents_user
    ON authsome_consents (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_consents;`)
				return err
			},
		},
	)
}
