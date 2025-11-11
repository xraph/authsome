package rbac

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/xraph/authsome/schema"
)

func setupTestDB(t *testing.T) *bun.DB {
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create necessary tables
	ctx := context.Background()

	// Organizations table
	_, err = db.NewCreateTable().
		Model((*schema.Organization)(nil)).
		IfNotExists().
		Exec(ctx)
	require.NoError(t, err)

	// Roles table
	_, err = db.NewCreateTable().
		Model((*schema.Role)(nil)).
		IfNotExists().
		Exec(ctx)
	require.NoError(t, err)

	return db
}

func TestRoleRegistry_RegisterRole(t *testing.T) {
	registry := NewRoleRegistry()

	role := &RoleDefinition{
		Name:        "admin",
		Description: "Administrator",
		IsPlatform:  false,
		Priority:    60,
		Permissions: []string{"view on users", "edit on users"},
	}

	err := registry.RegisterRole(role)
	assert.NoError(t, err)

	// Verify role was registered
	retrieved, exists := registry.GetRole("admin")
	assert.True(t, exists)
	assert.Equal(t, "admin", retrieved.Name)
	assert.Equal(t, "Administrator", retrieved.Description)
	assert.Len(t, retrieved.Permissions, 2)
}

func TestRoleRegistry_OverrideSemantics(t *testing.T) {
	registry := NewRoleRegistry()

	// Register initial role
	role1 := &RoleDefinition{
		Name:        "admin",
		Description: "Administrator v1",
		IsPlatform:  false,
		Priority:    60,
		Permissions: []string{"view on users"},
	}
	err := registry.RegisterRole(role1)
	require.NoError(t, err)

	// Override with additional permissions
	role2 := &RoleDefinition{
		Name:        "admin",
		Description: "Administrator v2",
		IsPlatform:  false,
		Priority:    70, // Higher priority
		Permissions: []string{"edit on users", "delete on users"},
	}
	err = registry.RegisterRole(role2)
	require.NoError(t, err)

	// Verify permissions were merged
	retrieved, exists := registry.GetRole("admin")
	require.True(t, exists)
	assert.Equal(t, "Administrator v2", retrieved.Description)
	assert.Equal(t, 70, retrieved.Priority)
	assert.Len(t, retrieved.Permissions, 3) // All three permissions should be present
}

func TestRoleRegistry_RoleInheritance(t *testing.T) {
	registry := NewRoleRegistry()

	// Register parent role
	parent := &RoleDefinition{
		Name:        "member",
		Description: "Member",
		IsPlatform:  false,
		Priority:    40,
		Permissions: []string{"view on profile", "edit on profile"},
	}
	err := registry.RegisterRole(parent)
	require.NoError(t, err)

	// Register child role that inherits from parent
	child := &RoleDefinition{
		Name:         "admin",
		Description:  "Administrator",
		IsPlatform:   false,
		InheritsFrom: "member",
		Priority:     60,
		Permissions:  []string{"view on users", "edit on users"},
	}
	err = registry.RegisterRole(child)
	require.NoError(t, err)

	// Resolve inheritance
	resolved, err := registry.resolveInheritance()
	require.NoError(t, err)

	// Find the resolved admin role
	var adminRole *RoleDefinition
	for _, r := range resolved {
		if r.Name == "admin" {
			adminRole = r
			break
		}
	}

	require.NotNil(t, adminRole)
	assert.Len(t, adminRole.Permissions, 4) // 2 from parent + 2 from child
	assert.Contains(t, adminRole.Permissions, "view on profile")
	assert.Contains(t, adminRole.Permissions, "edit on profile")
	assert.Contains(t, adminRole.Permissions, "view on users")
	assert.Contains(t, adminRole.Permissions, "edit on users")
}

