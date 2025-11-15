package app

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// Mock Repositories
// =============================================================================

type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *schema.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) FindByNameAndApp(ctx context.Context, name string, appID xid.ID) (*schema.Role, error) {
	args := m.Called(ctx, name, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Role), args.Error(1)
}

func (m *MockRoleRepository) ListByOrg(ctx context.Context, orgID *string) ([]schema.Role, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]schema.Role), args.Error(1)
}

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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]schema.Role), args.Error(1)
}

type MockMemberRepository struct {
	mock.Mock
}

func (m *MockMemberRepository) CreateMember(ctx context.Context, member *schema.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) FindMemberByID(ctx context.Context, id xid.ID) (*schema.Member, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Member), args.Error(1)
}

func (m *MockMemberRepository) FindMember(ctx context.Context, appID, userID xid.ID) (*schema.Member, error) {
	args := m.Called(ctx, appID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Member), args.Error(1)
}

func (m *MockMemberRepository) UpdateMember(ctx context.Context, member *schema.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) DeleteMember(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemberRepository) ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*schema.Member], error) {
	args := m.Called(ctx, filter)
	return nil, args.Error(1)
}

func (m *MockMemberRepository) ListMembersByUser(ctx context.Context, userID xid.ID) ([]*schema.Member, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*schema.Member), args.Error(1)
}

func (m *MockMemberRepository) CountMembers(ctx context.Context, appID xid.ID) (int, error) {
	args := m.Called(ctx, appID)
	return args.Int(0), args.Error(1)
}

type MockAppRepository struct {
	mock.Mock
}

func (m *MockAppRepository) CreateApp(ctx context.Context, app *schema.App) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *MockAppRepository) GetPlatformApp(ctx context.Context) (*schema.App, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.App), args.Error(1)
}

func (m *MockAppRepository) FindAppByID(ctx context.Context, id xid.ID) (*schema.App, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.App), args.Error(1)
}

func (m *MockAppRepository) FindAppBySlug(ctx context.Context, slug string) (*schema.App, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.App), args.Error(1)
}

func (m *MockAppRepository) UpdateApp(ctx context.Context, app *schema.App) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *MockAppRepository) DeleteApp(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAppRepository) ListApps(ctx context.Context, filter *ListAppsFilter) (*pagination.PageResponse[*schema.App], error) {
	args := m.Called(ctx, filter)
	return nil, args.Error(1)
}

func (m *MockAppRepository) CountApps(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupMemberServiceForTest() (*MemberService, *MockMemberRepository, *MockAppRepository, *MockRoleRepository, *MockUserRoleRepository) {
	memberRepo := new(MockMemberRepository)
	appRepo := new(MockAppRepository)
	roleRepo := new(MockRoleRepository)
	userRoleRepo := new(MockUserRoleRepository)

	service := NewMemberService(
		memberRepo,
		appRepo,
		roleRepo,
		userRoleRepo,
		Config{},
		nil, // rbacSvc not needed for these tests
	)

	return service, memberRepo, appRepo, roleRepo, userRoleRepo
}

// =============================================================================
// Test: mapMemberRoleToRBAC
// =============================================================================

func TestMemberService_mapMemberRoleToRBAC(t *testing.T) {
	service, _, _, _, _ := setupMemberServiceForTest()

	tests := []struct {
		name       string
		memberRole string
		wantRole   string
	}{
		{
			name:       "owner maps to owner",
			memberRole: "owner",
			wantRole:   rbac.RoleOwner,
		},
		{
			name:       "admin maps to admin",
			memberRole: "admin",
			wantRole:   rbac.RoleAdmin,
		},
		{
			name:       "member maps to member",
			memberRole: "member",
			wantRole:   rbac.RoleMember,
		},
		{
			name:       "unknown maps to member (default)",
			memberRole: "unknown",
			wantRole:   rbac.RoleMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.mapMemberRoleToRBAC(tt.memberRole)
			assert.Equal(t, tt.wantRole, got)
		})
	}
}

// =============================================================================
// Test: CreateMember with RBAC Sync
// =============================================================================

