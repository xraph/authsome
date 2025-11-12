package migrations

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// =================================================================
		// PHASE 1: Rename organizations table to apps
		// =================================================================
		fmt.Println("[Migration] Phase 1: Renaming organizations table to apps...")

		// Rename the table
		if _, err := db.ExecContext(ctx, "ALTER TABLE organizations RENAME TO apps"); err != nil {
			return fmt.Errorf("failed to rename organizations table: %w", err)
		}

		// Rename the index
		if _, err := db.ExecContext(ctx, "ALTER INDEX organizations_pkey RENAME TO apps_pkey"); err != nil {
			// Try alternate syntax for different databases
			if _, err2 := db.ExecContext(ctx, "ALTER INDEX IF EXISTS organizations_pkey RENAME TO apps_pkey"); err2 != nil {
				fmt.Printf("[Warning] Could not rename primary key index: %v\n", err)
			}
		}

		// Rename unique constraint on slug
		if _, err := db.ExecContext(ctx, "ALTER INDEX organizations_slug_idx RENAME TO apps_slug_idx"); err != nil {
			fmt.Printf("[Warning] Could not rename slug index: %v\n", err)
		}

		fmt.Println("[Migration] Phase 1: Complete - organizations → apps")

		// =================================================================
		// PHASE 2: Create environments table
		// =================================================================
		fmt.Println("[Migration] Phase 2: Creating environments table...")

		if _, err := db.NewCreateTable().
			Model((*schema.Environment)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create environments table: %w", err)
		}

		// Add indexes for environments
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_environments_app_id ON environments(app_id);
		`); err != nil {
			return fmt.Errorf("failed to create environments app_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_environments_app_slug ON environments(app_id, slug);
		`); err != nil {
			return fmt.Errorf("failed to create environments unique app+slug index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_environments_is_default ON environments(app_id, is_default) WHERE is_default = true;
		`); err != nil {
			return fmt.Errorf("failed to create environments default index: %w", err)
		}

		fmt.Println("[Migration] Phase 2: Complete - environments table created")

		// =================================================================
		// PHASE 3: Create environment_promotions table
		// =================================================================
		fmt.Println("[Migration] Phase 3: Creating environment_promotions table...")

		if _, err := db.NewCreateTable().
			Model((*schema.EnvironmentPromotion)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create environment_promotions table: %w", err)
		}

		// Add indexes for environment promotions
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_env_promotions_app_id ON environment_promotions(app_id);
		`); err != nil {
			return fmt.Errorf("failed to create environment_promotions app_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_env_promotions_status ON environment_promotions(status);
		`); err != nil {
			return fmt.Errorf("failed to create environment_promotions status index: %w", err)
		}

		fmt.Println("[Migration] Phase 3: Complete - environment_promotions table created")

		// =================================================================
		// PHASE 4: Create user_organizations table (user-created organizations)
		// =================================================================
		fmt.Println("[Migration] Phase 4: Creating user_organizations table...")

		if _, err := db.NewCreateTable().
			Model((*schema.Organization)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_organizations table: %w", err)
		}

		// Add indexes for user organizations
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_orgs_app_id ON user_organizations(app_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organizations app_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_orgs_env_id ON user_organizations(environment_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organizations environment_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_user_orgs_app_env_slug ON user_organizations(app_id, environment_id, slug);
		`); err != nil {
			return fmt.Errorf("failed to create user_organizations unique app+env+slug index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_orgs_created_by ON user_organizations(created_by);
		`); err != nil {
			return fmt.Errorf("failed to create user_organizations created_by index: %w", err)
		}

		fmt.Println("[Migration] Phase 4: Complete - user_organizations table created")

		// =================================================================
		// PHASE 5: Create user_organization_members table
		// =================================================================
		fmt.Println("[Migration] Phase 5: Creating user_organization_members table...")

		if _, err := db.NewCreateTable().
			Model((*schema.OrganizationMember)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_organization_members table: %w", err)
		}

		// Add indexes for organization members
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_members_org_id ON user_organization_members(organization_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_members org_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_members_user_id ON user_organization_members(user_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_members user_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org_members_unique ON user_organization_members(organization_id, user_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_members unique index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_members_role ON user_organization_members(role);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_members role index: %w", err)
		}

		fmt.Println("[Migration] Phase 5: Complete - user_organization_members table created")

		// =================================================================
		// PHASE 6: Create user_organization_teams table
		// =================================================================
		fmt.Println("[Migration] Phase 6: Creating user_organization_teams table...")

		if _, err := db.NewCreateTable().
			Model((*schema.OrganizationTeam)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_organization_teams table: %w", err)
		}

		// Add indexes for organization teams
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_teams_org_id ON user_organization_teams(organization_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_teams org_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_teams_name ON user_organization_teams(organization_id, name);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_teams name index: %w", err)
		}

		fmt.Println("[Migration] Phase 6: Complete - user_organization_teams table created")

		// =================================================================
		// PHASE 7: Create user_organization_team_members table
		// =================================================================
		fmt.Println("[Migration] Phase 7: Creating user_organization_team_members table...")

		if _, err := db.NewCreateTable().
			Model((*schema.OrganizationTeamMember)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_organization_team_members table: %w", err)
		}

		// Add indexes for team members
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_team_members_team_id ON user_organization_team_members(team_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_team_members team_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_team_members_member_id ON user_organization_team_members(member_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_team_members member_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org_team_members_unique ON user_organization_team_members(team_id, member_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_team_members unique index: %w", err)
		}

		fmt.Println("[Migration] Phase 7: Complete - user_organization_team_members table created")

		// =================================================================
		// PHASE 8: Create user_organization_invitations table
		// =================================================================
		fmt.Println("[Migration] Phase 8: Creating user_organization_invitations table...")

		if _, err := db.NewCreateTable().
			Model((*schema.OrganizationInvitation)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_organization_invitations table: %w", err)
		}

		// Add indexes for organization invitations
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_invitations_org_id ON user_organization_invitations(organization_id);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_invitations org_id index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_invitations_email ON user_organization_invitations(email);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_invitations email index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org_invitations_token ON user_organization_invitations(token);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_invitations token index: %w", err)
		}

		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_user_org_invitations_status ON user_organization_invitations(status);
		`); err != nil {
			return fmt.Errorf("failed to create user_organization_invitations status index: %w", err)
		}

		fmt.Println("[Migration] Phase 8: Complete - user_organization_invitations table created")

		// =================================================================
		// PHASE 9: Create default environments for existing apps
		// =================================================================
		fmt.Println("[Migration] Phase 9: Creating default environments for existing apps...")

		// Get all existing apps
		var apps []schema.App
		if err := db.NewSelect().
			Model(&apps).
			Scan(ctx); err != nil {
			return fmt.Errorf("failed to fetch existing apps: %w", err)
		}

		// Create default dev environment for each app
		for _, app := range apps {
			env := &schema.Environment{
				ID:        xid.New(),
				AppID:     app.ID,
				Name:      "Development",
				Slug:      "dev",
				Type:      "development",
				Status:    "active",
				IsDefault: true,
				Config:    make(map[string]interface{}),
			}

			if _, err := db.NewInsert().
				Model(env).
				Exec(ctx); err != nil {
				fmt.Printf("[Warning] Failed to create default environment for app %s: %v\n", app.ID, err)
			} else {
				fmt.Printf("[Success] Created default environment for app: %s (%s)\n", app.Name, app.ID)
			}
		}

		fmt.Println("[Migration] Phase 9: Complete - default environments created")

		// =================================================================
		// SUCCESS
		// =================================================================
		fmt.Println("[Migration] ✅ Architecture refactor migration completed successfully!")
		fmt.Printf("[Migration] Summary:\n")
		fmt.Printf("  - Renamed 'organizations' table to 'apps'\n")
		fmt.Printf("  - Created 'environments' table\n")
		fmt.Printf("  - Created 'environment_promotions' table\n")
		fmt.Printf("  - Created 'user_organizations' table\n")
		fmt.Printf("  - Created 'user_organization_members' table\n")
		fmt.Printf("  - Created 'user_organization_teams' table\n")
		fmt.Printf("  - Created 'user_organization_team_members' table\n")
		fmt.Printf("  - Created 'user_organization_invitations' table\n")
		fmt.Printf("  - Created default environments for %d existing app(s)\n", len(apps))

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// =================================================================
		// ROLLBACK MIGRATION
		// =================================================================
		fmt.Println("[Rollback] Starting architecture refactor rollback...")

		// Drop tables in reverse order (respecting foreign keys)
		tables := []string{
			"user_organization_invitations",
			"user_organization_team_members",
			"user_organization_teams",
			"user_organization_members",
			"user_organizations",
			"environment_promotions",
			"environments",
		}

		for _, table := range tables {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)); err != nil {
				fmt.Printf("[Rollback Warning] Failed to drop table %s: %v\n", table, err)
			} else {
				fmt.Printf("[Rollback] Dropped table: %s\n", table)
			}
		}

		// Rename apps back to organizations
		if _, err := db.ExecContext(ctx, "ALTER TABLE apps RENAME TO organizations"); err != nil {
			return fmt.Errorf("rollback failed: could not rename apps table back: %w", err)
		}

		// Rename indexes back
		if _, err := db.ExecContext(ctx, "ALTER INDEX apps_pkey RENAME TO organizations_pkey"); err != nil {
			fmt.Printf("[Rollback Warning] Could not rename primary key index: %v\n", err)
		}

		if _, err := db.ExecContext(ctx, "ALTER INDEX apps_slug_idx RENAME TO organizations_slug_idx"); err != nil {
			fmt.Printf("[Rollback Warning] Could not rename slug index: %v\n", err)
		}

		fmt.Println("[Rollback] ✅ Architecture refactor rollback completed successfully!")

		return nil
	})
}
