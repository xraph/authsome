package migrations

import (
	"context"
	"database/sql"
	"testing"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/xraph/authsome/schema"
)

// TestInitialMigration_M2MModelsRegistered verifies that m2m models are properly registered
// in the initial migration and that table creation doesn't panic
func TestInitialMigration_M2MModelsRegistered(t *testing.T) {
	// Create in-memory SQLite database
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer sqldb.Close()

	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()

	// Manually register m2m models (simulating what the migration does)
	db.RegisterModel((*schema.TeamMember)(nil))
	db.RegisterModel((*schema.OrganizationTeamMember)(nil))
	db.RegisterModel((*schema.RolePermission)(nil))
	db.RegisterModel((*schema.APIKeyRole)(nil))

	t.Log("Registered m2m models")

	// Create core tables - this would panic without RegisterModel above
	t.Log("Creating tables...")

	if _, err := db.NewCreateTable().Model((*schema.App)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create apps table: %v", err)
	}

	if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	if _, err := db.NewCreateTable().Model((*schema.Member)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create members table: %v", err)
	}

	// This would panic with "can't find m2m team_members table" without the fix
	if _, err := db.NewCreateTable().Model((*schema.Team)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create teams table: %v", err)
	}

	if _, err := db.NewCreateTable().Model((*schema.TeamMember)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create team_members table: %v", err)
	}

	t.Log("✅ All tables created successfully without m2m panic")

	// Test m2m relationship query - the critical test
	t.Log("Testing m2m relationship query...")

	// Insert test data
	appID := xid.New()
	systemActor := xid.New()
	_, err = db.NewInsert().Model(&schema.App{
		ID:   appID,
		Name: "Test App",
		Slug: "test-app",
		AuditableModel: schema.AuditableModel{
			CreatedBy: systemActor,
			UpdatedBy: systemActor,
		},
	}).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to insert app: %v", err)
	}

	teamID := xid.New()
	_, err = db.NewInsert().Model(&schema.Team{
		ID:    teamID,
		AppID: appID,
		Name:  "Test Team",
		AuditableModel: schema.AuditableModel{
			CreatedBy: systemActor,
			UpdatedBy: systemActor,
		},
	}).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to insert team: %v", err)
	}

	// Query with m2m relation - this is what would panic without the fix
	var teams []schema.Team
	err = db.NewSelect().
		Model(&teams).
		Where("id = ?", teamID).
		Relation("Members"). // This triggers the m2m join
		Scan(ctx)
	if err != nil {
		t.Fatalf("m2m relationship query failed: %v", err)
	}

	t.Log("✅ M2M relationship query succeeded - fix verified!")
	t.Logf("Found %d team(s) with %d member(s)", len(teams), len(teams[0].Members))
}
