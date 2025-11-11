package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/xraph/authsome/repository"
)

func main() {
	// Connect to database
	dsn := "postgresql://postgres:postgres@localhost/kineta?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	ctx := context.Background()

	// Get test data
	var userID xid.ID
	err := db.NewSelect().Table("users").Column("id").Limit(1).Scan(ctx, &userID)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("User ID: %s\n", userID.String())

	var roleID xid.ID
	err = db.NewSelect().Table("roles").Column("id").Where("name = ?", "superadmin").Limit(1).Scan(ctx, &roleID)
	if err != nil {
		log.Fatalf("Failed to get role: %v", err)
	}
	fmt.Printf("Role ID: %s\n", roleID.String())

	var orgID xid.ID
	err = db.NewSelect().Table("organizations").Column("id").Where("is_platform = true").Limit(1).Scan(ctx, &orgID)
	if err != nil {
		log.Fatalf("Failed to get org: %v", err)
	}
	fmt.Printf("Org ID: %s\n", orgID.String())

	// Test the Assign method
	userRoleRepo := repository.NewUserRoleRepository(db)

	fmt.Println("\nAttempting role assignment...")
	err = userRoleRepo.Assign(ctx, userID, roleID, orgID)
	if err != nil {
		log.Fatalf("ERROR: Assign failed: %v", err)
	}

	fmt.Println("✅ Role assigned successfully!")

	// Verify
	var count int
	err = db.NewSelect().Table("user_roles").ColumnExpr("COUNT(*)").Scan(ctx, &count)
	if err != nil {
		log.Fatalf("Failed to count user_roles: %v", err)
	}
	fmt.Printf("\nUser_roles count: %d\n", count)
}
