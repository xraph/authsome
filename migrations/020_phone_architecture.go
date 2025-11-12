package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Phone V2 Architecture Migration
		// Adds app_id and user_organization_id to phone_verifications table

		// Add app_id column (required)
		_, err := db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			ADD COLUMN IF NOT EXISTS app_id VARCHAR(20) NOT NULL DEFAULT 'default'
		`)
		if err != nil {
			return fmt.Errorf("failed to add app_id column: %w", err)
		}

		// Add user_organization_id column (optional)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			ADD COLUMN IF NOT EXISTS user_organization_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add user_organization_id column: %w", err)
		}

		// Create index on (app_id, user_organization_id, phone) for efficient lookups
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_phone_verifications_app_org_phone 
			ON phone_verifications (app_id, user_organization_id, phone)
		`)
		if err != nil {
			return fmt.Errorf("failed to create app_org_phone index: %w", err)
		}

		// Create index on (app_id, expires_at) for cleanup queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_phone_verifications_app_expires 
			ON phone_verifications (app_id, expires_at)
		`)
		if err != nil {
			return fmt.Errorf("failed to create app_expires index: %w", err)
		}

		// Add foreign key constraint for app_id
		_, err = db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			ADD CONSTRAINT fk_phone_verifications_app_id 
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		`)
		if err != nil {
			// Log but don't fail - foreign keys may not be supported
			fmt.Printf("Warning: failed to add app_id foreign key: %v\n", err)
		}

		// Add foreign key constraint for user_organization_id
		_, err = db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			ADD CONSTRAINT fk_phone_verifications_user_org_id 
			FOREIGN KEY (user_organization_id) REFERENCES organizations(id) ON DELETE CASCADE
		`)
		if err != nil {
			// Log but don't fail - foreign keys may not be supported
			fmt.Printf("Warning: failed to add user_organization_id foreign key: %v\n", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove V2 columns and restore original schema

		// Drop foreign key constraints
		_, err := db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			DROP CONSTRAINT IF EXISTS fk_phone_verifications_app_id
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop app_id foreign key: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			DROP CONSTRAINT IF EXISTS fk_phone_verifications_user_org_id
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop user_organization_id foreign key: %v\n", err)
		}

		// Drop indexes
		_, err = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_phone_verifications_app_org_phone
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop app_org_phone index: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_phone_verifications_app_expires
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop app_expires index: %v\n", err)
		}

		// Drop V2 columns
		_, err = db.ExecContext(ctx, `
			ALTER TABLE phone_verifications 
			DROP COLUMN IF EXISTS app_id,
			DROP COLUMN IF EXISTS user_organization_id
		`)
		if err != nil {
			return fmt.Errorf("failed to drop V2 columns: %w", err)
		}

		return nil
	})
}
