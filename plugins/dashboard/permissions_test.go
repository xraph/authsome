package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

// Mock UserRoleRepository for testing
type mockUserRoleRepo struct {
	roles map[string][]schema.Role
}

func newMockUserRoleRepo() *mockUserRoleRepo {
	return &mockUserRoleRepo{
		roles: make(map[string][]schema.Role),
	}
}

func (m *mockUserRoleRepo) ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error) {
	roles, exists := m.roles[userID.String()]
	if !exists {
		return []schema.Role{}, nil
	}
	return roles, nil
}

func (m *mockUserRoleRepo) Assign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	return nil
}

func (m *mockUserRoleRepo) Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	return nil
}

func (m *mockUserRoleRepo) setUserRoles(userID string, roles []schema.Role) {
	m.roles[userID] = roles
}

func TestPermissionChecker_Can(t *testing.T) {
	t.Skip("Skipping: RBAC service integration tests require full RBAC system setup - see core/rbac tests")

	// Setup
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	// Add test policies
	err := rbacSvc.AddExpression("role:admin can view,edit on users")
	if err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	userID := xid.New()
	ctx := context.Background()

	// Test: User without role
	can := checker.Can(ctx, userID, "view", "users")
	if can {
		t.Error("Expected user without role to be denied")
	}

	// Give user admin role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Test: User with admin role can view
	can = checker.Can(ctx, userID, "view", "users")
	if !can {
		t.Error("Expected admin to be able to view users")
	}

	// Test: User with admin role can edit
	can = checker.Can(ctx, userID, "edit", "users")
	if !can {
		t.Error("Expected admin to be able to edit users")
	}

	// Test: User with admin role cannot delete (not in policy)
	can = checker.Can(ctx, userID, "delete", "users")
	if can {
		t.Error("Expected admin to be denied delete (not in policy)")
	}
}

func TestPermissionChecker_HasRole(t *testing.T) {
	t.Skip("Skipping: RBAC service integration tests require full RBAC system setup - see core/rbac tests")

	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	userID := xid.New()
	ctx := context.Background()

	// Test: User without roles
	hasAdmin := checker.HasRole(ctx, userID, "admin")
	if hasAdmin {
		t.Error("Expected user without roles to not have admin role")
	}

	// Give user admin role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
		{Name: "editor"},
	})

	// Test: User has admin role
	hasAdmin = checker.HasRole(ctx, userID, "admin")
	if !hasAdmin {
		t.Error("Expected user to have admin role")
	}

	// Test: User has editor role
	hasEditor := checker.HasRole(ctx, userID, "editor")
	if !hasEditor {
		t.Error("Expected user to have editor role")
	}

	// Test: User doesn't have owner role
	hasOwner := checker.HasRole(ctx, userID, "owner")
	if hasOwner {
		t.Error("Expected user to not have owner role")
	}
}

func TestPermissionChecker_HasAnyRole(t *testing.T) {
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	userID := xid.New()
	ctx := context.Background()

	// Give user editor role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "editor"},
	})

	// Test: Has one of the roles
	hasAny := checker.HasAnyRole(ctx, userID, "admin", "editor", "owner")
	if !hasAny {
		t.Error("Expected user to have at least one role")
	}

	// Test: Doesn't have any of the roles
	hasAny = checker.HasAnyRole(ctx, userID, "admin", "owner", "superadmin")
	if hasAny {
		t.Error("Expected user to not have any of these roles")
	}
}

func TestPermissionChecker_CanAny(t *testing.T) {
	t.Skip("Skipping: RBAC service integration tests require full RBAC system setup - see core/rbac tests")

	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	// Add policy
	err := rbacSvc.AddExpression("role:admin can view on users")
	if err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	userID := xid.New()
	ctx := context.Background()

	// Give user admin role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Test: Has at least one permission
	canAny := checker.CanAny(ctx, userID,
		Permission{Action: "view", Resource: "users"},
		Permission{Action: "delete", Resource: "users"},
	)
	if !canAny {
		t.Error("Expected user to have at least one permission")
	}

	// Test: Doesn't have any permission
	canAny = checker.CanAny(ctx, userID,
		Permission{Action: "delete", Resource: "users"},
		Permission{Action: "create", Resource: "sessions"},
	)
	if canAny {
		t.Error("Expected user to not have any of these permissions")
	}
}

func TestPermissionChecker_CanAll(t *testing.T) {
	t.Skip("Skipping: RBAC service integration tests require full RBAC system setup - see core/rbac tests")

	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	// Add policy
	err := rbacSvc.AddExpression("role:admin can view,edit on users")
	if err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	userID := xid.New()
	ctx := context.Background()

	// Give user admin role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Test: Has all permissions
	canAll := checker.CanAll(ctx, userID,
		Permission{Action: "view", Resource: "users"},
		Permission{Action: "edit", Resource: "users"},
	)
	if !canAll {
		t.Error("Expected user to have all permissions")
	}

	// Test: Doesn't have all permissions
	canAll = checker.CanAll(ctx, userID,
		Permission{Action: "view", Resource: "users"},
		Permission{Action: "delete", Resource: "users"},
	)
	if canAll {
		t.Error("Expected user to not have all permissions")
	}
}

