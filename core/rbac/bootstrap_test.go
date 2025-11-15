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

	// Apps table
	_, err = db.NewCreateTable().
		Model((*schema.App)(nil)).
		IfNotExists().
		Exec(ctx)
	require.NoError(t, err)

	// Roles table
	_, err = db.NewCreateTable().
		Model((*schema.Role)(nil)).
		IfNotExists().
		Exec(ctx)
	require.NoError(t, err)

	// Permissions table
	_, err = db.NewCreateTable().
		Model((*schema.Permission)(nil)).
		IfNotExists().
		Exec(ctx)
	require.NoError(t, err)

	return db
}

func TestRoleRegistry_RegisterRole(t *testing.T) {
	registry := NewRoleRegistry()

	role := &RoleDefinition{
		Name:        RoleAdmin,
		Description: RoleDescAdmin,
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    RolePriorityAdmin,
		Permissions: []string{"view on users", "edit on users"},
	}

	err := registry.RegisterRole(role)
	assert.NoError(t, err)

	// Verify role was registered
	retrieved, exists := registry.GetRole(RoleAdmin)
	assert.True(t, exists)
	assert.Equal(t, RoleAdmin, retrieved.Name)
	assert.Equal(t, RoleDescAdmin, retrieved.Description)
	assert.Len(t, retrieved.Permissions, 2)
}

func TestRoleRegistry_OverrideSemantics(t *testing.T) {
	registry := NewRoleRegistry()

	// Register initial role
	role1 := &RoleDefinition{
		Name:        RoleAdmin,
		Description: "Administrator v1",
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    RolePriorityAdmin,
		Permissions: []string{"view on users"},
	}
	err := registry.RegisterRole(role1)
	require.NoError(t, err)

	// Override with additional permissions
	role2 := &RoleDefinition{
		Name:        RoleAdmin,
		Description: "Administrator v2",
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    70, // Higher priority
		Permissions: []string{"edit on users", "delete on users"},
	}
	err = registry.RegisterRole(role2)
	require.NoError(t, err)

	// Verify permissions were merged
	retrieved, exists := registry.GetRole(RoleAdmin)
	require.True(t, exists)
	assert.Equal(t, "Administrator v2", retrieved.Description)
	assert.Equal(t, 70, retrieved.Priority)
	assert.Len(t, retrieved.Permissions, 3) // All three permissions should be present
}