func TestRoleRegistry_CircularInheritanceDetection(t *testing.T) {
	registry := NewRoleRegistry()

	// Create circular dependency: admin -> member -> admin
	role1 := &RoleDefinition{
		Name:         "admin",
		Description:  "Administrator",
		InheritsFrom: "member",
		Permissions:  []string{"admin permission"},
	}
	err := registry.RegisterRole(role1)
	require.NoError(t, err)

	role2 := &RoleDefinition{
		Name:         "member",
		Description:  "Member",
		InheritsFrom: "admin", // Circular!
		Permissions:  []string{"member permission"},
	}
	err = registry.RegisterRole(role2)
	require.NoError(t, err)

	// Should detect circular dependency
	_, err = registry.resolveInheritance()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestRoleRegistry_Bootstrap(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create platform organization
	platformOrg := &schema.Organization{
		ID:         xid.New(),
		Name:       "Platform Organization",
		Slug:       "platform",
		IsPlatform: true,
		Metadata:   map[string]interface{}{},
	}
	platformOrg.CreatedAt = time.Now()
	platformOrg.UpdatedAt = time.Now()
	platformOrg.CreatedBy = platformOrg.ID
	platformOrg.UpdatedBy = platformOrg.ID
	platformOrg.Version = 1

	_, err := db.NewInsert().Model(platformOrg).Exec(ctx)
	require.NoError(t, err)

	// Create role registry with roles
	registry := NewRoleRegistry()

	err = registry.RegisterRole(&RoleDefinition{
		Name:        "admin",
		Description: "Administrator",
		IsPlatform:  false,
		Priority:    60,
		Permissions: []string{"view on users", "edit on users"},
	})
	require.NoError(t, err)

	err = registry.RegisterRole(&RoleDefinition{
		Name:        "member",
		Description: "Member",
		IsPlatform:  false,
		Priority:    40,
		Permissions: []string{"view on profile"},
	})
	require.NoError(t, err)

	// Bootstrap roles
	rbacService := NewService()
	err = registry.Bootstrap(ctx, db, rbacService, platformOrg.ID)
	require.NoError(t, err)

	// Verify roles were created in database
	var roles []schema.Role
	err = db.NewSelect().
		Model(&roles).
		Where("organization_id = ?", platformOrg.ID).
		Scan(ctx)
	require.NoError(t, err)

	assert.Len(t, roles, 2)

	// Verify role names
	roleNames := make(map[string]bool)
	for _, role := range roles {
		roleNames[role.Name] = true
	}
	assert.True(t, roleNames["admin"])
	assert.True(t, roleNames["member"])
}

func TestRoleRegistry_BootstrapIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create platform organization
	platformOrg := &schema.Organization{
		ID:         xid.New(),
		Name:       "Platform Organization",
		Slug:       "platform",
		IsPlatform: true,
		Metadata:   map[string]interface{}{},
	}
	platformOrg.CreatedAt = time.Now()
	platformOrg.UpdatedAt = time.Now()
	platformOrg.CreatedBy = platformOrg.ID
	platformOrg.UpdatedBy = platformOrg.ID
	platformOrg.Version = 1

	_, err := db.NewInsert().Model(platformOrg).Exec(ctx)
	require.NoError(t, err)

	// Create role registry
	registry := NewRoleRegistry()
	err = registry.RegisterRole(&RoleDefinition{
		Name:        "admin",
		Description: "Administrator v1",
		IsPlatform:  false,
		Priority:    60,
		Permissions: []string{"view on users"},
	})
	require.NoError(t, err)

	// Bootstrap first time
	rbacService := NewService()
	err = registry.Bootstrap(ctx, db, rbacService, platformOrg.ID)
	require.NoError(t, err)

	// Count roles
	count1, err := db.NewSelect().
		Model((*schema.Role)(nil)).
		Where("organization_id = ?", platformOrg.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	// Update role definition
	err = registry.RegisterRole(&RoleDefinition{
		Name:        "admin",
		Description: "Administrator v2",
		IsPlatform:  false,
		Priority:    60,
		Permissions: []string{"view on users", "edit on users"},
	})
	require.NoError(t, err)

	// Bootstrap again (idempotent)
	err = registry.Bootstrap(ctx, db, rbacService, platformOrg.ID)
	require.NoError(t, err)

	// Should still have only 1 role (updated, not duplicated)
	count2, err := db.NewSelect().
		Model((*schema.Role)(nil)).
		Where("organization_id = ?", platformOrg.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count2)

	// Verify role was updated
	var role schema.Role
	err = db.NewSelect().
		Model(&role).
		Where("name = ? AND organization_id = ?", "admin", platformOrg.ID).
		Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Administrator v2", role.Description)
}

func TestRoleRegistry_ValidateRoleAssignment(t *testing.T) {
	registry := NewRoleRegistry()

	// Register platform role
	err := registry.RegisterRole(&RoleDefinition{
		Name:        "superadmin",
		Description: "Superadministrator",
		IsPlatform:  true,
		Priority:    100,
		Permissions: []string{"* on *"},
	})
	require.NoError(t, err)

	// Register org role
	err = registry.RegisterRole(&RoleDefinition{
		Name:        "admin",
		Description: "Administrator",
		IsPlatform:  false,
		Priority:    60,
		Permissions: []string{"view on users"},
	})
	require.NoError(t, err)

	// Test platform role validation
	err = registry.ValidateRoleAssignment("superadmin", true)
	assert.NoError(t, err, "Platform role should be assignable in platform org")

	err = registry.ValidateRoleAssignment("superadmin", false)
	assert.Error(t, err, "Platform role should NOT be assignable in non-platform org")
	assert.Contains(t, err.Error(), "platform role")

	// Test org role validation
	err = registry.ValidateRoleAssignment("admin", true)
	assert.NoError(t, err, "Org role should be assignable in platform org")

	err = registry.ValidateRoleAssignment("admin", false)
	assert.NoError(t, err, "Org role should be assignable in non-platform org")
}

func TestRoleRegistry_GetRoleHierarchy(t *testing.T) {
	registry := NewRoleRegistry()

	// Register roles with different priorities
	roles := []*RoleDefinition{
		{Name: "member", Priority: 40, Permissions: []string{"view on profile"}},
		{Name: "admin", Priority: 60, Permissions: []string{"view on users"}},
		{Name: "owner", Priority: 80, Permissions: []string{"* on organization.*"}},
		{Name: "superadmin", Priority: 100, Permissions: []string{"* on *"}},
	}

	for _, role := range roles {
		err := registry.RegisterRole(role)
		require.NoError(t, err)
	}

	// Get hierarchy (should be sorted by priority descending)
	hierarchy := registry.GetRoleHierarchy()
	require.Len(t, hierarchy, 4)

	assert.Equal(t, "superadmin", hierarchy[0].Name)
	assert.Equal(t, "owner", hierarchy[1].Name)
	assert.Equal(t, "admin", hierarchy[2].Name)
	assert.Equal(t, "member", hierarchy[3].Name)
}

func TestRegisterDefaultPlatformRoles(t *testing.T) {
	registry := NewRoleRegistry()

	err := RegisterDefaultPlatformRoles(registry)
	require.NoError(t, err)

	// Verify all default roles were registered
	expectedRoles := []string{"superadmin", "owner", "admin", "member"}
	for _, roleName := range expectedRoles {
		role, exists := registry.GetRole(roleName)
		assert.True(t, exists, "Role %s should exist", roleName)
		assert.NotNil(t, role)
		assert.NotEmpty(t, role.Description)
		assert.NotEmpty(t, role.Permissions)
	}

	// Verify role hierarchy
	hierarchy := registry.GetRoleHierarchy()
	assert.Equal(t, "superadmin", hierarchy[0].Name)
	assert.Equal(t, "owner", hierarchy[1].Name)
	assert.Equal(t, "admin", hierarchy[2].Name)
	assert.Equal(t, "member", hierarchy[3].Name)

	// Verify platform flags
	superadmin, _ := registry.GetRole("superadmin")
	assert.True(t, superadmin.IsPlatform)

	owner, _ := registry.GetRole("owner")
	assert.False(t, owner.IsPlatform)
}
