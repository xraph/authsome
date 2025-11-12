package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("[Migration 015] Starting impersonation V2 architecture migration...")

		// =================================================================
		// PHASE 1: Add new columns
		// =================================================================
		fmt.Println("[Migration 015] Phase 1: Adding organization_id columns...")

		// Add organization_id to impersonation_sessions
		_, err := db.ExecContext(ctx, `
			ALTER TABLE impersonation_sessions 
			ADD COLUMN IF NOT EXISTS organization_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add organization_id to impersonation_sessions: %w", err)
		}
		fmt.Println("[Migration 015] ✅ Added organization_id to impersonation_sessions")

		// Add organization_id to impersonation_audit
		_, err = db.ExecContext(ctx, `
			ALTER TABLE impersonation_audit 
			ADD COLUMN IF NOT EXISTS organization_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add organization_id to impersonation_audit: %w", err)
		}
		fmt.Println("[Migration 015] ✅ Added organization_id to impersonation_audit")

		// =================================================================
		// PHASE 2: Add indexes for new columns
		// =================================================================
		fmt.Println("[Migration 015] Phase 2: Adding indexes...")

		// Index on impersonation_sessions.organization_id (for filtering)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_user_org 
			ON impersonation_sessions(organization_id) 
			WHERE organization_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create index on impersonation_sessions.organization_id: %w", err)
		}
		fmt.Println("[Migration 015] ✅ Created index on impersonation_sessions.organization_id")

		// Composite index on impersonation_sessions (organization_id, organization_id) for queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_app_org 
			ON impersonation_sessions(organization_id, organization_id) 
			WHERE organization_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create composite index: %w", err)
		}
		fmt.Println("[Migration 015] ✅ Created composite index on impersonation_sessions")

		// Index on impersonation_audit.organization_id
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_impersonation_audit_user_org 
			ON impersonation_audit(organization_id) 
			WHERE organization_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create index on impersonation_audit.organization_id: %w", err)
		}
		fmt.Println("[Migration 015] ✅ Created index on impersonation_audit.organization_id")

		// =================================================================
		// PHASE 3: Add foreign key constraints (optional, for referential integrity)
		// =================================================================
		fmt.Println("[Migration 015] Phase 3: Adding foreign key constraints...")

		// Foreign key: impersonation_sessions.organization_id → organizations.id
		_, err = db.ExecContext(ctx, `
			DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM information_schema.table_constraints 
					WHERE constraint_name = 'fk_impersonation_sessions_user_org'
				) THEN
					ALTER TABLE impersonation_sessions 
					ADD CONSTRAINT fk_impersonation_sessions_user_org 
					FOREIGN KEY (organization_id) 
					REFERENCES organizations(id) 
					ON DELETE CASCADE;
				END IF;
			END $$;
		`)
		if err != nil {
			// Log warning but don't fail (organizations table might not exist yet)
			fmt.Printf("[Migration 015] ⚠️  Could not add FK constraint (organizations table may not exist): %v\n", err)
		} else {
			fmt.Println("[Migration 015] ✅ Added foreign key constraint on impersonation_sessions")
		}

		// Foreign key: impersonation_audit.organization_id → organizations.id
		_, err = db.ExecContext(ctx, `
			DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM information_schema.table_constraints 
					WHERE constraint_name = 'fk_impersonation_audit_user_org'
				) THEN
					ALTER TABLE impersonation_audit 
					ADD CONSTRAINT fk_impersonation_audit_user_org 
					FOREIGN KEY (organization_id) 
					REFERENCES organizations(id) 
					ON DELETE CASCADE;
				END IF;
			END $$;
		`)
		if err != nil {
			fmt.Printf("[Migration 015] ⚠️  Could not add FK constraint for audit table: %v\n", err)
		} else {
			fmt.Println("[Migration 015] ✅ Added foreign key constraint on impersonation_audit")
		}

		// =================================================================
		// COMPLETE
		// =================================================================
		fmt.Println("[Migration 015] ✅ Impersonation V2 architecture migration complete!")
		fmt.Println("[Migration 015] Note: Column 'organization_id' now represents 'app_id' (platform app)")
		fmt.Println("[Migration 015] Note: Column 'organization_id' represents user-created organizations")
		return nil

	}, func(ctx context.Context, db *bun.DB) error {
		// =================================================================
		// Rollback 015: Impersonation V2 Architecture
		// =================================================================
		fmt.Println("[Rollback 015] Starting impersonation V2 architecture rollback...")

		// Drop foreign key constraints
		fmt.Println("[Rollback 015] Phase 1: Dropping foreign key constraints...")
		_, err := db.ExecContext(ctx, `
			ALTER TABLE impersonation_sessions 
			DROP CONSTRAINT IF EXISTS fk_impersonation_sessions_user_org
		`)
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop FK constraint: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			ALTER TABLE impersonation_audit 
			DROP CONSTRAINT IF EXISTS fk_impersonation_audit_user_org
		`)
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop FK constraint: %v\n", err)
		}

		// Drop indexes
		fmt.Println("[Rollback 015] Phase 2: Dropping indexes...")
		_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_impersonation_sessions_user_org")
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop index: %v\n", err)
		}

		_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_impersonation_sessions_app_org")
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop index: %v\n", err)
		}

		_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_impersonation_audit_user_org")
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop index: %v\n", err)
		}

		// Drop columns
		fmt.Println("[Rollback 015] Phase 3: Dropping organization_id columns...")
		_, err = db.ExecContext(ctx, "ALTER TABLE impersonation_sessions DROP COLUMN IF EXISTS organization_id")
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop column: %v\n", err)
		}

		_, err = db.ExecContext(ctx, "ALTER TABLE impersonation_audit DROP COLUMN IF EXISTS organization_id")
		if err != nil {
			fmt.Printf("[Rollback 015] Warning: Could not drop column: %v\n", err)
		}

		fmt.Println("[Rollback 015] ✅ Rollback complete")
		return nil
	})
}