func TestMemberService_CreateMember_SyncsWithRBAC(t *testing.T) {
	service, memberRepo, appRepo, roleRepo, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	appID := xid.New()
	userID := xid.New()
	roleID := xid.New()

	member := &Member{
		AppID:  appID,
		UserID: userID,
		Role:   MemberRoleAdmin,
		Status: MemberStatusActive,
	}

	// Expected: Check member count (not first user)
	memberRepo.On("CountMembers", ctx, appID).Return(5, nil)

	// Expected role lookup
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleAdmin, appID).Return(&schema.Role{
		ID:   roleID,
		Name: rbac.RoleAdmin,
	}, nil)

	// Expected no existing roles
	userRoleRepo.On("ListRolesForUser", ctx, userID, &appID).Return([]schema.Role{}, nil)

	// Expected role assignment
	userRoleRepo.On("Assign", ctx, userID, roleID, appID).Return(nil)

	// Expected member creation
	memberRepo.On("CreateMember", ctx, mock.AnythingOfType("*schema.Member")).Return(nil)

	// Expected member retrieval
	memberRepo.On("FindMemberByID", ctx, mock.AnythingOfType("xid.ID")).Return(&schema.Member{
		ID:     member.ID,
		AppID:  appID,
		UserID: userID,
		Role:   schema.MemberRole(MemberRoleAdmin),
		Status: schema.MemberStatus(MemberStatusActive),
	}, nil)

	// Execute
	created, err := service.CreateMember(ctx, member)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, MemberRoleAdmin, created.Role)

	// Verify all expectations
	appRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	userRoleRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestMemberService_CreateMember_FirstUserPromotedToOwner(t *testing.T) {
	service, memberRepo, appRepo, roleRepo, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	platformAppID := xid.New()
	userID := xid.New()
	ownerRoleID := xid.New()
	superAdminRoleID := xid.New()

	member := &Member{
		AppID:  platformAppID,
		UserID: userID,
		Role:   MemberRoleMember, // Will be promoted to owner
		Status: MemberStatusActive,
	}

	// Expected: Check member count (first user!)
	memberRepo.On("CountMembers", ctx, platformAppID).Return(0, nil)

	// Expected: Get platform app to check if this is platform app
	appRepo.On("GetPlatformApp", ctx).Return(&schema.App{
		ID:         platformAppID,
		Name:       "Platform",
		IsPlatform: true,
	}, nil)

	// Expected: Find owner role
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleOwner, platformAppID).Return(&schema.Role{
		ID:   ownerRoleID,
		Name: rbac.RoleOwner,
	}, nil)

	// Expected: Find superadmin role (for platform app first user)
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleSuperAdmin, platformAppID).Return(&schema.Role{
		ID:   superAdminRoleID,
		Name: rbac.RoleSuperAdmin,
	}, nil)

	// Expected no existing roles
	userRoleRepo.On("ListRolesForUser", ctx, userID, &platformAppID).Return([]schema.Role{}, nil)

	// Expected owner role assignment
	userRoleRepo.On("Assign", ctx, userID, ownerRoleID, platformAppID).Return(nil)

	// Expected superadmin role assignment
	userRoleRepo.On("Assign", ctx, userID, superAdminRoleID, platformAppID).Return(nil)

	// Expected member creation (with owner role!)
	memberRepo.On("CreateMember", ctx, mock.MatchedBy(func(m *schema.Member) bool {
		return m.Role == schema.MemberRole(MemberRoleOwner)
	})).Return(nil)

	// Expected member retrieval
	memberRepo.On("FindMemberByID", ctx, mock.AnythingOfType("xid.ID")).Return(&schema.Member{
		ID:     member.ID,
		AppID:  platformAppID,
		UserID: userID,
		Role:   schema.MemberRole(MemberRoleOwner), // Promoted!
		Status: schema.MemberStatus(MemberStatusActive),
	}, nil)

	// Execute
	created, err := service.CreateMember(ctx, member)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, MemberRoleOwner, created.Role) // Should be promoted to owner

	// Verify all expectations
	memberRepo.AssertExpectations(t)
	appRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	userRoleRepo.AssertExpectations(t)
}

