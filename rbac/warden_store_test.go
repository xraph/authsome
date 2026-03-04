package rbac_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	"github.com/xraph/authsome/rbac"
)

// newWardenStore creates a WardenStore backed by an in-memory Warden engine.
func newWardenStore(t *testing.T) *rbac.WardenStore {
	t.Helper()
	eng, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	return rbac.NewWardenStore(eng)
}

func TestWardenStore_RoleCRUD(t *testing.T) {
	s := newWardenStore(t)
	ctx := context.Background()
	now := time.Now()
	appID := "test-tenant"

	// Create role.
	r := &rbac.Role{
		AppID:       appID,
		Name:        "Admin",
		Slug:        "admin",
		Description: "Administrator role",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, s.CreateRole(ctx, r))
	assert.NotEmpty(t, r.ID, "CreateRole should populate the role ID")

	// Get role by ID.
	got, err := s.GetRole(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, "Admin", got.Name)
	assert.Equal(t, "admin", got.Slug)
	assert.Equal(t, appID, got.AppID)

	// Get role by slug.
	gotBySlug, err := s.GetRoleBySlug(ctx, appID, "admin")
	require.NoError(t, err)
	assert.Equal(t, r.ID, gotBySlug.ID)

	// Update role.
	r.Name = "Super Admin"
	r.UpdatedAt = time.Now()
	require.NoError(t, s.UpdateRole(ctx, r))

	got, err = s.GetRole(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "Super Admin", got.Name)

	// List roles filtered by appID.
	roles, err := s.ListRoles(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, roles, 1)

	// Delete role.
	require.NoError(t, s.DeleteRole(ctx, r.ID))

	_, err = s.GetRole(ctx, r.ID)
	assert.Error(t, err, "GetRole after delete should fail")
}

func TestWardenStore_PermissionCRUD(t *testing.T) {
	s := newWardenStore(t)
	ctx := context.Background()
	now := time.Now()
	appID := "test-tenant"

	// Create a role first (permissions are attached to roles).
	r := &rbac.Role{
		AppID:     appID,
		Name:      "Editor",
		Slug:      "editor",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateRole(ctx, r))
	require.NotEmpty(t, r.ID)

	// Add permission.
	perm := &rbac.Permission{
		RoleID:   r.ID,
		Action:   "read",
		Resource: "document",
	}
	require.NoError(t, s.AddPermission(ctx, perm))
	assert.NotEmpty(t, perm.ID, "AddPermission should populate the permission ID")

	// List role permissions.
	perms, err := s.ListRolePermissions(ctx, r.ID)
	require.NoError(t, err)
	assert.Len(t, perms, 1)
	assert.Equal(t, "read", perms[0].Action)
	assert.Equal(t, "document", perms[0].Resource)
	assert.Equal(t, r.ID, perms[0].RoleID)

	// Add a second permission.
	perm2 := &rbac.Permission{
		RoleID:   r.ID,
		Action:   "write",
		Resource: "document",
	}
	require.NoError(t, s.AddPermission(ctx, perm2))

	perms, err = s.ListRolePermissions(ctx, r.ID)
	require.NoError(t, err)
	assert.Len(t, perms, 2)

	// Remove first permission.
	require.NoError(t, s.RemovePermission(ctx, perm.ID))

	perms, err = s.ListRolePermissions(ctx, r.ID)
	require.NoError(t, err)
	assert.Len(t, perms, 1)
	assert.Equal(t, "write", perms[0].Action)
}

