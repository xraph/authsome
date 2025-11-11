package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add is_platform column to organizations table
		_, err := db.ExecContext(ctx, `
			ALTER TABLE organizations 
			ADD COLUMN IF NOT EXISTS is_platform BOOLEAN DEFAULT false
		`)
		if err != nil {
			return err
		}

		// Mark existing "platform" slug as the platform org (if it exists)
		_, err = db.ExecContext(ctx, `
			UPDATE organizations 
			SET is_platform = true 
			WHERE slug = 'platform' 
			AND NOT EXISTS (
				SELECT 1 FROM organizations WHERE is_platform = true
			)
		`)
		if err != nil {
			return err
		}

		// Add unique partial index to ensure only one platform org
		// This works on PostgreSQL and SQLite 3.15+
		_, err = db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_is_platform
			ON organizations (is_platform)
			WHERE is_platform = true
		`)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: remove the index and column
		_, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_organizations_is_platform
		`)
		if err != nil {
			return err
		}

		_, err = db.ExecContext(ctx, `
			ALTER TABLE organizations DROP COLUMN IF EXISTS is_platform
		`)
		return err
	})
}
