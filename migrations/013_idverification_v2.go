package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create identity verification tables if they don't exist
		// These tables might not exist if upgrading from a version without ID verification plugin

		// Create identity_verifications table
		if _, err := db.NewCreateTable().
			Model((*schema.IdentityVerification)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create identity_verifications table: %w", err)
		}

		// Create identity_verification_sessions table
		if _, err := db.NewCreateTable().
			Model((*schema.IdentityVerificationSession)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create identity_verification_sessions table: %w", err)
		}

		// Create user_verification_status table
		if _, err := db.NewCreateTable().
			Model((*schema.UserVerificationStatus)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_verification_status table: %w", err)
		}

		// Create identity_verification_documents table
		if _, err := db.NewCreateTable().
			Model((*schema.IdentityVerificationDocument)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create identity_verification_documents table: %w", err)
		}

		// If tables already existed (old version), add V2 fields
		// Otherwise, they were created with V2 fields already

		// Add V2 context fields to identity_verifications table (for existing tables)
		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verifications ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add app_id to identity_verifications: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verifications ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add environment_id to identity_verifications: %w", err)
		}

		// Update existing records to have a default app_id (only for upgraded databases)
		// For newly created tables, records will have app_id from the schema
		// Check if there are any rows with NULL app_id before updating
		nullCount, err := db.NewSelect().
			Table("identity_verifications").
			Where("app_id IS NULL").
			Count(ctx)
		if err == nil && nullCount > 0 {
			// Only update if there are actually NULL values
			if _, err := db.ExecContext(ctx, `
				UPDATE identity_verifications
				SET app_id = (SELECT id FROM apps LIMIT 1)
				WHERE app_id IS NULL
			`); err != nil {
				return fmt.Errorf("failed to set default app_id for identity_verifications: %w", err)
			}
		}

		// Make app_id NOT NULL after populating (idempotent - safe if already NOT NULL)
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verifications
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			// Ignore error if column is already NOT NULL
			// Some databases might error, but PostgreSQL handles this gracefully
			fmt.Printf("Warning: could not set app_id NOT NULL: %v\n", err)
		}

		// Add indexes for app_id filtering
		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index("idx_iv_app_id").
			Column("app_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create app_id index on identity_verifications: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index("idx_iv_environment_id").
			Column("environment_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create environment_id index on identity_verifications: %w", err)
		}

		// Create composite index for efficient multi-tenant queries
		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index("idx_iv_app_org_user").
			Column("app_id", "organization_id", "user_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create composite index on identity_verifications: %w", err)
		}

		// Update ID column types for consistency with xid format (20 chars)
		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verifications ALTER COLUMN organization_id TYPE VARCHAR(20)`); err != nil {
			// Ignore error if column doesn't exist or already has correct type
			fmt.Printf("Warning: could not update organization_id type: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verifications ALTER COLUMN user_id TYPE VARCHAR(20)`); err != nil {
			// Ignore error if column doesn't exist or already has correct type
			fmt.Printf("Warning: could not update user_id type: %v\n", err)
		}

		// Add V2 context fields to identity_verification_sessions table
		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_sessions ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add app_id to identity_verification_sessions: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_sessions ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add environment_id to identity_verification_sessions: %w", err)
		}

		// Update existing session records (only for upgraded databases)
		nullCount, err = db.NewSelect().
			Table("identity_verification_sessions").
			Where("app_id IS NULL").
			Count(ctx)
		if err == nil && nullCount > 0 {
			if _, err := db.ExecContext(ctx, `
				UPDATE identity_verification_sessions
				SET app_id = (SELECT id FROM apps LIMIT 1)
				WHERE app_id IS NULL
			`); err != nil {
				return fmt.Errorf("failed to set default app_id for identity_verification_sessions: %w", err)
			}
		}

		// Make app_id NOT NULL (idempotent)
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_sessions
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			fmt.Printf("Warning: could not set app_id NOT NULL on sessions: %v\n", err)
		}

		// Add indexes
		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index("idx_ivs_app_id").
			Column("app_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create app_id index on identity_verification_sessions: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index("idx_ivs_environment_id").
			Column("environment_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create environment_id index on identity_verification_sessions: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index("idx_ivs_app_org_user").
			Column("app_id", "organization_id", "user_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create composite index on identity_verification_sessions: %w", err)
		}

		// Update ID column types
		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_sessions ALTER COLUMN organization_id TYPE VARCHAR(20)`); err != nil {
			fmt.Printf("Warning: could not update organization_id type: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_sessions ALTER COLUMN user_id TYPE VARCHAR(20)`); err != nil {
			fmt.Printf("Warning: could not update user_id type: %v\n", err)
		}

		// Add V2 context fields to user_verification_status table
		if _, err := db.ExecContext(ctx, `ALTER TABLE user_verification_status ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add app_id to user_verification_status: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE user_verification_status ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add environment_id to user_verification_status: %w", err)
		}

		// Update existing status records (only for upgraded databases)
		nullCount, err = db.NewSelect().
			Table("user_verification_status").
			Where("app_id IS NULL").
			Count(ctx)
		if err == nil && nullCount > 0 {
			if _, err := db.ExecContext(ctx, `
				UPDATE user_verification_status
				SET app_id = (SELECT id FROM apps LIMIT 1)
				WHERE app_id IS NULL
			`); err != nil {
				return fmt.Errorf("failed to set default app_id for user_verification_status: %w", err)
			}
		}

		// Make app_id NOT NULL (idempotent)
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE user_verification_status
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			fmt.Printf("Warning: could not set app_id NOT NULL on status: %v\n", err)
		}

		// Add indexes
		if _, err := db.NewCreateIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_app_id").
			Column("app_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create app_id index on user_verification_status: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_environment_id").
			Column("environment_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create environment_id index on user_verification_status: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_app_org_user").
			Column("app_id", "organization_id", "user_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create composite index on user_verification_status: %w", err)
		}

		// Update ID column types
		if _, err := db.ExecContext(ctx, `ALTER TABLE user_verification_status ALTER COLUMN organization_id TYPE VARCHAR(20)`); err != nil {
			fmt.Printf("Warning: could not update organization_id type: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE user_verification_status ALTER COLUMN user_id TYPE VARCHAR(20)`); err != nil {
			fmt.Printf("Warning: could not update user_id type: %v\n", err)
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
		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_documents ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add app_id to identity_verification_documents: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_documents ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add environment_id to identity_verification_documents: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_documents ADD COLUMN IF NOT EXISTS organization_id VARCHAR(20)`); err != nil {
			return fmt.Errorf("failed to add organization_id to identity_verification_documents: %w", err)
		}

		// Update existing document records from parent verification (only for upgraded databases)
		nullCount, err = db.NewSelect().
			Table("identity_verification_documents").
			Where("app_id IS NULL").
			Count(ctx)
		if err == nil && nullCount > 0 {
			if _, err := db.ExecContext(ctx, `
				UPDATE identity_verification_documents ivd
				SET 
					app_id = COALESCE(iv.app_id, (SELECT id FROM apps LIMIT 1)),
					environment_id = iv.environment_id,
					organization_id = COALESCE(iv.organization_id, 'platform')
				FROM identity_verifications iv
				WHERE ivd.verification_id = iv.id
				AND ivd.app_id IS NULL
			`); err != nil {
				return fmt.Errorf("failed to populate V2 fields in identity_verification_documents: %w", err)
			}
		}

		// Make required fields NOT NULL (idempotent)
		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_documents
			ALTER COLUMN app_id SET NOT NULL
		`); err != nil {
			fmt.Printf("Warning: could not set app_id NOT NULL on documents: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `
			ALTER TABLE identity_verification_documents
			ALTER COLUMN organization_id SET NOT NULL
		`); err != nil {
			fmt.Printf("Warning: could not set organization_id NOT NULL on documents: %v\n", err)
		}

		// Add indexes
		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index("idx_ivd_app_id").
			Column("app_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create app_id index on identity_verification_documents: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index("idx_ivd_environment_id").
			Column("environment_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create environment_id index on identity_verification_documents: %w", err)
		}

		if _, err := db.NewCreateIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index("idx_ivd_app_org").
			Column("app_id", "organization_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create composite index on identity_verification_documents: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove V2 fields from all tables

		// identity_verifications
		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index("idx_iv_app_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index("idx_iv_environment_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index("idx_iv_app_org_user").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verifications DROP COLUMN IF EXISTS app_id`); err != nil {
			fmt.Printf("Warning: could not drop app_id from identity_verifications: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verifications DROP COLUMN IF EXISTS environment_id`); err != nil {
			fmt.Printf("Warning: could not drop environment_id from identity_verifications: %v\n", err)
		}

		// identity_verification_sessions
		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index("idx_ivs_app_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index("idx_ivs_environment_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index("idx_ivs_app_org_user").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_sessions DROP COLUMN IF EXISTS app_id`); err != nil {
			fmt.Printf("Warning: could not drop app_id from identity_verification_sessions: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_sessions DROP COLUMN IF EXISTS environment_id`); err != nil {
			fmt.Printf("Warning: could not drop environment_id from identity_verification_sessions: %v\n", err)
		}

		// user_verification_status
		if _, err := db.NewDropIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_app_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_environment_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_app_org_user").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index("idx_uvs_unique_app_org_user").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE user_verification_status DROP COLUMN IF EXISTS app_id`); err != nil {
			fmt.Printf("Warning: could not drop app_id from user_verification_status: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE user_verification_status DROP COLUMN IF EXISTS environment_id`); err != nil {
			fmt.Printf("Warning: could not drop environment_id from user_verification_status: %v\n", err)
		}

		// identity_verification_documents
		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index("idx_ivd_app_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index("idx_ivd_environment_id").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index("idx_ivd_app_org").
			IfExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_documents DROP COLUMN IF EXISTS app_id`); err != nil {
			fmt.Printf("Warning: could not drop app_id from identity_verification_documents: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_documents DROP COLUMN IF EXISTS environment_id`); err != nil {
			fmt.Printf("Warning: could not drop environment_id from identity_verification_documents: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, `ALTER TABLE identity_verification_documents DROP COLUMN IF EXISTS organization_id`); err != nil {
			fmt.Printf("Warning: could not drop organization_id from identity_verification_documents: %v\n", err)
		}

		return nil
	})
}
