package main

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
	"golang.org/x/crypto/bcrypt"
)

// seedCmd represents the seed command
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed database with test data",
	Long:  `Seed database with test data for development and testing purposes.`,
}

// seedBasicCmd seeds basic test data
var seedBasicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Seed basic test data",
	Long:  `Seed basic test data including organizations, users, and roles.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := connectSeedDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return seedBasicData(db)
	},
}

// seedUsersCmd seeds test users
var seedUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Seed test users",
	Long:  `Seed a specified number of test users.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		orgID, _ := cmd.Flags().GetString("org")

		db, err := connectSeedDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return seedUsers(db, count, orgID)
	},
}

// seedOrgsCmd seeds test organizations
var seedOrgsCmd = &cobra.Command{
	Use:   "orgs",
	Short: "Seed test organizations",
	Long:  `Seed a specified number of test organizations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")

		db, err := connectSeedDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return seedOrganizations(db, count)
	},
}

// seedClearCmd clears all seeded data
var seedClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all seeded data",
	Long:  `Clear all seeded data from the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("This will delete all seeded data. Use --confirm flag to proceed.")
			return nil
		}

		db, err := connectSeedDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return clearSeedData(db)
	},
}

func init() {
	// Add subcommands
	seedCmd.AddCommand(seedBasicCmd)
	seedCmd.AddCommand(seedUsersCmd)
	seedCmd.AddCommand(seedOrgsCmd)
	seedCmd.AddCommand(seedClearCmd)

	// Flags for users command
	seedUsersCmd.Flags().Int("count", 10, "Number of users to create")
	seedUsersCmd.Flags().String("org", "", "Organization ID to add users to")

	// Flags for orgs command
	seedOrgsCmd.Flags().Int("count", 5, "Number of organizations to create")

	// Flags for clear command
	seedClearCmd.Flags().Bool("confirm", false, "Confirm deletion of all seeded data")
}

// connectSeedDB connects to the database for seeding (now supports PostgreSQL, MySQL, SQLite)
func connectSeedDB() (*bun.DB, error) {
	return connectDatabaseMulti()
}

// seedBasicData seeds basic test data
func seedBasicData(db *bun.DB) error {
	ctx := context.Background()

	fmt.Println("Seeding basic test data...")

	// Create platform organization
	platformOrgID := xid.New()
	systemID := xid.New() // System user for CLI operations

	platformOrg := &schema.Organization{
		ID:   platformOrgID,
		Name: "Platform",
		Slug: "platform",
	}
	platformOrg.AuditableModel.ID = platformOrgID
	platformOrg.AuditableModel.CreatedBy = systemID
	platformOrg.AuditableModel.UpdatedBy = systemID

	_, err := db.NewInsert().Model(platformOrg).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create platform organization: %w", err)
	}

	// Create default organization
	defaultOrgID := xid.New()
	defaultOrg := &schema.Organization{
		ID:   defaultOrgID,
		Name: "Default Organization",
		Slug: "default",
	}
	defaultOrg.AuditableModel.ID = defaultOrgID
	defaultOrg.AuditableModel.CreatedBy = systemID
	defaultOrg.AuditableModel.UpdatedBy = systemID

	_, err = db.NewInsert().Model(defaultOrg).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create default organization: %w", err)
	}

	// Create admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	adminUserID := xid.New()
	adminUser := &schema.User{
		ID:            adminUserID,
		Email:         "admin@example.com",
		Name:          "Admin User",
		Username:      "admin",
		PasswordHash:  string(hashedPassword),
		EmailVerified: true,
	}
	adminUser.AuditableModel.ID = adminUserID
	adminUser.AuditableModel.CreatedBy = systemID
	adminUser.AuditableModel.UpdatedBy = systemID

	_, err = db.NewInsert().Model(adminUser).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Create admin member
	adminMemberID := xid.New()
	adminMember := &schema.Member{
		ID:             adminMemberID,
		OrganizationID: defaultOrgID,
		UserID:         adminUserID,
		Role:           "admin",
	}
	adminMember.AuditableModel.ID = adminMemberID
	adminMember.AuditableModel.CreatedBy = systemID
	adminMember.AuditableModel.UpdatedBy = systemID

	_, err = db.NewInsert().Model(adminMember).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create admin member: %w", err)
	}

	// Create roles
	roles := []*schema.Role{
		{
			ID:             xid.New(),
			OrganizationID: &defaultOrgID,
			Name:           "admin",
			Description:    "Administrator role with full access",
		},
		{
			ID:             xid.New(),
			OrganizationID: &defaultOrgID,
			Name:           "user",
			Description:    "Standard user role",
		},
		{
			ID:             xid.New(),
			OrganizationID: &defaultOrgID,
			Name:           "viewer",
			Description:    "Read-only access role",
		},
	}

	for _, role := range roles {
		// Set audit fields
		role.AuditableModel.ID = role.ID
		role.AuditableModel.CreatedBy = systemID
		role.AuditableModel.UpdatedBy = systemID

		_, err = db.NewInsert().Model(role).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create role %s: %w", role.Name, err)
		}
	}

	fmt.Println("✓ Basic test data seeded successfully")
	fmt.Printf("✓ Created organizations: %s, %s\n", platformOrgID, defaultOrgID)
	fmt.Printf("✓ Created admin user: %s (admin@example.com / admin123)\n", adminUserID)
	fmt.Println("✓ Created roles: admin, user, viewer")

	return nil
}

