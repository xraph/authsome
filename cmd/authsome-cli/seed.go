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
	Long:  `Seed basic test data including apps, users, and roles.`,
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
		appID, _ := cmd.Flags().GetString("app")

		db, err := connectSeedDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return seedUsers(db, count, appID)
	},
}

// seedAppsCmd seeds test apps
var seedAppsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Seed test apps",
	Long:  `Seed a specified number of test apps.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")

		db, err := connectSeedDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return seedApps(db, count)
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
	seedCmd.AddCommand(seedAppsCmd)
	seedCmd.AddCommand(seedClearCmd)

	// Flags for users command
	seedUsersCmd.Flags().Int("count", 10, "Number of users to create")
	seedUsersCmd.Flags().String("app", "", "App ID to add users to")

	// Flags for apps command
	seedAppsCmd.Flags().Int("count", 5, "Number of apps to create")

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

	// Create platform app
	platformAppID := xid.New()
	systemID := xid.New() // System user for CLI operations

	platformApp := &schema.App{
		ID:         platformAppID,
		Name:       "Platform",
		Slug:       "platform",
		IsPlatform: true,
	}
	platformApp.AuditableModel.ID = platformAppID
	platformApp.AuditableModel.CreatedBy = systemID
	platformApp.AuditableModel.UpdatedBy = systemID

	_, err := db.NewInsert().Model(platformApp).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create platform app: %w", err)
	}

	// Create default app
	defaultAppID := xid.New()
	defaultApp := &schema.App{
		ID:   defaultAppID,
		Name: "Default App",
		Slug: "default",
	}
	defaultApp.AuditableModel.ID = defaultAppID
	defaultApp.AuditableModel.CreatedBy = systemID
	defaultApp.AuditableModel.UpdatedBy = systemID

	_, err = db.NewInsert().Model(defaultApp).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create default app: %w", err)
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
		ID:     adminMemberID,
		AppID:  defaultAppID,
		UserID: adminUserID,
		Role:   schema.MemberRoleAdmin,
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
			ID:          xid.New(),
			AppID:       &defaultAppID,
			Name:        "admin",
			Description: "Administrator role with full access",
			IsTemplate:  true,
		},
		{
			ID:          xid.New(),
			AppID:       &defaultAppID,
			Name:        "user",
			Description: "Standard user role",
			IsTemplate:  true,
		},
		{
			ID:          xid.New(),
			AppID:       &defaultAppID,
			Name:        "viewer",
			Description: "Read-only access role",
			IsTemplate:  true,
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
	fmt.Printf("✓ Created apps: %s, %s\n", platformAppID, defaultAppID)
	fmt.Printf("✓ Created admin user: %s (admin@example.com / admin123)\n", adminUserID)
	fmt.Println("✓ Created roles: admin, user, viewer")

	return nil
}

// seedUsers seeds test users
func seedUsers(db *bun.DB, count int, appID string) error {
	ctx := context.Background()

	fmt.Printf("Seeding %d test users...\n", count)

	// Use system ID for created_by/updated_by
	systemID := xid.New() // System user for CLI operations

	// Parse app ID
	var targetAppID xid.ID
	if appID != "" {
		parsedID, err := xid.FromString(appID)
		if err != nil {
			return fmt.Errorf("invalid app ID: %w", err)
		}
		targetAppID = parsedID
	} else {
		// Use default app
		var app schema.App
		err := db.NewSelect().Model(&app).Where("slug = ?", "default").Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to find default app: %w", err)
		}
		targetAppID = app.ID
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
			ID:     memberID,
			AppID:  targetAppID,
			UserID: userID,
			Role:   schema.MemberRoleMember,
		}

		// Set audit fields
		member.AuditableModel.ID = memberID
		member.AuditableModel.CreatedBy = systemID
		member.AuditableModel.UpdatedBy = systemID

		_, err = db.NewInsert().Model(member).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create member for user %d: %w", i, err)
		}

		fmt.Printf("✓ Created user %d: %s\n", i, user.Email)
	}

	fmt.Printf("✓ Successfully seeded %d users\n", count)
	return nil
}

// seedApps seeds test apps
func seedApps(db *bun.DB, count int) error {
	ctx := context.Background()

	fmt.Printf("Seeding %d test apps...\n", count)

	// Use system ID for created_by/updated_by
	systemID := xid.New() // System user for CLI operations

	for i := 1; i <= count; i++ {
		appID := xid.New()
		app := &schema.App{
			ID:   appID,
			Name: fmt.Sprintf("Test App %d", i),
			Slug: fmt.Sprintf("test-app-%d", i),
		}

		// Set audit fields
		app.AuditableModel.ID = appID
		app.AuditableModel.CreatedBy = systemID
		app.AuditableModel.UpdatedBy = systemID

		_, err := db.NewInsert().Model(app).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create app %d: %w", i, err)
		}

		fmt.Printf("✓ Created app %d: %s (%s)\n", i, app.Name, appID)
	}

	fmt.Printf("✓ Successfully seeded %d apps\n", count)
	return nil
}

// clearSeedData clears all seeded data
func clearSeedData(db *bun.DB) error {
	ctx := context.Background()

	fmt.Println("Clearing all seeded data...")

	// Delete in reverse order of dependencies
	tables := []string{"members", "roles", "users", "apps"}

	for _, table := range tables {
		result, err := db.NewDelete().Table(table).Where("1 = 1").Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✓ Cleared %d rows from %s\n", rowsAffected, table)
	}

	fmt.Println("✓ All seeded data cleared successfully")
	return nil
}
