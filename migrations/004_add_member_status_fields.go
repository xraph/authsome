package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add status column to members table
		_, err := db.ExecContext(ctx, `
			ALTER TABLE members
			ADD COLUMN IF NOT EXISTS status VARCHAR DEFAULT 'active'
		`)
		if err != nil {
			return fmt.Errorf("failed to add status column: %w", err)
		}

		// Add joined_at column to members table
		_, err = db.ExecContext(ctx, `
			ALTER TABLE members
			ADD COLUMN IF NOT EXISTS joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		`)
		if err != nil {
			return fmt.Errorf("failed to add joined_at column: %w", err)
		}

		// Set default values for existing records
		_, err = db.ExecContext(ctx, `
			UPDATE members
			SET status = 'active'
			WHERE status IS NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to set default status: %w", err)
		}

		_, err = db.ExecContext(ctx, `
			UPDATE members
			SET joined_at = created_at
			WHERE joined_at IS NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to set default joined_at: %w", err)
		}

		// Make columns NOT NULL after setting defaults
		_, err = db.ExecContext(ctx, `
			ALTER TABLE members
			ALTER COLUMN status SET NOT NULL,
			ALTER COLUMN joined_at SET NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to set columns as NOT NULL: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback
		_, err := db.ExecContext(ctx, `
			ALTER TABLE members
			DROP COLUMN IF EXISTS status,
			DROP COLUMN IF EXISTS joined_at
		`)
		return err
	})
}
