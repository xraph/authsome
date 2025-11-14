package bun

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		// Up migration: Convert organization_id to app_id
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Migrating notification tables from organization_id to app_id...")

			// Step 1: Add new app_id columns to both tables
			fmt.Println("  Adding app_id columns...")
			
			// Add app_id to notification_templates
			_, err := db.ExecContext(ctx, `
				ALTER TABLE notification_templates 
				ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)
			`)
			if err != nil {
				return fmt.Errorf("failed to add app_id column to notification_templates: %w", err)
			}

			// Add app_id to notifications
			_, err = db.ExecContext(ctx, `
				ALTER TABLE notifications 
				ADD COLUMN IF NOT EXISTS app_id VARCHAR(20)
			`)
			if err != nil {
				return fmt.Errorf("failed to add app_id column to notifications: %w", err)
			}

			// Step 2: Get platform app ID
			fmt.Println("  Finding platform app...")
			var platformAppID string
			err = db.NewSelect().
				Table("apps").
				Column("id").
				Where("is_platform = ?", true).
				Limit(1).
				Scan(ctx, &platformAppID)
			if err != nil {
				return fmt.Errorf("failed to find platform app: %w", err)
			}

			// Step 3: Migrate existing data - all notifications belong to platform app
			fmt.Println("  Migrating existing notification templates to platform app...")
			_, err = db.ExecContext(ctx, `
				UPDATE notification_templates 
				SET app_id = ? 
				WHERE app_id IS NULL
			`, platformAppID)
			if err != nil {
				return fmt.Errorf("failed to migrate notification_templates data: %w", err)
			}

			fmt.Println("  Migrating existing notifications to platform app...")
			_, err = db.ExecContext(ctx, `
				UPDATE notifications 
				SET app_id = ? 
				WHERE app_id IS NULL
			`, platformAppID)
			if err != nil {
				return fmt.Errorf("failed to migrate notifications data: %w", err)
			}

			// Step 4: Make app_id NOT NULL
			fmt.Println("  Making app_id columns NOT NULL...")
			_, err = db.ExecContext(ctx, `
				ALTER TABLE notification_templates 
				ALTER COLUMN app_id SET NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to set app_id NOT NULL in notification_templates: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				ALTER TABLE notifications 
				ALTER COLUMN app_id SET NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to set app_id NOT NULL in notifications: %w", err)
			}

			// Step 5: Drop old indexes
			fmt.Println("  Dropping old indexes...")
			_, _ = db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_notification_templates_org_key`)
			_, _ = db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_notifications_org_status`)

			// Step 6: Create new indexes
			fmt.Println("  Creating new indexes...")
			_, err = db.ExecContext(ctx, `
				CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_templates_app_key 
				ON notification_templates (app_id, template_key, type, language)
				WHERE deleted_at IS NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to create idx_notification_templates_app_key: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_notifications_app_status 
				ON notifications (app_id, status)
			`)
			if err != nil {
				return fmt.Errorf("failed to create idx_notifications_app_status: %w", err)
			}

			// Step 7: Drop old organization_id columns
			fmt.Println("  Dropping old organization_id columns...")
			_, err = db.ExecContext(ctx, `
				ALTER TABLE notification_templates 
				DROP COLUMN IF EXISTS organization_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop organization_id from notification_templates: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				ALTER TABLE notifications 
				DROP COLUMN IF EXISTS organization_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop organization_id from notifications: %w", err)
			}

			fmt.Println("✓ Successfully migrated notification tables from organization_id to app_id")
			return nil
		},

		// Down migration: Revert app_id back to organization_id
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Rolling back notification tables from app_id to organization_id...")

			// Step 1: Add organization_id columns back
			fmt.Println("  Adding organization_id columns...")
			_, err := db.ExecContext(ctx, `
				ALTER TABLE notification_templates 
				ADD COLUMN IF NOT EXISTS organization_id VARCHAR(255)
			`)
			if err != nil {
				return fmt.Errorf("failed to add organization_id column to notification_templates: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				ALTER TABLE notifications 
				ADD COLUMN IF NOT EXISTS organization_id VARCHAR(255)
			`)
			if err != nil {
				return fmt.Errorf("failed to add organization_id column to notifications: %w", err)
			}

			// Step 2: Migrate data back - use "default" as organization_id
			fmt.Println("  Migrating data back to organization_id...")
			_, err = db.ExecContext(ctx, `
				UPDATE notification_templates 
				SET organization_id = 'default' 
				WHERE organization_id IS NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to migrate notification_templates data back: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				UPDATE notifications 
				SET organization_id = 'default' 
				WHERE organization_id IS NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to migrate notifications data back: %w", err)
			}

			// Step 3: Make organization_id NOT NULL
			fmt.Println("  Making organization_id columns NOT NULL...")
			_, err = db.ExecContext(ctx, `
				ALTER TABLE notification_templates 
				ALTER COLUMN organization_id SET NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to set organization_id NOT NULL in notification_templates: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				ALTER TABLE notifications 
				ALTER COLUMN organization_id SET NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to set organization_id NOT NULL in notifications: %w", err)
			}

			// Step 4: Drop new indexes
			fmt.Println("  Dropping new indexes...")
			_, _ = db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_notification_templates_app_key`)
			_, _ = db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_notifications_app_status`)

			// Step 5: Recreate old indexes
			fmt.Println("  Recreating old indexes...")
			_, err = db.ExecContext(ctx, `
				CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_templates_org_key 
				ON notification_templates (organization_id, template_key, type, language)
			`)
			if err != nil {
				return fmt.Errorf("failed to create idx_notification_templates_org_key: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_notifications_org_status 
				ON notifications (organization_id, status)
			`)
			if err != nil {
				return fmt.Errorf("failed to create idx_notifications_org_status: %w", err)
			}

			// Step 6: Drop app_id columns
			fmt.Println("  Dropping app_id columns...")
			_, err = db.ExecContext(ctx, `
				ALTER TABLE notification_templates 
				DROP COLUMN IF EXISTS app_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop app_id from notification_templates: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				ALTER TABLE notifications 
				DROP COLUMN IF EXISTS app_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop app_id from notifications: %w", err)
			}

			fmt.Println("✓ Successfully rolled back notification tables from app_id to organization_id")
			return nil
		},
	)
}

