package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Passkey V2 Architecture Migration
		// Updates user_id to xid type and adds app_id and user_organization_id

		// Step 1: Add app_id column (required)
		_, err := db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ADD COLUMN IF NOT EXISTS app_id VARCHAR(20) NOT NULL DEFAULT 'default'
		`)
		if err != nil {
			return fmt.Errorf("failed to add app_id column: %w", err)
		}

		// Step 2: Add user_organization_id column (optional)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ADD COLUMN IF NOT EXISTS user_organization_id VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to add user_organization_id column: %w", err)
		}

		// Step 3: Update user_id column type from string to xid (varchar(20))
		// Note: This assumes existing user_id values are valid xids
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ALTER COLUMN user_id TYPE VARCHAR(20)
		`)
		if err != nil {
			return fmt.Errorf("failed to update user_id column type: %w", err)
		}

		// Step 4: Create composite index for efficient lookups
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_passkeys_user_app_org 
			ON passkeys (user_id, app_id, user_organization_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create user_app_org index: %w", err)
		}

		// Step 5: Create index on (app_id, user_organization_id) for org-scoped queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_passkeys_app_org 
			ON passkeys (app_id, user_organization_id)
		`)
		if err != nil {
			return fmt.Errorf("failed to create app_org index: %w", err)
		}

		// Step 6: Add foreign key constraint for app_id
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ADD CONSTRAINT fk_passkeys_app_id 
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		`)
		if err != nil {
			// Log but don't fail - foreign keys may not be supported
			fmt.Printf("Warning: failed to add app_id foreign key: %v\n", err)
		}

		// Step 7: Add foreign key constraint for user_organization_id
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ADD CONSTRAINT fk_passkeys_user_org_id 
			FOREIGN KEY (user_organization_id) REFERENCES organizations(id) ON DELETE CASCADE
		`)
		if err != nil {
			// Log but don't fail - foreign keys may not be supported
			fmt.Printf("Warning: failed to add user_organization_id foreign key: %v\n", err)
		}

		// Step 8: Add foreign key constraint for user_id (now xid type)
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ADD CONSTRAINT fk_passkeys_user_id 
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		`)
		if err != nil {
			// Log but don't fail - foreign keys may not be supported
			fmt.Printf("Warning: failed to add user_id foreign key: %v\n", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove V2 columns and restore original schema

		// Drop foreign key constraints
		_, err := db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			DROP CONSTRAINT IF EXISTS fk_passkeys_app_id
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop app_id foreign key: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			DROP CONSTRAINT IF EXISTS fk_passkeys_user_org_id
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop user_organization_id foreign key: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			DROP CONSTRAINT IF EXISTS fk_passkeys_user_id
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop user_id foreign key: %v\n", err)
		}

		// Drop indexes
		_, err = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_passkeys_user_app_org
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop user_app_org index: %v\n", err)
		}

		_, err = db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_passkeys_app_org
		`)
		if err != nil {
			fmt.Printf("Warning: failed to drop app_org index: %v\n", err)
		}

		// Revert user_id column type (back to text/string)
		// Note: This may lose data if user_id values aren't compatible
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			ALTER COLUMN user_id TYPE TEXT
		`)
		if err != nil {
			fmt.Printf("Warning: failed to revert user_id column type: %v\n", err)
		}

		// Drop V2 columns
		_, err = db.ExecContext(ctx, `
			ALTER TABLE passkeys 
			DROP COLUMN IF EXISTS app_id,
			DROP COLUMN IF EXISTS user_organization_id
		`)
		if err != nil {
			return fmt.Errorf("failed to drop V2 columns: %w", err)
		}

		return nil
	})
}
