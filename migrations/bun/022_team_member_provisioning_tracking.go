package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/migrations"
)

func init() {
	migrations.Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add provisioning tracking field to team_members table
		teamMembersMigrations := []string{
			// Add provisioned_by field to track provisioning source (e.g., "scim")
			`ALTER TABLE team_members ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,

			// Create index on provisioned_by for filtering
			`CREATE INDEX IF NOT EXISTS idx_team_members_provisioned_by ON team_members(provisioned_by) WHERE provisioned_by IS NOT NULL`,
		}

		// Add same field to organization_team_members table
		orgTeamMembersMigrations := []string{
			// Add provisioned_by field
			`ALTER TABLE organization_team_members ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,

			// Create index on provisioned_by for filtering
			`CREATE INDEX IF NOT EXISTS idx_organization_team_members_provisioned_by ON organization_team_members(provisioned_by) WHERE provisioned_by IS NOT NULL`,
		}

		// Execute team members migrations
		for _, migration := range teamMembersMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute team_members migration: %w", err)
			}
		}

		// Execute organization team members migrations
		for _, migration := range orgTeamMembersMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute organization_team_members migration: %w", err)
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - remove the added columns and indexes
		teamMembersRollbacks := []string{
			`DROP INDEX IF EXISTS idx_team_members_provisioned_by`,
			`ALTER TABLE team_members DROP COLUMN IF EXISTS provisioned_by`,
		}

		orgTeamMembersRollbacks := []string{
			`DROP INDEX IF EXISTS idx_organization_team_members_provisioned_by`,
			`ALTER TABLE organization_team_members DROP COLUMN IF EXISTS provisioned_by`,
		}

		// Execute team members rollbacks
		for _, rollback := range teamMembersRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				// Log error but continue with other rollbacks
				fmt.Printf("team_members rollback error: %v\n", err)
			}
		}

		// Execute organization team members rollbacks
		for _, rollback := range orgTeamMembersRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				// Log error but continue with other rollbacks
				fmt.Printf("organization_team_members rollback error: %v\n", err)
			}
		}

		return nil
	})
}

