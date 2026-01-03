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

		// Check if organization_id column already exists
		var colExists bool
		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'audit_events' AND column_name = 'organization_id'
			)
		`).Scan(ctx, &colExists)

		if err != nil {
			return fmt.Errorf("failed to check if organization_id column exists: %w", err)
		}

		if colExists {
			// Column already exists, skip migration
			return nil
		}

		// Add organization_id column (nullable for backward compatibility)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			ADD COLUMN organization_id VARCHAR(20) NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to add organization_id column: %w", err)
		}

		// Add foreign key constraint
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			ADD CONSTRAINT fk_audit_events_organization 
			FOREIGN KEY (organization_id) 
			REFERENCES organizations(id) 
			ON DELETE SET NULL
		`)
		if err != nil {
			// Log but don't fail if foreign key constraint can't be added
			// (organizations table might not exist in all setups)
			fmt.Printf("Warning: failed to add foreign key constraint for organization_id: %v\n", err)
		}

		// Add index for performance
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_organization_id 
			ON audit_events(organization_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create index on organization_id: %w", err)
		}

		// Add composite index for app_id + organization_id (common query pattern)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_app_org 
			ON audit_events(app_id, organization_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create composite index for app_org: %w", err)
		}

		// Add composite index for organization_id + created_at (for time-based queries by org)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_events_org_created 
			ON audit_events(organization_id, created_at DESC)
		`)
		if err != nil {
			return fmt.Errorf("failed to create composite index for org_created: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - remove organization_id column

		// Check if audit_events table exists
		var tableExists bool
		err := db.NewSelect().
			ColumnExpr("to_regclass(?) IS NOT NULL", "public.audit_events").
			Scan(ctx, &tableExists)

		if err != nil || !tableExists {
			return nil
		}

		// Check if organization_id column exists
		var colExists bool
		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'audit_events' AND column_name = 'organization_id'
			)
		`).Scan(ctx, &colExists)

		if err != nil || !colExists {
			return nil
		}

		// Drop indexes
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_organization_id
		`)
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_app_org
		`)
		_, _ = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_audit_events_org_created
		`)

		// Drop foreign key constraint
		_, _ = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			DROP CONSTRAINT IF EXISTS fk_audit_events_organization
		`)

		// Drop column
		_, err = db.ExecContext(ctx, `
			ALTER TABLE audit_events 
			DROP COLUMN IF EXISTS organization_id
		`)
		if err != nil {
			return fmt.Errorf("failed to drop organization_id column: %w", err)
		}

		return nil
	})
}