func TestMemberService_CreateMember_RollsBackOnRBACFailure(t *testing.T) {
	service, memberRepo, _, roleRepo, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	appID := xid.New()
	userID := xid.New()
	memberID := xid.New()
	roleID := xid.New()

	member := &Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   MemberRoleAdmin,
		Status: MemberStatusActive,
	}

	// Expected: Check member count (not first user)
	memberRepo.On("CountMembers", ctx, appID).Return(3, nil)

	// Expected role lookup
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleAdmin, appID).Return(&schema.Role{
		ID:   roleID,
		Name: rbac.RoleAdmin,
	}, nil)

	// Expected no existing roles
	userRoleRepo.On("ListRolesForUser", ctx, userID, &appID).Return([]schema.Role{}, nil)

	// Expected role assignment FAILS
	userRoleRepo.On("Assign", ctx, userID, roleID, appID).Return(errors.New("rbac assignment failed"))

	// Expected member creation succeeds
	memberRepo.On("CreateMember", ctx, mock.AnythingOfType("*schema.Member")).Return(nil)

	// Expected rollback (delete member)
	memberRepo.On("DeleteMember", ctx, memberID).Return(nil)

	// Execute
	created, err := service.CreateMember(ctx, member)

	// Assert - should fail and rollback
	assert.Error(t, err)
	assert.Nil(t, created)
	assert.Contains(t, err.Error(), "failed to sync role to RBAC")

	// Verify rollback was called
	memberRepo.AssertCalled(t, "DeleteMember", ctx, memberID)
}

// =============================================================================
// Test: UpdateMember with RBAC Sync
// =============================================================================

func TestMemberService_UpdateMember_SyncsRoleChange(t *testing.T) {
	service, memberRepo, _, roleRepo, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	memberID := xid.New()
	appID := xid.New()
	userID := xid.New()
	oldRoleID := xid.New()
	newRoleID := xid.New()

	existingMember := &schema.Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   schema.MemberRole(MemberRoleMember),
	}

	updatedMember := &Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   MemberRoleAdmin, // Changed from member to admin
	}

	// Expected: Find existing member
	memberRepo.On("FindMemberByID", ctx, memberID).Return(existingMember, nil)

	// Expected: Update member
	memberRepo.On("UpdateMember", ctx, mock.AnythingOfType("*schema.Member")).Return(nil)

	// Expected: Role lookup for new role
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleAdmin, appID).Return(&schema.Role{
		ID:   newRoleID,
		Name: rbac.RoleAdmin,
	}, nil)

	// Expected: List existing roles
	userRoleRepo.On("ListRolesForUser", ctx, userID, &appID).Return([]schema.Role{
		{ID: oldRoleID, Name: rbac.RoleMember},
	}, nil)

	// Expected: Unassign old role
	userRoleRepo.On("Unassign", ctx, userID, oldRoleID, appID).Return(nil)

	// Expected: Assign new role
	userRoleRepo.On("Assign", ctx, userID, newRoleID, appID).Return(nil)

	// Execute
	err := service.UpdateMember(ctx, updatedMember)

	// Assert
	assert.NoError(t, err)

	// Verify all expectations
	memberRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	userRoleRepo.AssertExpectations(t)
}

func TestMemberService_UpdateMember_SkipsRBACIfRoleUnchanged(t *testing.T) {
	service, memberRepo, _, roleRepo, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	memberID := xid.New()
	appID := xid.New()
	userID := xid.New()

	existingMember := &schema.Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   schema.MemberRole(MemberRoleAdmin),
	}

	updatedMember := &Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   MemberRoleAdmin, // Same role
	}

	// Expected: Find existing member
	memberRepo.On("FindMemberByID", ctx, memberID).Return(existingMember, nil)

	// Expected: Update member
	memberRepo.On("UpdateMember", ctx, mock.AnythingOfType("*schema.Member")).Return(nil)

	// RBAC operations should NOT be called since role didn't change

	// Execute
	err := service.UpdateMember(ctx, updatedMember)

	// Assert
	assert.NoError(t, err)

	// Verify no RBAC calls were made
	roleRepo.AssertNotCalled(t, "FindByNameAndApp")
	userRoleRepo.AssertNotCalled(t, "ListRolesForUser")
	userRoleRepo.AssertNotCalled(t, "Unassign")
	userRoleRepo.AssertNotCalled(t, "Assign")
}

