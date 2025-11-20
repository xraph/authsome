package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add V2 context fields to identity_verifications table
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verifications
			ADD COLUMN IF NOT EXISTS app_id VARCHAR(20),
			ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to add V2 fields to identity_verifications: %w", err)
		}

		// Update existing records to have a default app_id (platform's default app)
		// In production, you may need to update this based on your actual default app ID
		if _, err := db.ExecContext(ctx, `
			UPDATE identity_verifications
			SET app_id = 'default_app_id'
			WHERE app_id IS NULL
		`); err != nil {
			return fmt.Errorf("failed to set default app_id for identity_verifications: %w", err)
		}

		// Make app_id NOT NULL after populating
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verifications
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			return fmt.Errorf("failed to make app_id NOT NULL in identity_verifications: %w", err)
		}

		// Add indexes for app_id filtering
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_iv_app_id ON identity_verifications(app_id)
		`); err != nil {
			return fmt.Errorf("failed to create app_id index on identity_verifications: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_iv_environment_id ON identity_verifications(environment_id)
		`); err != nil {
			return fmt.Errorf("failed to create environment_id index on identity_verifications: %w", err)
		}

		// Create composite index for efficient multi-tenant queries
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_iv_app_org_user ON identity_verifications(app_id, organization_id, user_id)
		`); err != nil {
			return fmt.Errorf("failed to create composite index on identity_verifications: %w", err)
		}

		// Update ID column types for consistency with xid format (20 chars)
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verifications
			ALTER COLUMN organization_id TYPE VARCHAR(20),
			ALTER COLUMN user_id TYPE VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to update ID column types in identity_verifications: %w", err)
		}

		// Add V2 context fields to identity_verification_sessions table
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_sessions
			ADD COLUMN IF NOT EXISTS app_id VARCHAR(20),
			ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to add V2 fields to identity_verification_sessions: %w", err)
		}

		// Update existing session records
		if _, err := db.ExecContext(ctx, `
			UPDATE identity_verification_sessions
			SET app_id = 'default_app_id'
			WHERE app_id IS NULL
		`); err != nil {
			return fmt.Errorf("failed to set default app_id for identity_verification_sessions: %w", err)
		}

		// Make app_id NOT NULL
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_sessions
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			return fmt.Errorf("failed to make app_id NOT NULL in identity_verification_sessions: %w", err)
		}

		// Add indexes
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_ivs_app_id ON identity_verification_sessions(app_id)
		`); err != nil {
			return fmt.Errorf("failed to create app_id index on identity_verification_sessions: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_ivs_environment_id ON identity_verification_sessions(environment_id)
		`); err != nil {
			return fmt.Errorf("failed to create environment_id index on identity_verification_sessions: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_ivs_app_org_user ON identity_verification_sessions(app_id, organization_id, user_id)
		`); err != nil {
			return fmt.Errorf("failed to create composite index on identity_verification_sessions: %w", err)
		}

		// Update ID column types
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_sessions
			ALTER COLUMN organization_id TYPE VARCHAR(20),
			ALTER COLUMN user_id TYPE VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to update ID column types in identity_verification_sessions: %w", err)
		}

		// Add V2 context fields to user_verification_status table
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE user_verification_status
			ADD COLUMN IF NOT EXISTS app_id VARCHAR(20),
			ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to add V2 fields to user_verification_status: %w", err)
		}

		// Update existing status records
		if _, err := db.ExecContext(ctx, `
			UPDATE user_verification_status
			SET app_id = 'default_app_id'
			WHERE app_id IS NULL
		`); err != nil {
			return fmt.Errorf("failed to set default app_id for user_verification_status: %w", err)
		}

		// Make app_id NOT NULL
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE user_verification_status
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			return fmt.Errorf("failed to make app_id NOT NULL in user_verification_status: %w", err)
		}

		// Add indexes
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_uvs_app_id ON user_verification_status(app_id)
		`); err != nil {
			return fmt.Errorf("failed to create app_id index on user_verification_status: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_uvs_environment_id ON user_verification_status(environment_id)
		`); err != nil {
			return fmt.Errorf("failed to create environment_id index on user_verification_status: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_uvs_app_org_user ON user_verification_status(app_id, organization_id, user_id)
		`); err != nil {
			return fmt.Errorf("failed to create composite index on user_verification_status: %w", err)
		}

		// Update ID column types
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE user_verification_status
			ALTER COLUMN organization_id TYPE VARCHAR(20),
			ALTER COLUMN user_id TYPE VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to update ID column types in user_verification_status: %w", err)
		}

		// Update the unique constraint on user_verification_status to include app_id
		// First drop the old constraint if it exists
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE user_verification_status
			DROP CONSTRAINT IF EXISTS user_verification_status_user_id_key
		`); err != nil {
			return fmt.Errorf("failed to drop old unique constraint on user_verification_status: %w", err)
		}

		// Create new composite unique constraint
		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_uvs_unique_app_org_user 
			ON user_verification_status(app_id, organization_id, user_id)
		`); err != nil {
			return fmt.Errorf("failed to create unique constraint on user_verification_status: %w", err)
		}

		// Add V2 context fields to identity_verification_documents table
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_documents
			ADD COLUMN IF NOT EXISTS app_id VARCHAR(20),
			ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20),
			ADD COLUMN IF NOT EXISTS organization_id VARCHAR(20)
		`); err != nil {
			return fmt.Errorf("failed to add V2 fields to identity_verification_documents: %w", err)
		}

		// Update existing document records from parent verification
		if _, err := db.ExecContext(ctx, `
			UPDATE identity_verification_documents ivd
			SET 
				app_id = iv.app_id,
				environment_id = iv.environment_id,
				organization_id = iv.organization_id
			FROM identity_verifications iv
			WHERE ivd.verification_id = iv.id
			AND ivd.app_id IS NULL
		`); err != nil {
			return fmt.Errorf("failed to populate V2 fields in identity_verification_documents: %w", err)
		}

		// Make required fields NOT NULL
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_documents
			ALTER COLUMN app_id SET NOT NULL,
			ALTER COLUMN organization_id SET NOT NULL
		`); err != nil {
			return fmt.Errorf("failed to make fields NOT NULL in identity_verification_documents: %w", err)
		}

		// Add indexes
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_ivd_app_id ON identity_verification_documents(app_id)
		`); err != nil {
			return fmt.Errorf("failed to create app_id index on identity_verification_documents: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_ivd_environment_id ON identity_verification_documents(environment_id)
		`); err != nil {
			return fmt.Errorf("failed to create environment_id index on identity_verification_documents: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_ivd_app_org ON identity_verification_documents(app_id, organization_id)
		`); err != nil {
			return fmt.Errorf("failed to create composite index on identity_verification_documents: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove V2 fields from all tables

		// identity_verifications
		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_iv_app_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_iv_environment_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_iv_app_org_user
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verifications
			DROP COLUMN IF EXISTS app_id,
			DROP COLUMN IF EXISTS environment_id
		`); err != nil {
			return fmt.Errorf("failed to remove V2 fields from identity_verifications: %w", err)
		}

		// identity_verification_sessions
		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_ivs_app_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_ivs_environment_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_ivs_app_org_user
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_sessions
			DROP COLUMN IF EXISTS app_id,
			DROP COLUMN IF EXISTS environment_id
		`); err != nil {
			return fmt.Errorf("failed to remove V2 fields from identity_verification_sessions: %w", err)
		}

		// user_verification_status
		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_uvs_app_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_uvs_environment_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_uvs_app_org_user
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_uvs_unique_app_org_user
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			ALTER TABLE user_verification_status
			DROP COLUMN IF EXISTS app_id,
			DROP COLUMN IF EXISTS environment_id
		`); err != nil {
			return fmt.Errorf("failed to remove V2 fields from user_verification_status: %w", err)
		}

		// identity_verification_documents
		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_ivd_app_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_ivd_environment_id
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_ivd_app_org
		`); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_documents
			DROP COLUMN IF EXISTS app_id,
			DROP COLUMN IF EXISTS environment_id,
			DROP COLUMN IF EXISTS organization_id
		`); err != nil {
			return fmt.Errorf("failed to remove V2 fields from identity_verification_documents: %w", err)
		}

		return nil
	})
}

