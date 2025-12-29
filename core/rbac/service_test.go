package rbac

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/schema"
)

// Mock repositories
type MockUserRoleRepository struct {
	mock.Mock
}

func (m *MockUserRoleRepository) Assign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	args := m.Called(ctx, userID, roleID, orgID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	args := m.Called(ctx, userID, roleID, orgID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error) {
	args := m.Called(ctx, userID, orgID)
	return args.Get(0).([]schema.Role), args.Error(1)
}

func (m *MockUserRoleRepository) AssignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	args := m.Called(ctx, userID, roleIDs, orgID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) AssignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error) {
	args := m.Called(ctx, userIDs, roleID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[xid.ID]error), args.Error(1)
}

func (m *MockUserRoleRepository) AssignAppLevel(ctx context.Context, userID, roleID, appID xid.ID) error {
	args := m.Called(ctx, userID, roleID, appID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) UnassignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	args := m.Called(ctx, userID, roleIDs, orgID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) UnassignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error) {
	args := m.Called(ctx, userIDs, roleID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[xid.ID]error), args.Error(1)
}

func (m *MockUserRoleRepository) ClearUserRolesInOrg(ctx context.Context, userID, orgID xid.ID) error {
	args := m.Called(ctx, userID, orgID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) ClearUserRolesInApp(ctx context.Context, userID, appID xid.ID) error {
	args := m.Called(ctx, userID, appID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) TransferRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	args := m.Called(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
	return args.Error(0)
}

func (m *MockUserRoleRepository) CopyRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	args := m.Called(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
	return args.Error(0)
}

func (m *MockUserRoleRepository) ReplaceUserRoles(ctx context.Context, userID, orgID xid.ID, newRoleIDs []xid.ID) error {
	args := m.Called(ctx, userID, orgID, newRoleIDs)
	return args.Error(0)
}

func (m *MockUserRoleRepository) ListRolesForUserInOrg(ctx context.Context, userID, orgID, envID xid.ID) ([]schema.Role, error) {
	args := m.Called(ctx, userID, orgID, envID)
	return args.Get(0).([]schema.Role), args.Error(1)
}

func (m *MockUserRoleRepository) ListRolesForUserInApp(ctx context.Context, userID, appID, envID xid.ID) ([]schema.Role, error) {
	args := m.Called(ctx, userID, appID, envID)
	return args.Get(0).([]schema.Role), args.Error(1)
}

func (m *MockUserRoleRepository) ListAllUserRolesInOrg(ctx context.Context, orgID, envID xid.ID) ([]schema.UserRole, error) {
	args := m.Called(ctx, orgID, envID)
	return args.Get(0).([]schema.UserRole), args.Error(1)
}

func (m *MockUserRoleRepository) ListAllUserRolesInApp(ctx context.Context, appID, envID xid.ID) ([]schema.UserRole, error) {
	args := m.Called(ctx, appID, envID)
	return args.Get(0).([]schema.UserRole), args.Error(1)
}

type MockRolePermissionRepository struct {
	mock.Mock
}

func (m *MockRolePermissionRepository) AssignPermission(ctx context.Context, roleID, permissionID xid.ID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) UnassignPermission(ctx context.Context, roleID, permissionID xid.ID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Permission), args.Error(1)
}

func (m *MockRolePermissionRepository) GetPermissionRoles(ctx context.Context, permissionID xid.ID) ([]*schema.Role, error) {
	args := m.Called(ctx, permissionID)
	return args.Get(0).([]*schema.Role), args.Error(1)
}

func (m *MockRolePermissionRepository) ReplaceRolePermissions(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	args := m.Called(ctx, roleID, permissionIDs)
	return args.Error(0)
}

// ====== Assignment Tests ======

func TestService_AssignRoleToUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		roleID := xid.New()
		orgID := xid.New()

		mockUserRoleRepo.On("Assign", ctx, userID, roleID, orgID).Return(nil)

		err := service.AssignRoleToUser(ctx, userID, roleID, orgID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("nil repository", func(t *testing.T) {
		service := &Service{}
		err := service.AssignRoleToUser(ctx, xid.New(), xid.New(), xid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.AssignRoleToUser(ctx, xid.ID{}, xid.New(), xid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("invalid role ID", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.AssignRoleToUser(ctx, xid.New(), xid.ID{}, xid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role_id")
	})

	t.Run("invalid org ID", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.AssignRoleToUser(ctx, xid.New(), xid.New(), xid.ID{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization_id")
	})
}

func TestService_AssignRolesToUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		roleIDs := []xid.ID{xid.New(), xid.New(), xid.New()}
		orgID := xid.New()

		mockUserRoleRepo.On("AssignBatch", ctx, userID, roleIDs, orgID).Return(nil)

		err := service.AssignRolesToUser(ctx, userID, roleIDs, orgID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("empty role IDs", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.AssignRolesToUser(ctx, xid.New(), []xid.ID{}, xid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one role_id")
	})

	t.Run("nil role ID in slice", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		roleIDs := []xid.ID{xid.New(), xid.ID{}, xid.New()}
		err := service.AssignRolesToUser(ctx, xid.New(), roleIDs, xid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role_id at index")
	})
}

func TestService_AssignRoleToUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("success - all users", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userIDs := []xid.ID{xid.New(), xid.New(), xid.New()}
		roleID := xid.New()
		orgID := xid.New()

		mockUserRoleRepo.On("AssignBulk", ctx, userIDs, roleID, orgID).Return(nil, nil)

		result, err := service.AssignRoleToUsers(ctx, userIDs, roleID, orgID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 3, result.SuccessCount)
		assert.Equal(t, 0, result.FailureCount)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("partial success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userIDs := []xid.ID{xid.New(), xid.New(), xid.New()}
		roleID := xid.New()
		orgID := xid.New()

		errors := map[xid.ID]error{
			userIDs[1]: fmt.Errorf("duplicate assignment"),
		}
		mockUserRoleRepo.On("AssignBulk", ctx, userIDs, roleID, orgID).Return(errors, nil)

		result, err := service.AssignRoleToUsers(ctx, userIDs, roleID, orgID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.SuccessCount)
		assert.Equal(t, 1, result.FailureCount)
		assert.Len(t, result.Errors, 1)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("empty user IDs", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		result, err := service.AssignRoleToUsers(ctx, []xid.ID{}, xid.New(), xid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "at least one user_id")
	})
}

func TestService_AssignAppLevelRole(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		roleID := xid.New()
		appID := xid.New()

		mockUserRoleRepo.On("AssignAppLevel", ctx, userID, roleID, appID).Return(nil)

		err := service.AssignAppLevelRole(ctx, userID, roleID, appID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("invalid app ID", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.AssignAppLevelRole(ctx, xid.New(), xid.New(), xid.ID{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app_id")
	})
}

// ====== Unassignment Tests ======

func TestService_UnassignRoleFromUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		roleID := xid.New()
		orgID := xid.New()

		mockUserRoleRepo.On("Unassign", ctx, userID, roleID, orgID).Return(nil)

		err := service.UnassignRoleFromUser(ctx, userID, roleID, orgID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("nil repository", func(t *testing.T) {
		service := &Service{}
		err := service.UnassignRoleFromUser(ctx, xid.New(), xid.New(), xid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

func TestService_UnassignRolesFromUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		roleIDs := []xid.ID{xid.New(), xid.New()}
		orgID := xid.New()

		mockUserRoleRepo.On("UnassignBatch", ctx, userID, roleIDs, orgID).Return(nil)

		err := service.UnassignRolesFromUser(ctx, userID, roleIDs, orgID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

func TestService_UnassignRoleFromUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("success - all users", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userIDs := []xid.ID{xid.New(), xid.New()}
		roleID := xid.New()
		orgID := xid.New()

		mockUserRoleRepo.On("UnassignBulk", ctx, userIDs, roleID, orgID).Return(nil, nil)

		result, err := service.UnassignRoleFromUsers(ctx, userIDs, roleID, orgID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.SuccessCount)
		assert.Equal(t, 0, result.FailureCount)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("partial success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userIDs := []xid.ID{xid.New(), xid.New()}
		roleID := xid.New()
		orgID := xid.New()

		errors := map[xid.ID]error{
			userIDs[0]: fmt.Errorf("not found"),
		}
		mockUserRoleRepo.On("UnassignBulk", ctx, userIDs, roleID, orgID).Return(errors, nil)

		result, err := service.UnassignRoleFromUsers(ctx, userIDs, roleID, orgID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 1, result.FailureCount)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

func TestService_ClearUserRolesInOrg(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		orgID := xid.New()

		mockUserRoleRepo.On("ClearUserRolesInOrg", ctx, userID, orgID).Return(nil)

		err := service.ClearUserRolesInOrg(ctx, userID, orgID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

func TestService_ClearUserRolesInApp(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		appID := xid.New()

		mockUserRoleRepo.On("ClearUserRolesInApp", ctx, userID, appID).Return(nil)

		err := service.ClearUserRolesInApp(ctx, userID, appID)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

// ====== Transfer Tests ======

func TestService_TransferUserRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()
		roleIDs := []xid.ID{xid.New(), xid.New()}

		mockUserRoleRepo.On("TransferRoles", ctx, userID, sourceOrgID, targetOrgID, roleIDs).Return(nil)

		err := service.TransferUserRoles(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("empty role IDs", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()

		mockUserRoleRepo.On("TransferRoles", ctx, userID, sourceOrgID, targetOrgID, []xid.ID{}).Return(nil)

		err := service.TransferUserRoles(ctx, userID, sourceOrgID, targetOrgID, []xid.ID{})
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("invalid source org", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.TransferUserRoles(ctx, xid.New(), xid.ID{}, xid.New(), []xid.ID{xid.New()})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source org")
	})

	t.Run("invalid target org", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.TransferUserRoles(ctx, xid.New(), xid.New(), xid.ID{}, []xid.ID{xid.New()})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target org")
	})
}

func TestService_CopyUserRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()
		roleIDs := []xid.ID{xid.New()}

		mockUserRoleRepo.On("CopyRoles", ctx, userID, sourceOrgID, targetOrgID, roleIDs).Return(nil)

		err := service.CopyUserRoles(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

func TestService_ReplaceUserRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("success with new roles", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		orgID := xid.New()
		newRoleIDs := []xid.ID{xid.New(), xid.New()}

		mockUserRoleRepo.On("ReplaceUserRoles", ctx, userID, orgID, newRoleIDs).Return(nil)

		err := service.ReplaceUserRoles(ctx, userID, orgID, newRoleIDs)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("success with empty roles (clear all)", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		orgID := xid.New()

		mockUserRoleRepo.On("ReplaceUserRoles", ctx, userID, orgID, []xid.ID{}).Return(nil)

		err := service.ReplaceUserRoles(ctx, userID, orgID, []xid.ID{})
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

func TestService_SyncRolesBetweenOrgs(t *testing.T) {
	ctx := context.Background()

	t.Run("mirror mode - success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()
		roleIDs := []xid.ID{xid.New(), xid.New()}

		config := &RoleSyncConfig{
			SourceOrgID: sourceOrgID,
			TargetOrgID: targetOrgID,
			RoleIDs:     roleIDs,
			Mode:        "mirror",
		}

		sourceRoles := []schema.Role{
			{ID: roleIDs[0], Name: "admin"},
			{ID: roleIDs[1], Name: "editor"},
		}

		mockUserRoleRepo.On("ListRolesForUser", ctx, userID, &sourceOrgID).Return(sourceRoles, nil)
		mockUserRoleRepo.On("ReplaceUserRoles", ctx, userID, targetOrgID, roleIDs).Return(nil)

		err := service.SyncRolesBetweenOrgs(ctx, userID, config)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("merge mode - success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()

		config := &RoleSyncConfig{
			SourceOrgID: sourceOrgID,
			TargetOrgID: targetOrgID,
			RoleIDs:     []xid.ID{}, // sync all
			Mode:        "merge",
		}

		roleID1 := xid.New()
		roleID2 := xid.New()
		sourceRoles := []schema.Role{
			{ID: roleID1, Name: "admin"},
			{ID: roleID2, Name: "editor"},
		}

		mockUserRoleRepo.On("ListRolesForUser", ctx, userID, &sourceOrgID).Return(sourceRoles, nil)
		mockUserRoleRepo.On("CopyRoles", ctx, userID, sourceOrgID, targetOrgID, []xid.ID{roleID1, roleID2}).Return(nil)

		err := service.SyncRolesBetweenOrgs(ctx, userID, config)
		assert.NoError(t, err)
		mockUserRoleRepo.AssertExpectations(t)
	})

	t.Run("nil config", func(t *testing.T) {
		service := &Service{userRoleRepo: new(MockUserRoleRepository)}
		err := service.SyncRolesBetweenOrgs(ctx, xid.New(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("invalid mode", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()

		config := &RoleSyncConfig{
			SourceOrgID: sourceOrgID,
			TargetOrgID: targetOrgID,
			Mode:        "invalid",
		}

		sourceRoles := []schema.Role{{ID: xid.New()}}
		mockUserRoleRepo.On("ListRolesForUser", ctx, userID, &sourceOrgID).Return(sourceRoles, nil)

		err := service.SyncRolesBetweenOrgs(ctx, userID, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid sync mode")
	})

	t.Run("no roles to sync", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		service := &Service{userRoleRepo: mockUserRoleRepo}

		userID := xid.New()
		sourceOrgID := xid.New()
		targetOrgID := xid.New()

		config := &RoleSyncConfig{
			SourceOrgID: sourceOrgID,
			TargetOrgID: targetOrgID,
			RoleIDs:     []xid.ID{xid.New()}, // specific role that user doesn't have
			Mode:        "mirror",
		}

		mockUserRoleRepo.On("ListRolesForUser", ctx, userID, &sourceOrgID).Return([]schema.Role{}, nil)

		err := service.SyncRolesBetweenOrgs(ctx, userID, config)
		assert.NoError(t, err) // No error, just nothing to sync
		mockUserRoleRepo.AssertExpectations(t)
	})
}

// ====== Listing Tests ======

func TestService_GetUserRolesInOrg(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		userID := xid.New()
		orgID := xid.New()
		envID := xid.New()
		roleID1 := xid.New()
		roleID2 := xid.New()

		roles := []schema.Role{
			{ID: roleID1, Name: "admin"},
			{ID: roleID2, Name: "editor"},
		}

		perm1 := &schema.Permission{ID: xid.New(), Name: "users:read"}
		perm2 := &schema.Permission{ID: xid.New(), Name: "users:write"}

		mockUserRoleRepo.On("ListRolesForUserInOrg", ctx, userID, orgID, envID).Return(roles, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID1).Return([]*schema.Permission{perm1}, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID2).Return([]*schema.Permission{perm2}, nil)

		result, err := service.GetUserRolesInOrg(ctx, userID, orgID, envID)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "admin", result[0].Name)
		assert.Len(t, result[0].Permissions, 1)
		assert.Equal(t, "users:read", result[0].Permissions[0].Name)
		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("nil repositories", func(t *testing.T) {
		service := &Service{}
		result, err := service.GetUserRolesInOrg(ctx, xid.New(), xid.New(), xid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("invalid env ID", func(t *testing.T) {
		service := &Service{
			userRoleRepo:       new(MockUserRoleRepository),
			rolePermissionRepo: new(MockRolePermissionRepository),
		}
		result, err := service.GetUserRolesInOrg(ctx, xid.New(), xid.New(), xid.ID{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "environment_id")
	})
}

func TestService_GetUserRolesInApp(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
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

		roles := []schema.Role{
			{ID: roleID, Name: "admin"},
		}

		perm := &schema.Permission{ID: xid.New(), Name: "users:manage"}

		mockUserRoleRepo.On("ListRolesForUserInApp", ctx, userID, appID, envID).Return(roles, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).Return([]*schema.Permission{perm}, nil)

		result, err := service.GetUserRolesInApp(ctx, userID, appID, envID)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "admin", result[0].Name)
		assert.Len(t, result[0].Permissions, 1)
		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})
}

func TestService_ListAllUserRolesInOrg(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		orgID := xid.New()
		envID := xid.New()
		user1ID := xid.New()
		user2ID := xid.New()
		role1ID := xid.New()
		role2ID := xid.New()

		userRoles := []schema.UserRole{
			{UserID: user1ID, RoleID: role1ID, AppID: orgID, Role: &schema.Role{ID: role1ID, Name: "admin"}},
			{UserID: user1ID, RoleID: role2ID, AppID: orgID, Role: &schema.Role{ID: role2ID, Name: "editor"}},
			{UserID: user2ID, RoleID: role1ID, AppID: orgID, Role: &schema.Role{ID: role1ID, Name: "admin"}},
		}

		perm1 := &schema.Permission{ID: xid.New(), Name: "perm1"}
		perm2 := &schema.Permission{ID: xid.New(), Name: "perm2"}

		mockUserRoleRepo.On("ListAllUserRolesInOrg", ctx, orgID, envID).Return(userRoles, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, role1ID).Return([]*schema.Permission{perm1}, nil).Times(2)
		mockRolePermRepo.On("GetRolePermissions", ctx, role2ID).Return([]*schema.Permission{perm2}, nil)

		result, err := service.ListAllUserRolesInOrg(ctx, orgID, envID)
		assert.NoError(t, err)
		assert.Len(t, result, 2) // 2 users
		
		// Verify grouping by user
		var user1Assignment, user2Assignment *UserRoleAssignment
		for _, assignment := range result {
			if assignment.UserID == user1ID {
				user1Assignment = assignment
			} else if assignment.UserID == user2ID {
				user2Assignment = assignment
			}
		}

		assert.NotNil(t, user1Assignment)
		assert.NotNil(t, user2Assignment)
		assert.Len(t, user1Assignment.Roles, 2) // user1 has 2 roles
		assert.Len(t, user2Assignment.Roles, 1) // user2 has 1 role
		
		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		orgID := xid.New()
		envID := xid.New()

		mockUserRoleRepo.On("ListAllUserRolesInOrg", ctx, orgID, envID).Return([]schema.UserRole{}, nil)

		result, err := service.ListAllUserRolesInOrg(ctx, orgID, envID)
		assert.NoError(t, err)
		assert.Len(t, result, 0)
		mockUserRoleRepo.AssertExpectations(t)
	})
}

func TestService_ListAllUserRolesInApp(t *testing.T) {
	ctx := context.Background()

	t.Run("success with multiple orgs", func(t *testing.T) {
		mockUserRoleRepo := new(MockUserRoleRepository)
		mockRolePermRepo := new(MockRolePermissionRepository)
		service := &Service{
			userRoleRepo:       mockUserRoleRepo,
			rolePermissionRepo: mockRolePermRepo,
		}

		appID := xid.New()
		envID := xid.New()
		org1ID := xid.New()
		org2ID := xid.New()
		userID := xid.New()
		roleID := xid.New()

		userRoles := []schema.UserRole{
			{UserID: userID, RoleID: roleID, AppID: org1ID, Role: &schema.Role{ID: roleID, Name: "admin"}},
			{UserID: userID, RoleID: roleID, AppID: org2ID, Role: &schema.Role{ID: roleID, Name: "admin"}},
		}

		perm := &schema.Permission{ID: xid.New(), Name: "perm"}

		mockUserRoleRepo.On("ListAllUserRolesInApp", ctx, appID, envID).Return(userRoles, nil)
		mockRolePermRepo.On("GetRolePermissions", ctx, roleID).Return([]*schema.Permission{perm}, nil).Times(2)

		result, err := service.ListAllUserRolesInApp(ctx, appID, envID)
		assert.NoError(t, err)
		assert.Len(t, result, 2) // 2 org assignments for same user
		
		mockUserRoleRepo.AssertExpectations(t)
		mockRolePermRepo.AssertExpectations(t)
	})
}

