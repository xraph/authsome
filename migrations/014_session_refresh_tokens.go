package migrations

import (
	"context"
	"strings"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add refresh token columns to sessions table (idempotent)
		// These columns may already exist if the table was created from the current schema

		columns := []struct {
			name string
			sql  string
		}{
			{"refresh_token", `ALTER TABLE sessions ADD COLUMN refresh_token TEXT`},
			{"refresh_token_expires_at", `ALTER TABLE sessions ADD COLUMN refresh_token_expires_at TIMESTAMP`},
			{"last_refreshed_at", `ALTER TABLE sessions ADD COLUMN last_refreshed_at TIMESTAMP`},
		}

		for _, col := range columns {
			if _, err := db.ExecContext(ctx, col.sql); err != nil {
				// Check if error is "column already exists" - this is expected if schema was created fresh
				errStr := strings.ToLower(err.Error())
				if strings.Contains(errStr, "already exists") ||
					strings.Contains(errStr, "duplicate column") {
					// Column already exists, skip
					continue
				}
				return err
			}
		}

		// Create unique index on refresh_token (for non-null values)
		indexQuery := `CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_refresh_token ON sessions(refresh_token) WHERE refresh_token IS NOT NULL`
		if _, err := db.ExecContext(ctx, indexQuery); err != nil {
			// For SQLite, the WHERE clause might not work, try without it
			_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_refresh_token ON sessions(refresh_token)`)
			if err != nil {
				// If index already exists, that's fine
				errStr := strings.ToLower(err.Error())
				if !strings.Contains(errStr, "already exists") {
					return err
				}
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove refresh token columns
		queries := []string{
			`DROP INDEX IF EXISTS idx_sessions_refresh_token`,
			`ALTER TABLE sessions DROP COLUMN IF EXISTS last_refreshed_at`,
			`ALTER TABLE sessions DROP COLUMN IF EXISTS refresh_token_expires_at`,
			`ALTER TABLE sessions DROP COLUMN IF EXISTS refresh_token`,
		}

		for _, query := range queries {
			if _, err := db.ExecContext(ctx, query); err != nil {
				// Ignore errors on rollback for compatibility
				continue
			}
		}

		return nil
	})
}
