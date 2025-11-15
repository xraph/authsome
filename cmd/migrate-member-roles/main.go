package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

func main() {
	// Parse command-line flags
	dsn := flag.String("dsn", "", "Database connection string (required)")
	driver := flag.String("driver", "postgres", "Database driver (postgres or sqlite)")
	dryRun := flag.Bool("dry-run", false, "Dry run mode - show what would be done without making changes")
	flag.Parse()

	if *dsn == "" {
		fmt.Println("Error: --dsn flag is required")
		fmt.Println("\nUsage:")
		fmt.Println("  go run cmd/migrate-member-roles/main.go --dsn <connection-string> [--driver postgres|sqlite] [--dry-run]")
		fmt.Println("\nExamples:")
		fmt.Println("  # PostgreSQL")
		fmt.Println("  go run cmd/migrate-member-roles/main.go --dsn 'postgres://user:pass@localhost:5432/dbname?sslmode=disable'")
		fmt.Println("\n  # SQLite")
		fmt.Println("  go run cmd/migrate-member-roles/main.go --dsn 'file:authsome.db' --driver sqlite")
		fmt.Println("\n  # Dry run")
		fmt.Println("  go run cmd/migrate-member-roles/main.go --dsn 'postgres://...' --dry-run")
		os.Exit(1)
	}

	// Initialize database connection
	db, err := initDB(*dsn, *driver)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run the backfill migration
	if *dryRun {
		fmt.Println("=== DRY RUN MODE - No changes will be made ===\n")
		if err := backfillMemberRolesDryRun(ctx, db); err != nil {
			log.Fatalf("Dry run failed: %v", err)
		}
	} else {
		if err := backfillMemberRoles(ctx, db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	fmt.Println("\nMigration completed successfully!")
}

// initDB initializes the database connection
func initDB(dsn, driver string) (*bun.DB, error) {
	var sqldb *sql.DB
	var db *bun.DB
	var err error

	switch driver {
	case "postgres":
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db = bun.NewDB(sqldb, pgdialect.New())
	case "sqlite":
		sqldb, err = sql.Open(sqliteshim.ShimName, dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite: %w", err)
		}
		db = bun.NewDB(sqldb, sqlitedialect.New())
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// backfillMemberRolesDryRun shows what would be done without making changes
func backfillMemberRolesDryRun(ctx context.Context, db *bun.DB) error {
	fmt.Println("Analyzing existing members and roles...")

	// Get all members
	var members []schema.Member
	err := db.NewSelect().Model(&members).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch members: %w", err)
	}

	fmt.Printf("\nFound %d members to process\n\n", len(members))

	// Track statistics
	stats := struct {
		total       int
		alreadyHave int
		needSync    int
		errors      []string
	}{}

	for _, member := range members {
		stats.total++
		
		// Map member role to RBAC role name
		rbacRoleName := mapMemberRoleToRBAC(string(member.Role))

		// Check if role exists for this app
		var role schema.Role
		err := db.NewSelect().
			Model(&role).
			Where("name = ?", rbacRoleName).
			Where("app_id = ?", member.AppID).
			Scan(ctx)
		if err != nil {
			stats.errors = append(stats.errors, fmt.Sprintf("Member %s: role %s not found in app %s", member.ID, rbacRoleName, member.AppID))
			continue
		}

		// Check if UserRole already exists
		var existingUserRole schema.UserRole
		err = db.NewSelect().
			Model(&existingUserRole).
			Where("user_id = ?", member.UserID).
			Where("role_id = ?", role.ID).
			Where("app_id = ?", member.AppID).
			Scan(ctx)

		if err == nil {
			// UserRole already exists
			stats.alreadyHave++
			fmt.Printf("✓ Member %s (user %s, app %s): Already has UserRole %s\n",
				member.ID, member.UserID, member.AppID, rbacRoleName)
		} else {
			// Need to create UserRole
			stats.needSync++
			fmt.Printf("→ Member %s (user %s, app %s): Would create UserRole %s (role_id %s)\n",
				member.ID, member.UserID, member.AppID, rbacRoleName, role.ID)
		}
	}

	// Print summary
	fmt.Println("\n=== Dry Run Summary ===")
	fmt.Printf("Total members: %d\n", stats.total)
	fmt.Printf("Already have UserRole: %d\n", stats.alreadyHave)
	fmt.Printf("Need UserRole sync: %d\n", stats.needSync)
	if len(stats.errors) > 0 {
		fmt.Printf("Errors: %d\n", len(stats.errors))
		fmt.Println("\nError details:")
		for _, errMsg := range stats.errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	}

	return nil
}

// backfillMemberRoles creates UserRole entries for all existing members
func backfillMemberRoles(ctx context.Context, db *bun.DB) error {
	fmt.Println("Starting member role backfill migration...")

	// Get all members
	var members []schema.Member
	err := db.NewSelect().Model(&members).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch members: %w", err)
	}

	fmt.Printf("Found %d members to process\n\n", len(members))

	// Track statistics
	stats := struct {
		total       int
		created     int
		skipped     int
		errors      int
	}{}

	for _, member := range members {
		stats.total++
		
		// Map member role to RBAC role name
		rbacRoleName := mapMemberRoleToRBAC(string(member.Role))

		// Get role ID
		var role schema.Role
		err := db.NewSelect().
			Model(&role).
			Where("name = ?", rbacRoleName).
			Where("app_id = ?", member.AppID).
			Scan(ctx)
		if err != nil {
			fmt.Printf("✗ Member %s: Failed to find role %s in app %s: %v\n",
				member.ID, rbacRoleName, member.AppID, err)
			stats.errors++
			continue
		}

		// Check if UserRole already exists
		var existingUserRole schema.UserRole
		err = db.NewSelect().
			Model(&existingUserRole).
			Where("user_id = ?", member.UserID).
			Where("role_id = ?", role.ID).
			Where("app_id = ?", member.AppID).
			Scan(ctx)

		if err == nil {
			// UserRole already exists
			fmt.Printf("→ Member %s: UserRole already exists (skipped)\n", member.ID)
			stats.skipped++
			continue
		}

		// Create UserRole entry
		userRole := &schema.UserRole{
			ID:     xid.New(),
			UserID: member.UserID,
			RoleID: role.ID,
			AppID:  member.AppID,
		}
		userRole.CreatedAt = member.CreatedAt
		userRole.UpdatedAt = member.UpdatedAt
		userRole.CreatedBy = member.UserID // User assigned themselves by accepting invitation or being added
		userRole.UpdatedBy = member.UserID

		_, err = db.NewInsert().Model(userRole).Exec(ctx)
		if err != nil {
			fmt.Printf("✗ Member %s: Failed to create UserRole: %v\n", member.ID, err)
			stats.errors++
			continue
		}

		fmt.Printf("✓ Member %s: Created UserRole %s (role_id %s)\n",
			member.ID, rbacRoleName, role.ID)
		stats.created++
	}

	// Print summary
	fmt.Println("\n=== Migration Summary ===")
	fmt.Printf("Total members: %d\n", stats.total)
	fmt.Printf("UserRoles created: %d\n", stats.created)
	fmt.Printf("Already existed (skipped): %d\n", stats.skipped)
	fmt.Printf("Errors: %d\n", stats.errors)

	if stats.errors > 0 {
		return fmt.Errorf("migration completed with %d errors", stats.errors)
	}

	return nil
}

// mapMemberRoleToRBAC maps member role strings to RBAC role constants
func mapMemberRoleToRBAC(memberRole string) string {
	switch memberRole {
	case "owner":
		return rbac.RoleOwner // "owner"
	case "admin":
		return rbac.RoleAdmin // "admin"
	case "member":
		return rbac.RoleMember // "member"
	default:
		return rbac.RoleMember // Default to member
	}
}