// =============================================================================
// Test: DeleteMember with RBAC Cleanup
// =============================================================================

func TestMemberService_DeleteMember_CleansUpRBAC(t *testing.T) {
	service, memberRepo, _, _, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	memberID := xid.New()
	appID := xid.New()
	userID := xid.New()
	roleID := xid.New()

	member := &schema.Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   schema.MemberRole(MemberRoleAdmin),
	}

	// Expected: Find member before deletion
	memberRepo.On("FindMemberByID", ctx, memberID).Return(member, nil)

	// Expected: Delete member
	memberRepo.On("DeleteMember", ctx, memberID).Return(nil)

	// Expected: List roles for cleanup
	userRoleRepo.On("ListRolesForUser", ctx, userID, &appID).Return([]schema.Role{
		{ID: roleID, Name: rbac.RoleAdmin},
	}, nil)

	// Expected: Unassign role
	userRoleRepo.On("Unassign", ctx, userID, roleID, appID).Return(nil)

	// Execute
	err := service.DeleteMember(ctx, memberID)

	// Assert
	assert.NoError(t, err)

	// Verify all expectations
	memberRepo.AssertExpectations(t)
	userRoleRepo.AssertExpectations(t)
}

func TestMemberService_DeleteMember_ContinuesOnRBACCleanupFailure(t *testing.T) {
	service, memberRepo, _, _, userRoleRepo := setupMemberServiceForTest()
	ctx := context.Background()

	// Test data
	memberID := xid.New()
	appID := xid.New()
	userID := xid.New()

	member := &schema.Member{
		ID:     memberID,
		AppID:  appID,
		UserID: userID,
		Role:   schema.MemberRole(MemberRoleAdmin),
	}

	// Expected: Find member before deletion
	memberRepo.On("FindMemberByID", ctx, memberID).Return(member, nil)

	// Expected: Delete member
	memberRepo.On("DeleteMember", ctx, memberID).Return(nil)

	// Expected: List roles for cleanup FAILS
	userRoleRepo.On("ListRolesForUser", ctx, userID, &appID).Return(nil, errors.New("failed to list roles"))

	// Execute - should not fail even if RBAC cleanup fails
	err := service.DeleteMember(ctx, memberID)

	// Assert - no error because member is already deleted
	assert.NoError(t, err)

	// Verify member was deleted
	memberRepo.AssertCalled(t, "DeleteMember", ctx, memberID)
}

// =============================================================================
// Test: getRoleIDByName
// =============================================================================

func TestMemberService_getRoleIDByName(t *testing.T) {
	service, _, _, roleRepo, _ := setupMemberServiceForTest()
	ctx := context.Background()

	appID := xid.New()
	roleID := xid.New()

	// Mock successful role lookup
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleAdmin, appID).Return(&schema.Role{
		ID:   roleID,
		Name: rbac.RoleAdmin,
	}, nil)

	// Execute
	gotRoleID, err := service.getRoleIDByName(ctx, appID, "admin")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, roleID, gotRoleID)
	roleRepo.AssertExpectations(t)
}

func TestMemberService_getRoleIDByName_RoleNotFound(t *testing.T) {
	service, _, _, roleRepo, _ := setupMemberServiceForTest()
	ctx := context.Background()

	appID := xid.New()

	// Mock role not found
	roleRepo.On("FindByNameAndApp", ctx, rbac.RoleAdmin, appID).Return(nil, errors.New("not found"))

	// Execute
	gotRoleID, err := service.getRoleIDByName(ctx, appID, "admin")

	// Assert
	assert.Error(t, err)
	assert.True(t, gotRoleID.IsNil())
	assert.Contains(t, err.Error(), "role admin not found")
	roleRepo.AssertExpectations(t)
}

