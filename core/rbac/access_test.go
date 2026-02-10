package rbac

import (
	"context"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/schema"
)

// ====== Helper Function Tests ======

func TestMatchesPermission(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		perm := &schema.Permission{Name: "view on users"}
		matches, isWildcard := matchesPermission(perm, "view", "users")
		assert.True(t, matches)
		assert.False(t, isWildcard)
	})

	t.Run("wildcard action - matches", func(t *testing.T) {
		perm := &schema.Permission{Name: "* on users"}
		matches, isWildcard := matchesPermission(perm, "view", "users")
		assert.True(t, matches)
		assert.True(t, isWildcard)

		matches, isWildcard = matchesPermission(perm, "edit", "users")
		assert.True(t, matches)
		assert.True(t, isWildcard)
	})

	t.Run("wildcard resource - matches", func(t *testing.T) {
		perm := &schema.Permission{Name: "view on *"}
		matches, isWildcard := matchesPermission(perm, "view", "users")
		assert.True(t, matches)
		assert.True(t, isWildcard)

		matches, isWildcard = matchesPermission(perm, "view", "posts")
		assert.True(t, matches)
		assert.True(t, isWildcard)
	})

	t.Run("full wildcard - matches everything", func(t *testing.T) {
		perm := &schema.Permission{Name: "* on *"}
		matches, isWildcard := matchesPermission(perm, "view", "users")
		assert.True(t, matches)
		assert.True(t, isWildcard)

		matches, isWildcard = matchesPermission(perm, "delete", "posts")
		assert.True(t, matches)
		assert.True(t, isWildcard)
	})

	t.Run("no match - different action", func(t *testing.T) {
		perm := &schema.Permission{Name: "view on users"}
		matches, isWildcard := matchesPermission(perm, "edit", "users")
		assert.False(t, matches)
		assert.False(t, isWildcard)
	})

	t.Run("no match - different resource", func(t *testing.T) {
		perm := &schema.Permission{Name: "view on users"}
		matches, isWildcard := matchesPermission(perm, "view", "posts")
		assert.False(t, matches)
		assert.False(t, isWildcard)
	})

	t.Run("invalid format - no 'on' separator", func(t *testing.T) {
		perm := &schema.Permission{Name: "view_users"}
		matches, isWildcard := matchesPermission(perm, "view", "users")
		assert.False(t, matches)
		assert.False(t, isWildcard)
	})

	t.Run("wildcard action no match - different resource", func(t *testing.T) {
		perm := &schema.Permission{Name: "* on users"}
		matches, isWildcard := matchesPermission(perm, "view", "posts")
		assert.False(t, matches)
		assert.False(t, isWildcard)
	})

	t.Run("wildcard resource no match - different action", func(t *testing.T) {
		perm := &schema.Permission{Name: "view on *"}
		matches, isWildcard := matchesPermission(perm, "edit", "users")
		assert.False(t, matches)
		assert.False(t, isWildcard)
	})
}

// ====== CheckUserAccessInOrg Tests ======