func TestWardenStore_Assignment(t *testing.T) {
	s := newWardenStore(t)
	ctx := context.Background()
	now := time.Now()
	appID := "test-app"

	// Create role with real AppID.
	r := &rbac.Role{
		AppID:     appID,
		Name:      "Viewer",
		Slug:      "viewer",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateRole(ctx, r))

	perm := &rbac.Permission{
		RoleID:   r.ID,
		Action:   "read",
		Resource: "page",
	}
	require.NoError(t, s.AddPermission(ctx, perm))

	// Assign role to user.
	userID := "user-42"
	ur := &rbac.UserRole{
		UserID:     userID,
		RoleID:     r.ID,
		AssignedAt: now,
	}
	require.NoError(t, s.AssignUserRole(ctx, ur))

	// ListUserRolesForApp should find the role under the correct app.
	roles, err := s.ListUserRolesForApp(ctx, appID, userID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, r.ID, roles[0].ID)
	assert.Equal(t, "Viewer", roles[0].Name)
	assert.Equal(t, appID, roles[0].AppID)

	// Unassign role.
	require.NoError(t, s.UnassignUserRole(ctx, userID, r.ID))

	roles, err = s.ListUserRolesForApp(ctx, appID, userID)
	require.NoError(t, err)
	assert.Len(t, roles, 0)
}

func TestWardenStore_HasPermission(t *testing.T) {
	s := newWardenStore(t)
	appID := "test-app"
	ctx := warden.WithTenant(context.Background(), appID, appID)
	now := time.Now()

	// Create role with real AppID.
	r := &rbac.Role{
		AppID:     appID,
		Name:      "Writer",
		Slug:      "writer",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateRole(ctx, r))

	// Add permission: write on document.
	perm := &rbac.Permission{
		RoleID:   r.ID,
		Action:   "write",
		Resource: "document",
	}
	require.NoError(t, s.AddPermission(ctx, perm))

	// Assign role to user.
	userID := "user-99"
	ur := &rbac.UserRole{
		UserID:     userID,
		RoleID:     r.ID,
		AssignedAt: now,
	}
	require.NoError(t, s.AssignUserRole(ctx, ur))

	// HasPermission should return true for exact match.
	ok, err := s.HasPermission(ctx, userID, "write", "document")
	require.NoError(t, err)
	assert.True(t, ok, "user should have write permission on document")

	// HasPermission should return false for wrong action.
	ok, err = s.HasPermission(ctx, userID, "delete", "document")
	require.NoError(t, err)
	assert.False(t, ok, "user should not have delete permission on document")

	// HasPermission should return false for wrong resource.
	ok, err = s.HasPermission(ctx, userID, "write", "user")
	require.NoError(t, err)
	assert.False(t, ok, "user should not have write permission on user")
}

func TestWardenStore_ListUserRolesForApp_Isolation(t *testing.T) {
	s := newWardenStore(t)
	ctx := context.Background()
	now := time.Now()
	app1 := "app-1"
	app2 := "app-2"

	// Create roles in different apps.
	r1 := &rbac.Role{AppID: app1, Name: "Admin", Slug: "admin", CreatedAt: now, UpdatedAt: now}
	r2 := &rbac.Role{AppID: app2, Name: "Admin", Slug: "admin", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRole(ctx, r1))
	require.NoError(t, s.CreateRole(ctx, r2))

	userID := "user-1"
	require.NoError(t, s.AssignUserRole(ctx, &rbac.UserRole{UserID: userID, RoleID: r1.ID, AssignedAt: now}))
	require.NoError(t, s.AssignUserRole(ctx, &rbac.UserRole{UserID: userID, RoleID: r2.ID, AssignedAt: now}))

	// Should only return roles for app1.
	roles, err := s.ListUserRolesForApp(ctx, app1, userID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, app1, roles[0].AppID)

	// Should only return roles for app2.
	roles, err = s.ListUserRolesForApp(ctx, app2, userID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, app2, roles[0].AppID)
}

func TestWardenStore_HasPermission_NoRoles(t *testing.T) {
	s := newWardenStore(t)
	ctx := warden.WithTenant(context.Background(), "test-app", "test-app")

	// User with no role assignments should not have any permissions.
	ok, err := s.HasPermission(ctx, "unknown-user", "read", "anything")
	require.NoError(t, err)
	assert.False(t, ok, "user with no roles should have no permissions")
}
