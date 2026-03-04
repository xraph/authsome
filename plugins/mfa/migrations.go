package mfa

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the MFA plugin.
var PostgresMigrations = migrate.NewGroup("authsome-mfa", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the MFA plugin.
var SqliteMigrations = migrate.NewGroup("authsome-mfa", migrate.DependsOn("authsome"))

func init() {
	// ──────────────────────────────────────────────────
	// PostgreSQL migrations
	// ──────────────────────────────────────────────────

	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_mfa_tables",
			Version: "20240201000001",
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
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_mfa_recovery_codes;
DROP TABLE IF EXISTS authsome_mfa_enrollments;
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
			Name:    "create_mfa_tables",
			Version: "20240201000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_mfa_enrollments (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES authsome_users(id),
    method     TEXT NOT NULL,
    secret     TEXT NOT NULL,
    verified   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_mfa_enrollments_user_method
    ON authsome_mfa_enrollments (user_id, method);
CREATE INDEX IF NOT EXISTS idx_authsome_mfa_enrollments_user
    ON authsome_mfa_enrollments (user_id);

CREATE TABLE IF NOT EXISTS authsome_mfa_recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES authsome_users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,
    used       INTEGER NOT NULL DEFAULT 0,
    used_at    TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_authsome_mfa_recovery_codes_user
    ON authsome_mfa_recovery_codes (user_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_mfa_recovery_codes;
DROP TABLE IF EXISTS authsome_mfa_enrollments;
`)
				return err
			},
		},
	)
}
