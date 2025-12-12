package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add provider sync fields to subscription_features table
		_, err := db.ExecContext(ctx, `
			ALTER TABLE subscription_features 
			ADD COLUMN IF NOT EXISTS provider_feature_id VARCHAR(255),
			ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMP
		`)
		if err != nil {
			return fmt.Errorf("failed to add feature sync fields: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: remove the columns
		_, err := db.ExecContext(ctx, `
			ALTER TABLE subscription_features 
			DROP COLUMN IF EXISTS provider_feature_id,
			DROP COLUMN IF EXISTS last_synced_at
		`)
		if err != nil {
			return fmt.Errorf("failed to remove feature sync fields: %w", err)
		}

		return nil
	})
}

