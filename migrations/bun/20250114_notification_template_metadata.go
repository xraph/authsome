package bun

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

func init() {
	Migrations.Add(&migrate.Migration{
		Name: "20250114_notification_template_metadata",
		Up: func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Adding template metadata fields to notification_templates table...")

			// Add is_default column
			_, err := db.ExecContext(ctx, `ALTER TABLE notification_templates ADD COLUMN IF NOT EXISTS is_default BOOLEAN NOT NULL DEFAULT FALSE;`)
			if err != nil {
				return fmt.Errorf("failed to add is_default column: %w", err)
			}

			// Add is_modified column
			_, err = db.ExecContext(ctx, `ALTER TABLE notification_templates ADD COLUMN IF NOT EXISTS is_modified BOOLEAN NOT NULL DEFAULT FALSE;`)
			if err != nil {
				return fmt.Errorf("failed to add is_modified column: %w", err)
			}

			// Add default_hash column
			_, err = db.ExecContext(ctx, `ALTER TABLE notification_templates ADD COLUMN IF NOT EXISTS default_hash VARCHAR(64);`)
			if err != nil {
				return fmt.Errorf("failed to add default_hash column: %w", err)
			}

			fmt.Println("Template metadata fields added successfully.")
			return nil
		},
		Down: func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Removing template metadata fields from notification_templates table...")

			// Drop default_hash column
			_, err := db.ExecContext(ctx, `ALTER TABLE notification_templates DROP COLUMN IF EXISTS default_hash;`)
			if err != nil {
				return fmt.Errorf("failed to drop default_hash column: %w", err)
			}

			// Drop is_modified column
			_, err = db.ExecContext(ctx, `ALTER TABLE notification_templates DROP COLUMN IF EXISTS is_modified;`)
			if err != nil {
				return fmt.Errorf("failed to drop is_modified column: %w", err)
			}

			// Drop is_default column
			_, err = db.ExecContext(ctx, `ALTER TABLE notification_templates DROP COLUMN IF EXISTS is_default;`)
			if err != nil {
				return fmt.Errorf("failed to drop is_default column: %w", err)
			}

			fmt.Println("Template metadata fields removed successfully.")
			return nil
		},
	})
}

