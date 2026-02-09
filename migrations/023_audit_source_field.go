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

		// Check if source column already exists
		var colExists bool

		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'audit_events' AND column_name = 'source'
			)
		`).Scan(ctx, &colExists)
		if err != nil {
			return fmt.Errorf("failed to check if source column exists: %w", err)
		}

		if colExists {
			// Column already exists, skip migration
			return nil
		}

		// Add source column (NOT NULL with default 'system')
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			ADD COLUMN source VARCHAR(20) NOT NULL DEFAULT 'system'
		`)
		if err != nil {
			return fmt.Errorf("failed to add source column: %w", err)
		}

		// Add index for filtering by source
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_source 
			ON audit_events(source)
		`)
		if err != nil {
			return fmt.Errorf("failed to create index on source: %w", err)
		}

		// Add composite index for app_id + source (common query pattern)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_app_source 
			ON audit_events(app_id, source)
		`)
		if err != nil {
			return fmt.Errorf("failed to create composite index for app_source: %w", err)
		}

		// Add composite index for source + created_at (for time-based queries by source)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_source_created 
			ON audit_events(source, created_at DESC)
		`)
		if err != nil {
			return fmt.Errorf("failed to create composite index for source_created: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - remove source column

		// Check if audit_events table exists
		var tableExists bool

		err := db.NewSelect().
			ColumnExpr("to_regclass(?) IS NOT NULL", "public.audit_events").
			Scan(ctx, &tableExists)

		if err != nil || !tableExists {
			return nil
		}

		// Check if source column exists
		var colExists bool

		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'audit_events' AND column_name = 'source'
			)
		`).Scan(ctx, &colExists)

		if err != nil || !colExists {
			return nil
		}

		// Drop indexes
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_source
		`)
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_app_source
		`)
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_source_created
		`)

		// Drop column
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			DROP COLUMN IF EXISTS source
		`)
		if err != nil {
			return fmt.Errorf("failed to drop source column: %w", err)
		}

		return nil
	})
}
