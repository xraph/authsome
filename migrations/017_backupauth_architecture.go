package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Backup-Auth V2 Architecture Migration
		// Adds app_id and organization_id to all backup-auth tables

		tables := []string{
			"backup_security_questions",
			"backup_trusted_contacts",
			"backup_recovery_sessions",
			"backup_video_sessions",
			"backup_document_verifications",
			"backup_recovery_logs",
			"backup_recovery_configs",
			"backup_code_usage",
		}

		for _, table := range tables {
			// Add app_id column
			_, err := db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				ADD COLUMN IF NOT EXISTS app_id VARCHAR(20) NOT NULL DEFAULT 'default'
			`, table))
			if err != nil {
				return fmt.Errorf("failed to add app_id to %s: %w", table, err)
			}

			// Add organization_id column (nullable)
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				ADD COLUMN IF NOT EXISTS organization_id VARCHAR(20)
			`, table))
			if err != nil {
				return fmt.Errorf("failed to add organization_id to %s: %w", table, err)
			}

			// Migrate existing organization_id to app_id
			if table != "backup_recovery_configs" { // recovery_configs has organization_id as unique key
				_, err = db.ExecContext(ctx, fmt.Sprintf(`
					UPDATE %s 
					SET app_id = organization_id 
					WHERE app_id = 'default' AND organization_id IS NOT NULL AND organization_id != ''
				`, table))
				if err != nil {
					return fmt.Errorf("failed to migrate organization_id to app_id in %s: %w", table, err)
				}
			} else {
				// Special handling for recovery_configs
				_, err = db.ExecContext(ctx, `
					UPDATE backup_recovery_configs 
					SET app_id = organization_id 
					WHERE organization_id IS NOT NULL AND organization_id != ''
				`)
				if err != nil {
					return fmt.Errorf("failed to migrate organization_id to app_id in backup_recovery_configs: %w", err)
				}
			}

			// Drop old organization_id column after migration
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				DROP COLUMN IF EXISTS organization_id
			`, table))
			if err != nil {
				return fmt.Errorf("failed to drop organization_id from %s: %w", table, err)
			}

			// Add index on app_id and organization_id
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				CREATE INDEX IF NOT EXISTS idx_%s_app_org 
				ON %s (app_id, organization_id)
			`, table, table))
			if err != nil {
				return fmt.Errorf("failed to create app_org index on %s: %w", table, err)
			}

			// Add foreign key constraint for app_id
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				ADD CONSTRAINT fk_%s_app_id 
				FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
			`, table, table))
			if err != nil {
				// Log but don't fail - foreign keys may not be supported or already exist
				fmt.Printf("Warning: failed to add app_id foreign key to %s: %v\n", table, err)
			}

			// Add foreign key constraint for organization_id
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				ADD CONSTRAINT fk_%s_user_org_id 
				FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
			`, table, table))
			if err != nil {
				// Log but don't fail - foreign keys may not be supported or already exist
				fmt.Printf("Warning: failed to add organization_id foreign key to %s: %v\n", table, err)
			}
		}

		// Update recovery_configs unique constraint
		_, err := db.ExecContext(ctx, `
			ALTER TABLE backup_recovery_configs 
			DROP CONSTRAINT IF EXISTS backup_recovery_configs_organization_id_key
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop old unique constraint: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_backup_recovery_configs_app_org_unique 
			ON backup_recovery_configs (app_id, COALESCE(organization_id, ''))
		`)
		if err != nil {
			return fmt.Errorf("failed to create unique index on backup_recovery_configs: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Restore organization_id column

		tables := []string{
			"backup_security_questions",
			"backup_trusted_contacts",
			"backup_recovery_sessions",
			"backup_video_sessions",
			"backup_document_verifications",
			"backup_recovery_logs",
			"backup_recovery_configs",
			"backup_code_usage",
		}

		for _, table := range tables {
			// Re-add organization_id column
			_, err := db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				ADD COLUMN IF NOT EXISTS organization_id VARCHAR(255) NOT NULL DEFAULT 'default'
			`, table))
			if err != nil {
				return fmt.Errorf("failed to re-add organization_id to %s: %w", table, err)
			}

			// Migrate app_id back to organization_id
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				UPDATE %s 
				SET organization_id = app_id 
				WHERE app_id IS NOT NULL AND app_id != ''
			`, table))
			if err != nil {
				return fmt.Errorf("failed to migrate app_id back to organization_id in %s: %w", table, err)
			}

			// Drop foreign key constraints
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				DROP CONSTRAINT IF EXISTS fk_%s_app_id
			`, table, table))
			if err != nil {
				fmt.Printf("Warning: failed to drop app_id foreign key from %s: %v\n", table, err)
			}

			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				DROP CONSTRAINT IF EXISTS fk_%s_user_org_id
			`, table, table))
			if err != nil {
				fmt.Printf("Warning: failed to drop organization_id foreign key from %s: %v\n", table, err)
			}

			// Drop indexes
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				DROP INDEX IF EXISTS idx_%s_app_org
			`, table))
			if err != nil {
				fmt.Printf("Warning: failed to drop app_org index from %s: %v\n", table, err)
			}

			// Drop new columns
			_, err = db.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s 
				DROP COLUMN IF EXISTS app_id,
				DROP COLUMN IF EXISTS organization_id
			`, table))
			if err != nil {
				return fmt.Errorf("failed to drop V2 columns from %s: %w", table, err)
			}
		}

		// Restore recovery_configs unique constraint
		_, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_backup_recovery_configs_app_org_unique
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop V2 unique index: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			ALTER TABLE backup_recovery_configs 
			ADD CONSTRAINT backup_recovery_configs_organization_id_key 
			UNIQUE (organization_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to restore unique constraint on backup_recovery_configs: %w", err)
		}

		return nil
	})
}
