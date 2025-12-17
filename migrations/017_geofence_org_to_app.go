package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		tables := []string{
			"geofence_rules",
			"location_events",
			"geofence_violations",
			"trusted_locations",
			"travel_alerts",
		}

		for _, table := range tables {
			// Check if table exists before trying to rename column
			var exists bool
			err := db.NewSelect().
				ColumnExpr("to_regclass(?) IS NOT NULL", "public."+table).
				Scan(ctx, &exists)

			if err != nil || !exists {
				// Table doesn't exist, skip it
				continue
			}

			// Check if organization_id column exists
			var colExists bool
			err = db.NewRaw(`
				SELECT EXISTS (
					SELECT 1 
					FROM information_schema.columns 
					WHERE table_name = ? AND column_name = 'organization_id'
				)
			`, table).Scan(ctx, &colExists)

			if err != nil || !colExists {
				// Column doesn't exist or already migrated, skip
				continue
			}

			// Rename column
			_, err = db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s RENAME COLUMN organization_id TO app_id",
				table,
			))
			if err != nil {
				return fmt.Errorf("failed to rename column in %s: %w", table, err)
			}

			// Update index name if exists
			_, _ = db.ExecContext(ctx, fmt.Sprintf(
				"ALTER INDEX IF EXISTS idx_%s_organization_id RENAME TO idx_%s_app_id",
				table, table,
			))
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - rename back to organization_id
		tables := []string{
			"geofence_rules",
			"location_events",
			"geofence_violations",
			"trusted_locations",
			"travel_alerts",
		}

		for _, table := range tables {
			// Check if table exists
			var exists bool
			err := db.NewSelect().
				ColumnExpr("to_regclass(?) IS NOT NULL", "public."+table).
				Scan(ctx, &exists)

			if err != nil || !exists {
				continue
			}

			// Check if app_id column exists
			var colExists bool
			err = db.NewRaw(`
				SELECT EXISTS (
					SELECT 1 
					FROM information_schema.columns 
					WHERE table_name = ? AND column_name = 'app_id'
				)
			`, table).Scan(ctx, &colExists)

			if err != nil || !colExists {
				continue
			}

			_, err = db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s RENAME COLUMN app_id TO organization_id",
				table,
			))
			if err != nil {
				return fmt.Errorf("failed to rollback column in %s: %w", table, err)
			}

			_, _ = db.ExecContext(ctx, fmt.Sprintf(
				"ALTER INDEX IF EXISTS idx_%s_app_id RENAME TO idx_%s_organization_id",
				table, table,
			))
		}

		return nil
	})
}
