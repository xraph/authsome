package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/xraph/authsome/internal/errs"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Detect database dialect
		isPostgres := db.Dialect().Name() == dialect.PG
		isSQLite := db.Dialect().Name() == dialect.SQLite

		// Helper function to check if table exists
		tableExists := func(tableName string) (bool, error) {
			var (
				exists bool
				query  string
			)

			if isPostgres {
				query = fmt.Sprintf("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '%s')", tableName)
			} else if isSQLite {
				query = fmt.Sprintf("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='%s'", tableName)
			} else {
				return false, fmt.Errorf("unsupported database dialect: %s", db.Dialect().Name())
			}

			err := db.NewRaw(query).Scan(ctx, &exists)

			return exists, err
		}

		// Helper function to check if column exists
		columnExists := func(tableName, columnName string) (bool, error) {
			var (
				exists bool
				query  string
			)

			if isPostgres {
				query = fmt.Sprintf("SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s')", tableName, columnName)
			} else if isSQLite {
				// For SQLite, we need to parse PRAGMA table_info
				var count int

				err := db.NewRaw(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'", tableName, columnName)).Scan(ctx, &count)

				return count > 0, err
			} else {
				return false, fmt.Errorf("unsupported database dialect: %s", db.Dialect().Name())
			}

			err := db.NewRaw(query).Scan(ctx, &exists)

			return exists, err
		}

		// Process attribute_mappings table
		// Always try to add columns if they don't exist (works even if table is new)
		if exists, err := tableExists("attribute_mappings"); err != nil {
			return fmt.Errorf("failed to check attribute_mappings table: %w", err)
		} else if exists {
			// Check if columns need to be added
			if colExists, err := columnExists("attribute_mappings", "app_id"); err != nil {
				return fmt.Errorf("failed to check app_id column: %w", err)
			} else if !colExists {
				// Add columns (syntax differs between PostgreSQL and SQLite)
				if isPostgres {
					_, err = db.ExecContext(ctx, `
						ALTER TABLE attribute_mappings 
						ADD COLUMN IF NOT EXISTS app_id VARCHAR(20),
						ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)
					`)
				} else if isSQLite {
					// SQLite doesn't support multiple ADD COLUMN in one statement
					_, err = db.ExecContext(ctx, `ALTER TABLE attribute_mappings ADD COLUMN app_id VARCHAR(20)`)
					if err != nil {
						return fmt.Errorf("failed to add app_id column: %w", err)
					}

					_, err = db.ExecContext(ctx, `ALTER TABLE attribute_mappings ADD COLUMN environment_id VARCHAR(20)`)
				}

				if err != nil {
					return fmt.Errorf("failed to add columns to attribute_mappings: %w", err)
				}

				// Set default values for existing rows
				_, err = db.ExecContext(ctx, `
					UPDATE attribute_mappings 
					SET app_id = 'default_app', 
					    environment_id = 'default_env'
					WHERE app_id IS NULL OR environment_id IS NULL
				`)
				if err != nil {
					return fmt.Errorf("failed to set default values: %w", err)
				}

				// Note: SQLite doesn't support ALTER COLUMN SET NOT NULL on existing columns
				// For new tables, the NOT NULL constraint is in the struct tags
				if isPostgres {
					_, err = db.ExecContext(ctx, `
						ALTER TABLE attribute_mappings 
						ALTER COLUMN app_id SET NOT NULL,
						ALTER COLUMN environment_id SET NOT NULL
					`)
					if err != nil {
						return fmt.Errorf("failed to set NOT NULL constraints: %w", err)
					}
				}

				// Create unique index
				_, err = db.ExecContext(ctx, `
					CREATE UNIQUE INDEX IF NOT EXISTS idx_attribute_mappings_app_env_org 
					ON attribute_mappings (app_id, environment_id, organization_id)
				`)
				if err != nil {
					return fmt.Errorf("failed to create unique index: %w", err)
				}
			}
		}

		// Process group_mappings table
		if exists, err := tableExists("group_mappings"); err != nil {
			return fmt.Errorf("failed to check group_mappings table: %w", err)
		} else if exists {
			if colExists, err := columnExists("group_mappings", "app_id"); err != nil {
				return fmt.Errorf("failed to check app_id column in group_mappings: %w", err)
			} else if !colExists {
				// Add columns
				if isPostgres {
					_, err = db.ExecContext(ctx, `
						ALTER TABLE group_mappings 
						ADD COLUMN IF NOT EXISTS app_id VARCHAR(20),
						ADD COLUMN IF NOT EXISTS environment_id VARCHAR(20)
					`)
				} else if isSQLite {
					_, err = db.ExecContext(ctx, `ALTER TABLE group_mappings ADD COLUMN app_id VARCHAR(20)`)
					if err != nil {
						return fmt.Errorf("failed to add app_id column to group_mappings: %w", err)
					}

					_, err = db.ExecContext(ctx, `ALTER TABLE group_mappings ADD COLUMN environment_id VARCHAR(20)`)
				}

				if err != nil {
					return fmt.Errorf("failed to add columns to group_mappings: %w", err)
				}

				// Set default values
				_, err = db.ExecContext(ctx, `
					UPDATE group_mappings 
					SET app_id = 'default_app', 
					    environment_id = 'default_env'
					WHERE app_id IS NULL OR environment_id IS NULL
				`)
				if err != nil {
					return fmt.Errorf("failed to set default values in group_mappings: %w", err)
				}

				// Set NOT NULL for PostgreSQL only
				if isPostgres {
					_, err = db.ExecContext(ctx, `
						ALTER TABLE group_mappings 
						ALTER COLUMN app_id SET NOT NULL,
						ALTER COLUMN environment_id SET NOT NULL
					`)
					if err != nil {
						return fmt.Errorf("failed to set NOT NULL constraints on group_mappings: %w", err)
					}
				}

				// Create indexes
				_, err = db.ExecContext(ctx, `
					CREATE INDEX IF NOT EXISTS idx_group_mappings_app_env_org 
					ON group_mappings (app_id, environment_id, organization_id)
				`)
				if err != nil {
					return fmt.Errorf("failed to create index on group_mappings: %w", err)
				}

				_, err = db.ExecContext(ctx, `
					CREATE UNIQUE INDEX IF NOT EXISTS idx_group_mappings_scim_group_id 
					ON group_mappings (app_id, environment_id, organization_id, scim_group_id)
				`)
				if err != nil {
					return fmt.Errorf("failed to create unique index on group_mappings: %w", err)
				}
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Remove the added columns
		// Note: SQLite doesn't support DROP COLUMN before version 3.35.0
		// This rollback may not work on older SQLite versions
		isPostgres := db.Dialect().Name() == dialect.PG

		if isPostgres {
			_, err := db.ExecContext(ctx, `
				ALTER TABLE attribute_mappings 
				DROP COLUMN IF EXISTS app_id,
				DROP COLUMN IF EXISTS environment_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop columns from attribute_mappings: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				ALTER TABLE group_mappings 
				DROP COLUMN IF EXISTS app_id,
				DROP COLUMN IF EXISTS environment_id
			`)
			if err != nil {
				return fmt.Errorf("failed to drop columns from group_mappings: %w", err)
			}
		} else {
			// For SQLite, we'd need to recreate the table without these columns
			// This is complex, so we'll just return an error for now
			return errs.InternalServerErrorWithMessage("rollback not supported for SQLite - manual intervention required")
		}

		return nil
	})
}
