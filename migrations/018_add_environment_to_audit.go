package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Check if audit_events table exists
		var tableExists bool
		err := db.NewSelect().
			ColumnExpr("to_regclass(?) IS NOT NULL", "public.audit_events").
			Scan(ctx, &tableExists)

		if err != nil || !tableExists {
			// Table doesn't exist, skip migration
			return nil
		}

		// Check if environment_id column already exists
		var colExists bool
		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'audit_events' AND column_name = 'environment_id'
			)
		`).Scan(ctx, &colExists)

		if err != nil {
			return fmt.Errorf("failed to check if environment_id column exists: %w", err)
		}

		if colExists {
			// Column already exists, skip migration
			return nil
		}

		// Add environment_id column (nullable for backward compatibility)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			ADD COLUMN environment_id VARCHAR(20) NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to add environment_id column: %w", err)
		}

		// Add foreign key constraint
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			ADD CONSTRAINT fk_audit_events_environment 
			FOREIGN KEY (environment_id) 
			REFERENCES environments(id) 
			ON DELETE SET NULL
		`)
		if err != nil {
			// Log but don't fail if foreign key constraint can't be added
			// (environments table might not exist in all setups)
			fmt.Printf("Warning: failed to add foreign key constraint: %v\n", err)
		}

		// Add index for performance
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_environment_id 
			ON audit_events(environment_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create index on environment_id: %w", err)
		}

		// Add composite index for app_id + environment_id (common query pattern)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_app_env 
			ON audit_events(app_id, environment_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create composite index: %w", err)
		}

		// Optional: Backfill existing audit events with default environment for their app
		// This is a best-effort approach - only for apps with a default environment
		_, err = db.ExecContext(ctx, `
			UPDATE audit_events ae
			SET environment_id = (
				SELECT e.id 
				FROM environments e 
				WHERE e.app_id = ae.app_id 
				AND e.is_default = true 
				LIMIT 1
			)
			WHERE ae.environment_id IS NULL
		`)
		if err != nil {
			// Log but don't fail - backfill is optional
			fmt.Printf("Warning: failed to backfill environment_id: %v\n", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - remove environment_id column

		// Check if audit_events table exists
		var tableExists bool
		err := db.NewSelect().
			ColumnExpr("to_regclass(?) IS NOT NULL", "public.audit_events").
			Scan(ctx, &tableExists)

		if err != nil || !tableExists {
			return nil
		}

		// Check if environment_id column exists
		var colExists bool
		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'audit_events' AND column_name = 'environment_id'
			)
		`).Scan(ctx, &colExists)

		if err != nil || !colExists {
			return nil
		}

		// Drop indexes
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_environment_id
		`)
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_app_env
		`)

		// Drop foreign key constraint
		_, _ = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			DROP CONSTRAINT IF EXISTS fk_audit_events_environment
		`)

		// Drop column
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			DROP COLUMN IF EXISTS environment_id
		`)
		if err != nil {
			return fmt.Errorf("failed to drop environment_id column: %w", err)
		}

		return nil
	})
}
