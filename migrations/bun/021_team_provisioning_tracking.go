package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/migrations"
)

func init() {
	migrations.Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add provisioning tracking fields to teams table
		teamsMigrations := []string{
			// Add provisioned_by field to track provisioning source (e.g., "scim")
			`ALTER TABLE teams ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,

			// Add external_id field to store external system identifier
			`ALTER TABLE teams ADD COLUMN IF NOT EXISTS external_id VARCHAR(255)`,

			// Create index on external_id for efficient lookups
			`CREATE INDEX IF NOT EXISTS idx_teams_external_id ON teams(external_id) WHERE external_id IS NOT NULL`,

			// Create index on provisioned_by for filtering
			`CREATE INDEX IF NOT EXISTS idx_teams_provisioned_by ON teams(provisioned_by) WHERE provisioned_by IS NOT NULL`,
		}

		// Add same fields to organization_teams table
		orgTeamsMigrations := []string{
			// Add provisioned_by field
			`ALTER TABLE organization_teams ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,

			// Add external_id field
			`ALTER TABLE organization_teams ADD COLUMN IF NOT EXISTS external_id VARCHAR(255)`,

			// Create index on external_id for efficient lookups
			`CREATE INDEX IF NOT EXISTS idx_organization_teams_external_id ON organization_teams(external_id) WHERE external_id IS NOT NULL`,

			// Create index on provisioned_by for filtering
			`CREATE INDEX IF NOT EXISTS idx_organization_teams_provisioned_by ON organization_teams(provisioned_by) WHERE provisioned_by IS NOT NULL`,
		}

		// Execute teams migrations
		for _, migration := range teamsMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute teams migration: %w", err)
			}
		}

		// Execute organization teams migrations
		for _, migration := range orgTeamsMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute organization_teams migration: %w", err)
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - remove the added columns and indexes
		teamsRollbacks := []string{
			`DROP INDEX IF EXISTS idx_teams_provisioned_by`,
			`DROP INDEX IF EXISTS idx_teams_external_id`,
			`ALTER TABLE teams DROP COLUMN IF EXISTS external_id`,
			`ALTER TABLE teams DROP COLUMN IF EXISTS provisioned_by`,
		}

		orgTeamsRollbacks := []string{
			`DROP INDEX IF EXISTS idx_organization_teams_provisioned_by`,
			`DROP INDEX IF EXISTS idx_organization_teams_external_id`,
			`ALTER TABLE organization_teams DROP COLUMN IF EXISTS external_id`,
			`ALTER TABLE organization_teams DROP COLUMN IF EXISTS provisioned_by`,
		}

		// Execute teams rollbacks
		for _, rollback := range teamsRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				// Log error but continue with other rollbacks
				fmt.Printf("teams rollback error: %v\n", err)
			}
		}

		// Execute organization teams rollbacks
		for _, rollback := range orgTeamsRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				// Log error but continue with other rollbacks
				fmt.Printf("organization_teams rollback error: %v\n", err)
			}
		}

		return nil
	})
}