func TestService_CheckUserAccessInOrg(t *testing.T) {
	ctx := context.Background()

	t.Run("exact permission match - allowed", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		roleID := xid.New()
		permID := xid.New()

		role := schema.Role{ID: roleID, Name: "editor"}
		perm := &schema.Permission{ID: permID, Name: "view on users"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInOrg(ctx, userID, orgID, envID, "view", "users", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.False(t, result.IsWildcard)
		assert.Equal(t, permID, result.MatchedPermission.ID)
		assert.Equal(t, roleID, result.MatchedRole.ID)
		assert.Contains(t, result.Reason, "view on users")
		assert.Contains(t, result.Reason, "editor")

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("wildcard action match - allowed", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		roleID := xid.New()

		role := schema.Role{ID: roleID, Name: "admin"}
		perm := &schema.Permission{ID: xid.New(), Name: "* on users"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInOrg(ctx, userID, orgID, envID, "delete", "users", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.True(t, result.IsWildcard)
		assert.Equal(t, "* on users", result.MatchedPermission.Name)

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("wildcard resource match - allowed", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		roleID := xid.New()

		role := schema.Role{ID: roleID, Name: "viewer"}
		perm := &schema.Permission{ID: xid.New(), Name: "view on *"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInOrg(ctx, userID, orgID, envID, "view", "posts", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.True(t, result.IsWildcard)

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("full wildcard match - super admin", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		roleID := xid.New()

		role := schema.Role{ID: roleID, Name: "super_admin"}
		perm := &schema.Permission{ID: xid.New(), Name: "* on *"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInOrg(ctx, userID, orgID, envID, "delete", "anything", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.True(t, result.IsWildcard)
		assert.Equal(t, "* on *", result.MatchedPermission.Name)

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("no permission - denied", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		roleID := xid.New()

		role := schema.Role{ID: roleID, Name: "viewer"}
		perm := &schema.Permission{ID: xid.New(), Name: "view on posts"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInOrg(ctx, userID, orgID, envID, "delete", "users", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.False(t, result.IsWildcard)
		assert.Nil(t, result.MatchedPermission)
		assert.Nil(t, result.MatchedRole)
		assert.Contains(t, result.Reason, "does not have permission")

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("with cached roles - no DB call", func(t *testing.T) {
		service := &Service{}

		roleID := xid.New()
		perm := &schema.Permission{ID: xid.New(), Name: "edit on users"}
		cachedRoles := []*RoleWithPermissions{
			{
				Role:        &schema.Role{ID: roleID, Name: "editor"},
				Permissions: []*schema.Permission{perm},
			},
		}

		result, err := service.CheckUserAccessInOrg(ctx, xid.New(), xid.New(), xid.New(), "edit", "users", cachedRoles)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.False(t, result.IsWildcard)
	})

	t.Run("multiple roles - first match wins", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		role1ID := xid.New()
		role2ID := xid.New()

		role1 := schema.Role{ID: role1ID, Name: "viewer"}
		role2 := schema.Role{ID: role2ID, Name: "editor"}
		perm1 := &schema.Permission{ID: xid.New(), Name: "view on users"}
		perm2 := &schema.Permission{ID: xid.New(), Name: "edit on users"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).
			Return([]schema.Role{role1, role2}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, role1ID).
			Return([]*schema.Permission{perm1}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, role2ID).
			Return([]*schema.Permission{perm2}, nil)

		result, err := service.CheckUserAccessInOrg(ctx, userID, orgID, envID, "edit", "users", nil)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, "edit on users", result.MatchedPermission.Name)
		assert.Equal(t, "editor", result.MatchedRole.Name)

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})
}

// ====== CheckUserAccessInApp Tests ======

func TestService_CheckUserAccessInApp(t *testing.T) {
	ctx := context.Background()

	t.Run("exact permission match - allowed", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		appID := xid.New()
		envID := xid.New()
		roleID := xid.New()
		permID := xid.New()

		role := schema.Role{ID: roleID, Name: "app_admin"}
		perm := &schema.Permission{ID: permID, Name: "manage on settings"}

		mockUserRoleRepo.On("ListRolesForUserInApp", ctx, userID, appID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInApp(ctx, userID, appID, envID, "manage", "settings", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.False(t, result.IsWildcard)
		assert.Equal(t, permID, result.MatchedPermission.ID)

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("wildcard permission - allowed", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		appID := xid.New()
		envID := xid.New()
		roleID := xid.New()

		role := schema.Role{ID: roleID, Name: "super_admin"}
		perm := &schema.Permission{ID: xid.New(), Name: "* on *"}

		mockUserRoleRepo.On("ListRolesForUserInApp", ctx, userID, appID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInApp(ctx, userID, appID, envID, "delete", "anything", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.True(t, result.IsWildcard)

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("no permission - denied", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		appID := xid.New()
		envID := xid.New()
		roleID := xid.New()

		role := schema.Role{ID: roleID, Name: "viewer"}
		perm := &schema.Permission{ID: xid.New(), Name: "view on dashboard"}

		mockUserRoleRepo.On("ListRolesForUserInApp", ctx, userID, appID, envID).
			Return([]schema.Role{role}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).
			Return([]*schema.Permission{perm}, nil)

		result, err := service.CheckUserAccessInApp(ctx, userID, appID, envID, "manage", "settings", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "does not have permission")

		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("with cached roles - no DB call", func(t *testing.T) {
		service := &Service{}

		roleID := xid.New()
		perm := &schema.Permission{ID: xid.New(), Name: "manage on *"}
		cachedRoles := []*RoleWithPermissions{
			{
				Role:        &schema.Role{ID: roleID, Name: "admin"},
				Permissions: []*schema.Permission{perm},
			},
		}

		result, err := service.CheckUserAccessInApp(ctx, xid.New(), xid.New(), xid.New(), "manage", "users", cachedRoles)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.True(t, result.IsWildcard)
	})

	t.Run("no roles - denied", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		appID := xid.New()
		envID := xid.New()

		mockUserRoleRepo.On("ListRolesForUserInApp", ctx, userID, appID, envID).
			Return([]schema.Role{}, nil)

		result, err := service.CheckUserAccessInApp(ctx, userID, appID, envID, "view", "users", nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "does not have permission")

		mockUserRoleRepo.AssertExpectations(t)
	})
}
