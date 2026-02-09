package migration

import (
	"context"
	"sync"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// RBAC SERVICE ADAPTER
// =============================================================================

// RBACServiceAdapter adapts the core rbac.Service to the migration.RBACService interface.
type RBACServiceAdapter struct {
	rbacService    *rbac.Service
	roleRepo       rbac.RoleRepository
	permissionRepo rbac.PermissionRepository
	rolePermRepo   rbac.RolePermissionRepository
	policyRepo     rbac.PolicyRepository

	// Cache for policies (they're in-memory in rbac.Service)
	mu       sync.RWMutex
	policies []*RBACPolicy
}

// RBACAdapterConfig configures the RBAC adapter.
type RBACAdapterConfig struct {
	RBACService    *rbac.Service
	RoleRepo       rbac.RoleRepository
	PermissionRepo rbac.PermissionRepository
	RolePermRepo   rbac.RolePermissionRepository
	PolicyRepo     rbac.PolicyRepository
}

// NewRBACServiceAdapter creates a new RBAC service adapter.
func NewRBACServiceAdapter(cfg RBACAdapterConfig) *RBACServiceAdapter {
	return &RBACServiceAdapter{
		rbacService:    cfg.RBACService,
		roleRepo:       cfg.RoleRepo,
		permissionRepo: cfg.PermissionRepo,
		rolePermRepo:   cfg.RolePermRepo,
		policyRepo:     cfg.PolicyRepo,
		policies:       make([]*RBACPolicy, 0),
	}
}

// GetAllPolicies returns all RBAC policies.
func (a *RBACServiceAdapter) GetAllPolicies(ctx context.Context) ([]*RBACPolicy, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If we have a policy repository, fetch from there
	if a.policyRepo != nil {
		expressions, err := a.policyRepo.ListAll(ctx)
		if err != nil {
			return nil, err
		}

		// Parse each expression into an RBACPolicy
		parser := rbac.NewParser()
		policies := make([]*RBACPolicy, 0, len(expressions))

		for _, expr := range expressions {
			parsed, err := parser.Parse(expr)
			if err != nil {
				// Skip invalid policies
				continue
			}

			policies = append(policies, &RBACPolicy{
				Subject:   parsed.Subject,
				Actions:   parsed.Actions,
				Resource:  parsed.Resource,
				Condition: parsed.Condition,
			})
		}

		return policies, nil
	}

	// Otherwise, return any manually added policies
	return a.policies, nil
}

// GetRoles returns all roles for an app and environment.
func (a *RBACServiceAdapter) GetRoles(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error) {
	if a.roleRepo == nil {
		return nil, nil
	}

	// Get role templates (app-level roles)
	templates, err := a.roleRepo.GetRoleTemplates(ctx, appID, envID)
	if err != nil {
		return nil, err
	}

	return templates, nil
}

// GetRolePermissions returns permissions for a role.
func (a *RBACServiceAdapter) GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error) {
	if a.rolePermRepo == nil {
		return nil, nil
	}

	return a.rolePermRepo.GetRolePermissions(ctx, roleID)
}

// =============================================================================
// ADDITIONAL HELPER METHODS
// =============================================================================

// GetOrgRoles returns all roles for an organization and environment.
func (a *RBACServiceAdapter) GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*schema.Role, error) {
	if a.roleRepo == nil {
		return nil, nil
	}

	return a.roleRepo.GetOrgRoles(ctx, orgID, envID)
}

// GetAllAppPermissions returns all permissions for an app.
func (a *RBACServiceAdapter) GetAllAppPermissions(ctx context.Context, appID xid.ID) ([]*schema.Permission, error) {
	if a.permissionRepo == nil {
		return nil, nil
	}

	return a.permissionRepo.ListByApp(ctx, appID)
}

// AddPolicy adds a policy to the in-memory list (for testing or manual policies).
func (a *RBACServiceAdapter) AddPolicy(policy *RBACPolicy) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.policies = append(a.policies, policy)
}

// ClearPolicies clears the in-memory policy list.
func (a *RBACServiceAdapter) ClearPolicies() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.policies = make([]*RBACPolicy, 0)
}

// =============================================================================
// USER ROLE OPERATIONS
// =============================================================================

// UserRoleAdapter provides user role operations for attribute resolution.
type UserRoleAdapter struct {
	userRoleRepo rbac.UserRoleRepository
	roleRepo     rbac.RoleRepository
	rolePermRepo rbac.RolePermissionRepository
}

// NewUserRoleAdapter creates a new user role adapter.
func NewUserRoleAdapter(
	userRoleRepo rbac.UserRoleRepository,
	roleRepo rbac.RoleRepository,
	rolePermRepo rbac.RolePermissionRepository,
) *UserRoleAdapter {
	return &UserRoleAdapter{
		userRoleRepo: userRoleRepo,
		roleRepo:     roleRepo,
		rolePermRepo: rolePermRepo,
	}
}

// GetUserRoles returns role names for a user in an organization.
func (a *UserRoleAdapter) GetUserRoles(ctx context.Context, userID, orgID xid.ID) ([]string, error) {
	if a.userRoleRepo == nil {
		return nil, nil
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	roles, err := a.userRoleRepo.ListRolesForUser(ctx, userID, orgIDPtr)
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, 0, len(roles))
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}

	return roleNames, nil
}

// GetUserPermissions returns permission names for a user based on their roles.
func (a *UserRoleAdapter) GetUserPermissions(ctx context.Context, userID, orgID xid.ID) ([]string, error) {
	if a.userRoleRepo == nil || a.rolePermRepo == nil {
		return nil, nil
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Get user's roles
	roles, err := a.userRoleRepo.ListRolesForUser(ctx, userID, orgIDPtr)
	if err != nil {
		return nil, err
	}

	// Collect all permissions from all roles
	permissionSet := make(map[string]bool)

	for _, role := range roles {
		perms, err := a.rolePermRepo.GetRolePermissions(ctx, role.ID)
		if err != nil {
			continue // Skip roles where we can't get permissions
		}

		for _, perm := range perms {
			permissionSet[perm.Name] = true
		}
	}

	// Convert to slice
	permissions := make([]string, 0, len(permissionSet))
	for permName := range permissionSet {
		permissions = append(permissions, permName)
	}

	return permissions, nil
}

// =============================================================================
// MIGRATION POLICY REPOSITORY ADAPTER
// =============================================================================

// MigrationPolicyRepoAdapter adapts the permissions storage.Repository to migration.PolicyRepository.
type MigrationPolicyRepoAdapter struct {
	repo interface {
		CreatePolicy(ctx context.Context, policy any) error
		GetPoliciesByResourceType(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, resourceType string) (any, error)
	}
}

// Ensure RBACServiceAdapter implements RBACService interface.
var _ RBACService = (*RBACServiceAdapter)(nil)