func TestRoleRegistry_RoleInheritance(t *testing.T) {
	registry := NewRoleRegistry()

	// Register parent role
	parent := &RoleDefinition{
		Name:        RoleMember,
		Description: RoleDescMember,
		IsPlatform:  RoleIsPlatformMember,
		Priority:    RolePriorityMember,
		Permissions: []string{"view on profile", "edit on profile"},
	}
	err := registry.RegisterRole(parent)
	require.NoError(t, err)

	// Register child role that inherits from parent
	child := &RoleDefinition{
		Name:         RoleAdmin,
		Description:  RoleDescAdmin,
		IsPlatform:   RoleIsPlatformAdmin,
		InheritsFrom: RoleMember,
		Priority:     RolePriorityAdmin,
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
		if r.Name == RoleAdmin {
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
		Name:         RoleAdmin,
		Description:  RoleDescAdmin,
		InheritsFrom: RoleMember,
		Permissions:  []string{"admin permission"},
	}
	err := registry.RegisterRole(role1)
	require.NoError(t, err)

	role2 := &RoleDefinition{
		Name:         RoleMember,
		Description:  RoleDescMember,
		InheritsFrom: RoleAdmin, // Circular!
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

	// Create platform app
	platformOrg := &schema.App{
		ID:         xid.New(),
		Name:       "Platform App",
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
		Name:        RoleAdmin,
		Description: RoleDescAdmin,
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    RolePriorityAdmin,
		Permissions: []string{"view on users", "edit on users"},
	})
	require.NoError(t, err)

	err = registry.RegisterRole(&RoleDefinition{
		Name:        RoleMember,
		Description: RoleDescMember,
		IsPlatform:  RoleIsPlatformMember,
		Priority:    RolePriorityMember,
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
		Where("app_id = ?", platformOrg.ID).
		Scan(ctx)
	require.NoError(t, err)

	assert.Len(t, roles, 2)

	// Verify role names
	roleNames := make(map[string]bool)
	for _, role := range roles {
		roleNames[role.Name] = true
	}
	assert.True(t, roleNames[RoleAdmin])
	assert.True(t, roleNames[RoleMember])

	// Verify permissions were created in database
	var permissions []schema.Permission
	err = db.NewSelect().
		Model(&permissions).
		Where("app_id = ?", platformOrg.ID).
		Scan(ctx)
	require.NoError(t, err)

	assert.Greater(t, len(permissions), 0, "Permissions should have been created")

	// Verify at least one permission exists
	permNames := make(map[string]bool)
	for _, perm := range permissions {
		permNames[perm.Name] = true
	}
	assert.True(t, permNames["view on users"] || permNames["edit on users"] || permNames["view on profile"])
}

func TestRoleRegistry_BootstrapIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create platform app
	platformOrg := &schema.App{
		ID:         xid.New(),
		Name:       "Platform App",
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
		Name:        RoleAdmin,
		Description: "Administrator v1",
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    RolePriorityAdmin,
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
		Where("app_id = ?", platformOrg.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	// Update role definition
	err = registry.RegisterRole(&RoleDefinition{
		Name:        RoleAdmin,
		Description: "Administrator v2",
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    RolePriorityAdmin,
		Permissions: []string{"view on users", "edit on users"},
	})
	require.NoError(t, err)

	// Bootstrap again (idempotent)
	err = registry.Bootstrap(ctx, db, rbacService, platformOrg.ID)
	require.NoError(t, err)

	// Should still have only 1 role (updated, not duplicated)
	count2, err := db.NewSelect().
		Model((*schema.Role)(nil)).
		Where("app_id = ?", platformOrg.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count2)

	// Verify role was updated
	var role schema.Role
	err = db.NewSelect().
		Model(&role).
		Where("name = ? AND app_id = ?", RoleAdmin, platformOrg.ID).
		Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Administrator v2", role.Description)
}

func TestRoleRegistry_ValidateRoleAssignment(t *testing.T) {
	registry := NewRoleRegistry()

	// Register platform role
	err := registry.RegisterRole(&RoleDefinition{
		Name:        RoleSuperAdmin,
		Description: RoleDescSuperAdmin,
		IsPlatform:  RoleIsPlatformSuperAdmin,
		Priority:    RolePrioritySuperAdmin,
		Permissions: []string{"* on *"},
	})
	require.NoError(t, err)

	// Register org role
	err = registry.RegisterRole(&RoleDefinition{
		Name:        RoleAdmin,
		Description: RoleDescAdmin,
		IsPlatform:  RoleIsPlatformAdmin,
		Priority:    RolePriorityAdmin,
		Permissions: []string{"view on users"},
	})
	require.NoError(t, err)

	// Test platform role validation
	err = registry.ValidateRoleAssignment(RoleSuperAdmin, true)
	assert.NoError(t, err, "Platform role should be assignable in platform org")

	err = registry.ValidateRoleAssignment(RoleSuperAdmin, false)
	assert.Error(t, err, "Platform role should NOT be assignable in non-platform org")
	assert.Contains(t, err.Error(), "platform role")

	// Test org role validation
	err = registry.ValidateRoleAssignment(RoleAdmin, true)
	assert.NoError(t, err, "Org role should be assignable in platform org")

	err = registry.ValidateRoleAssignment(RoleAdmin, false)
	assert.NoError(t, err, "Org role should be assignable in non-platform org")
}

func TestRoleRegistry_GetRoleHierarchy(t *testing.T) {
	registry := NewRoleRegistry()

	// Register roles with different priorities
	roles := []*RoleDefinition{
		{Name: RoleMember, Priority: RolePriorityMember, Permissions: []string{"view on profile"}},
		{Name: RoleAdmin, Priority: RolePriorityAdmin, Permissions: []string{"view on users"}},
		{Name: RoleOwner, Priority: RolePriorityOwner, Permissions: []string{"* on organization.*"}},
		{Name: RoleSuperAdmin, Priority: RolePrioritySuperAdmin, Permissions: []string{"* on *"}},
	}

	for _, role := range roles {
		err := registry.RegisterRole(role)
		require.NoError(t, err)
	}

	// Get hierarchy (should be sorted by priority descending)
	hierarchy := registry.GetRoleHierarchy()
	require.Len(t, hierarchy, 4)

	assert.Equal(t, RoleSuperAdmin, hierarchy[0].Name)
	assert.Equal(t, RoleOwner, hierarchy[1].Name)
	assert.Equal(t, RoleAdmin, hierarchy[2].Name)
	assert.Equal(t, RoleMember, hierarchy[3].Name)
}

func TestRegisterDefaultPlatformRoles(t *testing.T) {
	registry := NewRoleRegistry()

	err := RegisterDefaultPlatformRoles(registry)
	require.NoError(t, err)

	// Verify all default roles were registered
	expectedRoles := []string{RoleSuperAdmin, RoleOwner, RoleAdmin, RoleMember}
	for _, roleName := range expectedRoles {
		role, exists := registry.GetRole(roleName)
		assert.True(t, exists, "Role %s should exist", roleName)
		assert.NotNil(t, role)
		assert.NotEmpty(t, role.Description)
		assert.NotEmpty(t, role.Permissions)
	}

	// Verify role hierarchy
	hierarchy := registry.GetRoleHierarchy()
	assert.Equal(t, RoleSuperAdmin, hierarchy[0].Name)
	assert.Equal(t, RoleOwner, hierarchy[1].Name)
	assert.Equal(t, RoleAdmin, hierarchy[2].Name)
	assert.Equal(t, RoleMember, hierarchy[3].Name)

	// Verify platform flags
	superadmin, _ := registry.GetRole(RoleSuperAdmin)
	assert.True(t, superadmin.IsPlatform)

	owner, _ := registry.GetRole(RoleOwner)
	assert.False(t, owner.IsPlatform)
}
