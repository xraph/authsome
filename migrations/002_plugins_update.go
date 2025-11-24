package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Register m2m models before any table operations
		// These models are used as join tables for many-to-many relationships
		// and must be explicitly registered with Bun before operations on models with m2m relations
		db.RegisterModel((*schema.TeamMember)(nil))
		db.RegisterModel((*schema.OrganizationTeamMember)(nil))
		db.RegisterModel((*schema.RolePermission)(nil))
		db.RegisterModel((*schema.APIKeyRole)(nil))

		// ============================================================
		// Migration 1: Passkey WebAuthn Fields
		// ============================================================
		// Add new WebAuthn fields to passkeys table
		passkeyMigrations := []string{
			// Add public key field (required for WebAuthn verification)
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS public_key BYTEA`,

			// Add AAGUID field (authenticator identifier)
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS aaguid BYTEA`,

			// Add sign count for replay attack detection
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS sign_count INTEGER DEFAULT 0`,

			// Add authenticator type (platform vs cross-platform)
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS authenticator_type VARCHAR(50)`,

			// Add name field for user-friendly device naming
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS name VARCHAR(255)`,

			// Add resident key flag
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS is_resident_key BOOLEAN DEFAULT FALSE`,

			// Add last used timestamp
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP`,
		}

		for _, migration := range passkeyMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute passkey migration: %w", err)
			}
		}

		// ============================================================
		// Migration 2: Team Provisioning Tracking
		// ============================================================
		// Add provisioning tracking fields to teams table
		teamsMigrations := []string{
			// Add provisioned_by field to track provisioning source (e.g., "scim")
			`ALTER TABLE teams ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,

			// Add external_id field to store external system identifier
			`ALTER TABLE teams ADD COLUMN IF NOT EXISTS external_id VARCHAR(255)`,
		}

		// Execute teams migrations
		for _, migration := range teamsMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute teams migration: %w", err)
			}
		}

		// Create index on external_id for efficient lookups
		if _, err := db.NewCreateIndex().
			Model((*schema.Team)(nil)).
			Index("idx_teams_external_id").
			Column("external_id").
			Where("external_id IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create teams external_id index: %w", err)
		}

		// Create index on provisioned_by for filtering
		if _, err := db.NewCreateIndex().
			Model((*schema.Team)(nil)).
			Index("idx_teams_provisioned_by").
			Column("provisioned_by").
			Where("provisioned_by IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create teams provisioned_by index: %w", err)
		}

		// Add same fields to organization_teams table
		orgTeamsMigrations := []string{
			// Add provisioned_by field
			`ALTER TABLE organization_teams ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,

			// Add external_id field
			`ALTER TABLE organization_teams ADD COLUMN IF NOT EXISTS external_id VARCHAR(255)`,
		}

		// Execute organization teams migrations
		for _, migration := range orgTeamsMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute organization_teams migration: %w", err)
			}
		}

		// Create index on external_id for efficient lookups
		if _, err := db.NewCreateIndex().
			Model((*schema.OrganizationTeam)(nil)).
			Index("idx_organization_teams_external_id").
			Column("external_id").
			Where("external_id IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create organization_teams external_id index: %w", err)
		}

		// Create index on provisioned_by for filtering
		if _, err := db.NewCreateIndex().
			Model((*schema.OrganizationTeam)(nil)).
			Index("idx_organization_teams_provisioned_by").
			Column("provisioned_by").
			Where("provisioned_by IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create organization_teams provisioned_by index: %w", err)
		}

		// ============================================================
		// Migration 3: Team Member Provisioning Tracking
		// ============================================================
		// Add provisioning tracking field to team_members table
		teamMembersMigrations := []string{
			// Add provisioned_by field to track provisioning source (e.g., "scim")
			`ALTER TABLE team_members ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,
		}

		// Execute team members migrations
		for _, migration := range teamMembersMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute team_members migration: %w", err)
			}
		}

		// Create index on provisioned_by for filtering
		if _, err := db.NewCreateIndex().
			Model((*schema.TeamMember)(nil)).
			Index("idx_team_members_provisioned_by").
			Column("provisioned_by").
			Where("provisioned_by IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create team_members provisioned_by index: %w", err)
		}

		// Add same field to organization_team_members table
		orgTeamMembersMigrations := []string{
			// Add provisioned_by field
			`ALTER TABLE organization_team_members ADD COLUMN IF NOT EXISTS provisioned_by VARCHAR(50)`,
		}

		// Execute organization team members migrations
		for _, migration := range orgTeamMembersMigrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute organization_team_members migration: %w", err)
			}
		}

		// Create index on provisioned_by for filtering
		if _, err := db.NewCreateIndex().
			Model((*schema.OrganizationTeamMember)(nil)).
			Index("idx_organization_team_members_provisioned_by").
			Column("provisioned_by").
			Where("provisioned_by IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create organization_team_members provisioned_by index: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// ============================================================
		// Rollback Migration 3: Team Member Provisioning Tracking
		// ============================================================
		// Drop team members index
		if _, err := db.NewDropIndex().
			Model((*schema.TeamMember)(nil)).
			Index("idx_team_members_provisioned_by").
			IfExists().
			Exec(ctx); err != nil {
			fmt.Printf("team_members rollback error: %v\n", err)
		}

		// Drop team members column
		teamMembersRollbacks := []string{
			`ALTER TABLE team_members DROP COLUMN IF EXISTS provisioned_by`,
		}

		for _, rollback := range teamMembersRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				fmt.Printf("team_members rollback error: %v\n", err)
			}
		}

		// Drop organization team members index
		if _, err := db.NewDropIndex().
			Model((*schema.OrganizationTeamMember)(nil)).
			Index("idx_organization_team_members_provisioned_by").
			IfExists().
			Exec(ctx); err != nil {
			fmt.Printf("organization_team_members rollback error: %v\n", err)
		}

		// Drop organization team members column
		orgTeamMembersRollbacks := []string{
			`ALTER TABLE organization_team_members DROP COLUMN IF EXISTS provisioned_by`,
		}

		for _, rollback := range orgTeamMembersRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				fmt.Printf("organization_team_members rollback error: %v\n", err)
			}
		}

		// ============================================================
		// Rollback Migration 2: Team Provisioning Tracking
		// ============================================================
		// Drop teams indexes
		if _, err := db.NewDropIndex().
			Model((*schema.Team)(nil)).
			Index("idx_teams_provisioned_by").
			IfExists().
			Exec(ctx); err != nil {
			fmt.Printf("teams rollback error: %v\n", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.Team)(nil)).
			Index("idx_teams_external_id").
			IfExists().
			Exec(ctx); err != nil {
			fmt.Printf("teams rollback error: %v\n", err)
		}

		// Drop teams columns
		teamsRollbacks := []string{
			`ALTER TABLE teams DROP COLUMN IF EXISTS external_id`,
			`ALTER TABLE teams DROP COLUMN IF EXISTS provisioned_by`,
		}

		for _, rollback := range teamsRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				fmt.Printf("teams rollback error: %v\n", err)
			}
		}

		// Drop organization teams indexes
		if _, err := db.NewDropIndex().
			Model((*schema.OrganizationTeam)(nil)).
			Index("idx_organization_teams_provisioned_by").
			IfExists().
			Exec(ctx); err != nil {
			fmt.Printf("organization_teams rollback error: %v\n", err)
		}

		if _, err := db.NewDropIndex().
			Model((*schema.OrganizationTeam)(nil)).
			Index("idx_organization_teams_external_id").
			IfExists().
			Exec(ctx); err != nil {
			fmt.Printf("organization_teams rollback error: %v\n", err)
		}

		// Drop organization teams columns
		orgTeamsRollbacks := []string{
			`ALTER TABLE organization_teams DROP COLUMN IF EXISTS external_id`,
			`ALTER TABLE organization_teams DROP COLUMN IF EXISTS provisioned_by`,
		}

		for _, rollback := range orgTeamsRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				fmt.Printf("organization_teams rollback error: %v\n", err)
			}
		}

		// ============================================================
		// Rollback Migration 1: Passkey WebAuthn Fields
		// ============================================================
		// Rollback - remove the added columns
		passkeyRollbacks := []string{
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS last_used_at`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS is_resident_key`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS name`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS authenticator_type`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS sign_count`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS aaguid`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS public_key`,
		}

		for _, rollback := range passkeyRollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				// Log error but continue with other rollbacks
				fmt.Printf("passkey rollback error: %v\n", err)
			}
		}

		return nil
	})
}

