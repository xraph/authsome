package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add refresh token columns to sessions table
		queries := []string{
			// Add refresh_token column (nullable, unique for non-null values)
			`ALTER TABLE sessions ADD COLUMN refresh_token TEXT`,
			
			// Add refresh_token_expires_at column
			`ALTER TABLE sessions ADD COLUMN refresh_token_expires_at TIMESTAMP`,
			
			// Add last_refreshed_at column to track when token was last refreshed
			`ALTER TABLE sessions ADD COLUMN last_refreshed_at TIMESTAMP`,
			
			// Create unique index on refresh_token (for non-null values)
			// Different syntax for different databases
			`CREATE UNIQUE INDEX idx_sessions_refresh_token ON sessions(refresh_token) WHERE refresh_token IS NOT NULL`,
		}

		for _, query := range queries {
			if _, err := db.ExecContext(ctx, query); err != nil {
				// For SQLite, the WHERE clause might not work, try without it
				if query == `CREATE UNIQUE INDEX idx_sessions_refresh_token ON sessions(refresh_token) WHERE refresh_token IS NOT NULL` {
					_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX idx_sessions_refresh_token ON sessions(refresh_token)`)
					if err != nil {
						return err
					}
					continue
				}
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove refresh token columns
		queries := []string{
			`DROP INDEX IF EXISTS idx_sessions_refresh_token`,
			`ALTER TABLE sessions DROP COLUMN last_refreshed_at`,
			`ALTER TABLE sessions DROP COLUMN refresh_token_expires_at`,
			`ALTER TABLE sessions DROP COLUMN refresh_token`,
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

