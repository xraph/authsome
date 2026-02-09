package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/enterprise/compliance"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Register compliance models
		compliance.RegisterModels(db)

		// Create all compliance tables with app_id fields
		if err := compliance.CreateTables(ctx, db); err != nil {
			return fmt.Errorf("failed to create compliance tables: %w", err)
		}

		fmt.Println("✓ Successfully created compliance tables")

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Drop all compliance tables
		if err := compliance.DropTables(ctx, db); err != nil {
			return fmt.Errorf("failed to drop compliance tables: %w", err)
		}

		return nil
	})
}