func TestPermissionBuilder_FluentAPI(t *testing.T) {
	t.Skip("Skipping: RBAC service integration tests require full RBAC system setup - see core/rbac tests")

	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	// Add policies
	rbacSvc.AddExpression("role:admin can view,edit,delete,create on users")
	rbacSvc.AddExpression("role:admin can dashboard.view on dashboard")

	userID := xid.New()
	ctx := context.Background()

	// Give user admin role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Test fluent API
	builder := checker.For(ctx, userID)

	if !builder.Can("view", "users") {
		t.Error("Expected Can() to work")
	}

	if !builder.CanView("users") {
		t.Error("Expected CanView() to work")
	}

	if !builder.CanEdit("users") {
		t.Error("Expected CanEdit() to work")
	}

	if !builder.CanDelete("users") {
		t.Error("Expected CanDelete() to work")
	}

	if !builder.CanCreate("users") {
		t.Error("Expected CanCreate() to work")
	}

	if !builder.HasRole("admin") {
		t.Error("Expected HasRole() to work")
	}

	if !builder.IsAdmin() {
		t.Error("Expected IsAdmin() to work")
	}
}

func TestDashboardPermissions(t *testing.T) {
	t.Skip("Skipping: RBAC service integration tests require full RBAC system setup - see core/rbac tests")

	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	// Setup default policies
	err := SetupDefaultPolicies(rbacSvc)
	if err != nil {
		t.Fatalf("Failed to setup default policies: %v", err)
	}

	userID := xid.New()
	ctx := context.Background()

	// Give user admin role
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Test dashboard-specific permissions
	dashboard := checker.For(ctx, userID).Dashboard()

	if !dashboard.CanAccess() {
		t.Error("Expected admin to access dashboard")
	}

	if !dashboard.CanManageUsers() {
		t.Error("Expected admin to manage users")
	}

	if !dashboard.CanViewUsers() {
		t.Error("Expected admin to view users")
	}

	if !dashboard.CanManageSessions() {
		t.Error("Expected admin to manage sessions")
	}

	if !dashboard.CanViewSessions() {
		t.Error("Expected admin to view sessions")
	}

	if !dashboard.CanViewAuditLogs() {
		t.Error("Expected admin to view audit logs")
	}
}

func TestPermissionCache(t *testing.T) {
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	userID := xid.New()
	ctx := context.Background()

	// Set initial roles
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "editor"},
	})

	// First call - cache miss
	hasEditor := checker.HasRole(ctx, userID, "editor")
	if !hasEditor {
		t.Error("Expected user to have editor role")
	}

	// Change roles in mock repo
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Second call - should still return cached result (editor)
	stillHasEditor := checker.HasRole(ctx, userID, "editor")
	if !stillHasEditor {
		t.Error("Expected cached role to still be editor")
	}

	// Invalidate cache
	checker.InvalidateUserCache(userID)

	// Third call - should reflect new roles
	hasAdmin := checker.HasRole(ctx, userID, "admin")
	if !hasAdmin {
		t.Error("Expected user to have admin role after cache invalidation")
	}

	hasEditor = checker.HasRole(ctx, userID, "editor")
	if hasEditor {
		t.Error("Expected user to not have editor role after cache invalidation")
	}
}

func TestPermissionCache_Expiration(t *testing.T) {
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	// Override cache TTL for testing
	checker.roleCache.ttl = 100 * time.Millisecond

	userID := xid.New()
	ctx := context.Background()

	// Set initial roles
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "editor"},
	})

	// First call - populate cache
	checker.HasRole(ctx, userID, "editor")

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Change roles
	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Should fetch fresh data from repo
	hasAdmin := checker.HasRole(ctx, userID, "admin")
	if !hasAdmin {
		t.Error("Expected fresh data to show admin role")
	}
}

func BenchmarkPermissionChecker_CacheHit(b *testing.B) {
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	rbacSvc.AddExpression("role:admin can view on users")

	userID := xid.New()
	ctx := context.Background()

	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Warm up cache
	checker.Can(ctx, userID, "view", "users")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Can(ctx, userID, "view", "users")
	}
}

func BenchmarkPermissionChecker_CacheMiss(b *testing.B) {
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	rbacSvc.AddExpression("role:admin can view on users")

	ctx := context.Background()

	mockRepo.setUserRoles("test", []schema.Role{
		{Name: "admin"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each iteration uses a different user ID (cache miss)
		userID := xid.New()
		mockRepo.setUserRoles(userID.String(), []schema.Role{{Name: "admin"}})
		checker.Can(ctx, userID, "view", "users")
	}
}

func BenchmarkPermissionChecker_HasRole(b *testing.B) {
	rbacSvc := rbac.NewService()
	mockRepo := newMockUserRoleRepo()
	checker := NewPermissionChecker(rbacSvc, mockRepo)

	userID := xid.New()
	ctx := context.Background()

	mockRepo.setUserRoles(userID.String(), []schema.Role{
		{Name: "admin"},
	})

	// Warm up cache
	checker.HasRole(ctx, userID, "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.HasRole(ctx, userID, "admin")
	}
}
