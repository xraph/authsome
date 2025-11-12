package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("[Migration 014] API Key V2 Architecture: Adding App/Environment/Organization support...")

		// =================================================================
		// PHASE 1: Add new columns
		// =================================================================
		fmt.Println("[Migration 014] Phase 1: Adding new columns...")

		// Add app_id column (nullable initially)
		_, err := db.ExecContext(ctx, `
			ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add app_id column: %w", err)
		}

		// Add environment_id column (nullable)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add environment_id column: %w", err)
		}

		// Add organization_id column (nullable)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS organization_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add organization_id column: %w", err)
		}

		fmt.Println("[Migration 014] ✅ New columns added")

		// =================================================================
		// PHASE 2: Migrate existing data
		// =================================================================
		fmt.Println("[Migration 014] Phase 2: Migrating existing data...")

		// Check if org_id column exists before migration
		var orgIDExists bool
		err = db.NewRaw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'api_keys' 
				AND column_name = 'org_id'
			)
		`).Scan(ctx, &orgIDExists)
		if err != nil {
			return fmt.Errorf("failed to check for org_id column: %w", err)
		}

		// Migrate org_id → organization_id if column exists
		if orgIDExists {
			_, err = db.ExecContext(ctx, `
				UPDATE api_keys 
				SET organization_id = org_id 
				WHERE org_id IS NOT NULL AND organization_id IS NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to migrate org_id to organization_id: %w", err)
			}
			fmt.Println("[Migration 014] ✅ Migrated org_id to organization_id")
		}

		// Set default app for existing keys (get platform app)
		var platformAppID string
		err = db.NewRaw(`
			SELECT id FROM apps WHERE is_platform = true LIMIT 1
		`).Scan(ctx, &platformAppID)
		if err != nil {
			// If no platform app exists, try to get first app
			err = db.NewRaw(`SELECT id FROM apps LIMIT 1`).Scan(ctx, &platformAppID)
			if err != nil {
				return fmt.Errorf("failed to get platform app ID: %w", err)
			}
		}

		if platformAppID != "" {
			_, err = db.ExecContext(ctx, `
				UPDATE api_keys 
				SET app_id = $1 
				WHERE app_id IS NULL
			`, platformAppID)
			if err != nil {
				return fmt.Errorf("failed to set default app_id: %w", err)
			}
			fmt.Println("[Migration 014] ✅ Set default app_id for existing keys:", platformAppID)
		}

		fmt.Println("[Migration 014] ✅ Data migration complete")

		// =================================================================
		// PHASE 3: Add constraints
		// =================================================================
		fmt.Println("[Migration 014] Phase 3: Adding constraints...")

		// Make app_id NOT NULL
		_, err = db.ExecContext(ctx, `
			ALTER TABLE api_keys ALTER COLUMN app_id SET NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to make app_id NOT NULL: %w", err)
		}

		// Ensure user_id is VARCHAR(20) - may already be correct
		_, err = db.ExecContext(ctx, `
			DO $$ 
			BEGIN
				BEGIN
					ALTER TABLE api_keys ALTER COLUMN user_id TYPE VARCHAR(20);
				EXCEPTION 
					WHEN OTHERS THEN 
						-- Column may already be VARCHAR(20), ignore error
						NULL;
				END;
			END $$;
		`)
		if err != nil {
			// SQLite doesn't support this, ignore for SQLite
			fmt.Printf("[Migration 014] Warning: Could not alter user_id type: %v\n", err)
		}

		fmt.Println("[Migration 014] ✅ Constraints added")

		// =================================================================
		// PHASE 4: Drop old column
		// =================================================================
		fmt.Println("[Migration 014] Phase 4: Dropping old org_id column...")

		if orgIDExists {
			_, err = db.ExecContext(ctx, `
				ALTER TABLE api_keys DROP COLUMN IF EXISTS org_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop org_id column: %w", err)
			}
			fmt.Println("[Migration 014] ✅ Dropped org_id column")
		}

		// =================================================================
		// PHASE 5: Add indexes
		// =================================================================
		fmt.Println("[Migration 014] Phase 5: Adding indexes...")

		// Index on app_id
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_api_keys_app ON api_keys(app_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create idx_api_keys_app: %w", err)
		}

		// Index on environment_id (where not null)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_api_keys_env 
			ON api_keys(environment_id) 
			WHERE environment_id IS NOT NULL
		`)
		if err != nil {
			// Partial index may not be supported in all DB engines
			fmt.Printf("[Migration 014] Warning: Could not create partial index on environment_id: %v\n", err)
			// Try without WHERE clause
			_, err = db.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_api_keys_env ON api_keys(environment_id)
			`)
			if err != nil {
				fmt.Printf("[Migration 014] Warning: Could not create index on environment_id: %v\n", err)
			}
		}

		// Index on organization_id (where not null)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_api_keys_org 
			ON api_keys(organization_id) 
			WHERE organization_id IS NOT NULL
		`)
		if err != nil {
			// Partial index may not be supported in all DB engines
			fmt.Printf("[Migration 014] Warning: Could not create partial index on organization_id: %v\n", err)
			// Try without WHERE clause
			_, err = db.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(organization_id)
			`)
			if err != nil {
				fmt.Printf("[Migration 014] Warning: Could not create index on organization_id: %v\n", err)
			}
		}

		// Index on user_id
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create idx_api_keys_user: %w", err)
		}

		// Composite index on app_id + organization_id (where org not null)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_api_keys_app_org 
			ON api_keys(app_id, organization_id) 
			WHERE organization_id IS NOT NULL
		`)
		if err != nil {
			// Partial index may not be supported
			fmt.Printf("[Migration 014] Warning: Could not create partial composite index: %v\n", err)
			// Try without WHERE clause
			_, err = db.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_api_keys_app_org ON api_keys(app_id, organization_id)
			`)
			if err != nil {
				fmt.Printf("[Migration 014] Warning: Could not create composite index: %v\n", err)
			}
		}

		// Composite index on app_id + user_id
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_api_keys_app_user ON api_keys(app_id, user_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create idx_api_keys_app_user: %w", err)
		}

		fmt.Println("[Migration 014] ✅ Indexes added")
		fmt.Println("[Migration 014] 🎉 API Key V2 Architecture migration complete!")

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// =================================================================
		// ROLLBACK
		// =================================================================
		fmt.Println("[Rollback 014] Rolling back API Key V2 Architecture migration...")

		// Re-add org_id column
		_, err := db.ExecContext(ctx, `
			ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS org_id VARCHAR(255)
		`)
		if err != nil {
			fmt.Printf("[Rollback 014] Warning: Could not re-add org_id column: %v\n", err)
		}

		// Migrate organization_id back to org_id
		_, err = db.ExecContext(ctx, `
			UPDATE api_keys 
			SET org_id = organization_id 
			WHERE organization_id IS NOT NULL AND org_id IS NULL
		`)
		if err != nil {
			fmt.Printf("[Rollback 014] Warning: Could not migrate organization_id back to org_id: %v\n", err)
		}

		// Drop indexes
		indexesToDrop := []string{
			"idx_api_keys_app",
			"idx_api_keys_env",
			"idx_api_keys_org",
			"idx_api_keys_user",
			"idx_api_keys_app_org",
			"idx_api_keys_app_user",
		}
		for _, idx := range indexesToDrop {
			_, err = db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s", idx))
			if err != nil {
				fmt.Printf("[Rollback 014] Warning: Could not drop index %s: %v\n", idx, err)
			}
		}

		// Drop new columns
		columnsToRemove := []string{"app_id", "environment_id", "organization_id"}
		for _, col := range columnsToRemove {
			_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE api_keys DROP COLUMN IF EXISTS %s", col))
			if err != nil {
				fmt.Printf("[Rollback 014] Warning: Could not drop column %s: %v\n", col, err)
			}
		}

		fmt.Println("[Rollback 014] ✅ API Key V2 Architecture rollback complete")
		return nil
	})
}