// seedUsers seeds test users
func seedUsers(db *bun.DB, count int, orgID string) error {
	ctx := context.Background()

	fmt.Printf("Seeding %d test users...\n", count)

	// Parse organization ID
	var targetOrgID xid.ID
	if orgID != "" {
		parsedID, err := xid.FromString(orgID)
		if err != nil {
			return fmt.Errorf("invalid organization ID: %w", err)
		}
		targetOrgID = parsedID
	} else {
		// Use default organization
		var org schema.Organization
		err := db.NewSelect().Model(&org).Where("slug = ?", "default").Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to find default organization: %w", err)
		}
		targetOrgID = org.ID
	}

	// Create users
	for i := 1; i <= count; i++ {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		userID := xid.New()
		user := &schema.User{
			ID:            userID,
			Email:         fmt.Sprintf("user%d@example.com", i),
			Name:          fmt.Sprintf("Test User %d", i),
			PasswordHash:  string(hashedPassword),
			EmailVerified: i%2 == 0, // Alternate verified status
		}

		_, err = db.NewInsert().Model(user).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create user %d: %w", i, err)
		}

		// Create member
		memberID := xid.New()
		member := &schema.Member{
			ID:             memberID,
			OrganizationID: targetOrgID,
			UserID:         userID,
			Role:           "user",
		}

		_, err = db.NewInsert().Model(member).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create member for user %d: %w", i, err)
		}

		fmt.Printf("✓ Created user %d: %s\n", i, user.Email)
	}

	fmt.Printf("✓ Successfully seeded %d users\n", count)
	return nil
}

// seedOrganizations seeds test organizations
func seedOrganizations(db *bun.DB, count int) error {
	ctx := context.Background()

	fmt.Printf("Seeding %d test organizations...\n", count)

	for i := 1; i <= count; i++ {
		orgID := xid.New()
		org := &schema.Organization{
			ID:   orgID,
			Name: fmt.Sprintf("Test Organization %d", i),
			Slug: fmt.Sprintf("test-org-%d", i),
		}

		_, err := db.NewInsert().Model(org).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create organization %d: %w", i, err)
		}

		fmt.Printf("✓ Created organization %d: %s (%s)\n", i, org.Name, orgID)
	}

	fmt.Printf("✓ Successfully seeded %d organizations\n", count)
	return nil
}

// clearSeedData clears all seeded data
func clearSeedData(db *bun.DB) error {
	ctx := context.Background()

	fmt.Println("Clearing all seeded data...")

	// Delete in reverse order of dependencies
	tables := []string{"members", "roles", "users", "organizations"}

	for _, table := range tables {
		result, err := db.NewDelete().Table(table).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✓ Cleared %d rows from %s\n", rowsAffected, table)
	}

	fmt.Println("✓ All seeded data cleared successfully")
	return nil
}
