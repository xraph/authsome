package dashboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Permission represents a fine-grained permission check
type Permission struct {
	Action   string // e.g., "view", "edit", "delete"
	Resource string // e.g., "dashboard", "users", "sessions"
}

// PermissionChecker provides a fast, expressive API for checking permissions
type PermissionChecker struct {
	rbacSvc      *rbac.Service
	userRoleRepo *repository.UserRoleRepository
	roleCache    *permissionCache
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(rbacSvc *rbac.Service, userRoleRepo *repository.UserRoleRepository) *PermissionChecker {
	return &PermissionChecker{
		rbacSvc:      rbacSvc,
		userRoleRepo: userRoleRepo,
		roleCache:    newPermissionCache(5 * time.Minute),
	}
}

// permissionCache caches user roles to avoid repeated database queries
type permissionCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	roles     []string
	expiresAt time.Time
}

func newPermissionCache(ttl time.Duration) *permissionCache {
	c := &permissionCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}

	// Background cleanup
	go func() {
		ticker := time.NewTicker(ttl)
		defer ticker.Stop()
		for range ticker.C {
			c.cleanup()
		}
	}()

	return c
}

func (c *permissionCache) get(userID string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[userID]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.roles, true
}

func (c *permissionCache) set(userID string, roles []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[userID] = &cacheEntry{
		roles:     roles,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *permissionCache) invalidate(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}

func (c *permissionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// getUserRoles retrieves user roles with caching
func (p *PermissionChecker) getUserRoles(ctx context.Context, userID xid.ID) ([]string, error) {
	// Check cache first
	if roles, ok := p.roleCache.get(userID.String()); ok {
		return roles, nil
	}

	// Fetch from database
	roleSchemas, err := p.userRoleRepo.ListRolesForUser(ctx, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user roles: %w", err)
	}

	// Extract role names
	roles := make([]string, len(roleSchemas))
	for i, r := range roleSchemas {
		roles[i] = r.Name
	}

	// Cache the result
	p.roleCache.set(userID.String(), roles)

	return roles, nil
}

// Can checks if a user has permission to perform an action on a resource
// This is the main expressive API for permission checking
func (p *PermissionChecker) Can(ctx context.Context, userID xid.ID, action, resource string) bool {
	// Get user roles (cached)
	roles, err := p.getUserRoles(ctx, userID)
	if err != nil {
		return false
	}

	// Build RBAC context
	rbacCtx := &rbac.Context{
		Subject:  userID.String(),
		Action:   action,
		Resource: resource,
	}

	// Check permission with roles
	return p.rbacSvc.AllowedWithRoles(rbacCtx, roles)
}

// CanAny checks if a user has any of the specified permissions
func (p *PermissionChecker) CanAny(ctx context.Context, userID xid.ID, permissions ...Permission) bool {
	for _, perm := range permissions {
		if p.Can(ctx, userID, perm.Action, perm.Resource) {
			return true
		}
	}
	return false
}

// CanAll checks if a user has all of the specified permissions
func (p *PermissionChecker) CanAll(ctx context.Context, userID xid.ID, permissions ...Permission) bool {
	for _, perm := range permissions {
		if !p.Can(ctx, userID, perm.Action, perm.Resource) {
			return false
		}
	}
	return true
}

// HasRole checks if a user has a specific role
func (p *PermissionChecker) HasRole(ctx context.Context, userID xid.ID, roleName string) bool {
	roles, err := p.getUserRoles(ctx, userID)
	if err != nil {
		return false
	}

	for _, role := range roles {
		if role == roleName {
			return true
		}
	}
	return false
}

// HasAnyRole checks if a user has any of the specified roles
func (p *PermissionChecker) HasAnyRole(ctx context.Context, userID xid.ID, roleNames ...string) bool {
	roles, err := p.getUserRoles(ctx, userID)
	if err != nil {
		return false
	}

	roleSet := make(map[string]bool, len(roles))
	for _, role := range roles {
		roleSet[role] = true
	}

	for _, roleName := range roleNames {
		if roleSet[roleName] {
			return true
		}
	}
	return false
}

// InvalidateUserCache clears the cached roles for a user
// Call this when user roles are modified
func (p *PermissionChecker) InvalidateUserCache(userID xid.ID) {
	p.roleCache.invalidate(userID.String())
}

// PermissionBuilder provides a fluent API for building permission checks
type PermissionBuilder struct {
	checker *PermissionChecker
	userID  xid.ID
	ctx     context.Context
}

// For creates a new permission builder for a user
func (p *PermissionChecker) For(ctx context.Context, userID xid.ID) *PermissionBuilder {
	return &PermissionBuilder{
		checker: p,
		userID:  userID,
		ctx:     ctx,
	}
}

// Can checks a single permission
func (b *PermissionBuilder) Can(action, resource string) bool {
	return b.checker.Can(b.ctx, b.userID, action, resource)
}

// CanView is a shorthand for Can("view", resource)
func (b *PermissionBuilder) CanView(resource string) bool {
	return b.Can("view", resource)
}

// CanEdit is a shorthand for Can("edit", resource)
func (b *PermissionBuilder) CanEdit(resource string) bool {
	return b.Can("edit", resource)
}

// CanDelete is a shorthand for Can("delete", resource)
func (b *PermissionBuilder) CanDelete(resource string) bool {
	return b.Can("delete", resource)
}

// CanCreate is a shorthand for Can("create", resource)
func (b *PermissionBuilder) CanCreate(resource string) bool {
	return b.Can("create", resource)
}

// HasRole checks if the user has a specific role
func (b *PermissionBuilder) HasRole(roleName string) bool {
	return b.checker.HasRole(b.ctx, b.userID, roleName)
}

// IsAdmin checks if the user has the admin role
func (b *PermissionBuilder) IsAdmin() bool {
	return b.HasRole("admin")
}

// IsOwner checks if the user has the owner role
func (b *PermissionBuilder) IsOwner() bool {
	return b.HasRole("owner")
}

// IsSuperAdmin checks if the user has the superadmin role
func (b *PermissionBuilder) IsSuperAdmin() bool {
	return b.HasRole("superadmin")
}

// DashboardPermissions provides dashboard-specific permission checks
type DashboardPermissions struct {
	*PermissionBuilder
}

// Dashboard returns a dashboard-specific permission checker
func (b *PermissionBuilder) Dashboard() *DashboardPermissions {
	return &DashboardPermissions{PermissionBuilder: b}
}

// CanAccess checks if user can access the dashboard
func (d *DashboardPermissions) CanAccess() bool {
	return d.Can("dashboard.view", "dashboard") || d.IsAdmin() || d.IsSuperAdmin()
}

// CanManageUsers checks if user can manage users
func (d *DashboardPermissions) CanManageUsers() bool {
	return d.Can("manage", "users") || d.IsAdmin()
}

// CanViewUsers checks if user can view users
func (d *DashboardPermissions) CanViewUsers() bool {
	return d.Can("view", "users") || d.IsAdmin()
}

// CanManageSessions checks if user can manage sessions
func (d *DashboardPermissions) CanManageSessions() bool {
	return d.Can("manage", "sessions") || d.IsAdmin()
}

// CanViewSessions checks if user can view sessions
func (d *DashboardPermissions) CanViewSessions() bool {
	return d.Can("view", "sessions") || d.IsAdmin()
}

// CanViewAuditLogs checks if user can view audit logs
func (d *DashboardPermissions) CanViewAuditLogs() bool {
	return d.Can("view", "audit_logs") || d.IsSuperAdmin()
}

// SetupDefaultPolicies creates default RBAC policies for the dashboard
func SetupDefaultPolicies(rbacSvc *rbac.Service) error {
	// Admin role policies
	policies := []string{
		"role:admin can dashboard.view on dashboard",
		"role:admin can view,edit,delete,create on users",
		"role:admin can view,delete on sessions",
		"role:admin can view on audit_logs",

		// Owner role policies (inherits admin + more)
		"role:owner can dashboard.view on dashboard",
		"role:owner can view,edit,delete,create on users",
		"role:owner can view,delete on sessions",
		"role:owner can view on audit_logs",
		"role:owner can manage on system",

		// Superadmin role (full access)
		"role:superadmin can * on *",
	}

	for _, policy := range policies {
		if err := rbacSvc.AddExpression(policy); err != nil {
			return fmt.Errorf("failed to add policy %q: %w", policy, err)
		}
	}

	return nil
}

// EnsureFirstUserIsAdmin assigns admin role to the first user
func EnsureFirstUserIsAdmin(ctx context.Context, userID, orgID xid.ID, userRoleRepo *repository.UserRoleRepository, roleRepo *repository.RoleRepository) error {
	// Check if admin role exists
	roles, err := roleRepo.ListByOrg(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	var adminRole *schema.Role
	for i := range roles {
		if roles[i].Name == "admin" {
			adminRole = &roles[i]
			break
		}
	}

	// Create admin role if it doesn't exist
	if adminRole == nil {
		adminRole = &schema.Role{
			ID:          xid.New(),
			Name:        "admin",
			Description: "Dashboard Administrator",
		}
		adminRole.CreatedBy = userID
		adminRole.UpdatedBy = userID

		if err := roleRepo.Create(ctx, adminRole); err != nil {
			return fmt.Errorf("failed to create admin role: %w", err)
		}
	}

	// Assign admin role to user
	if err := userRoleRepo.Assign(ctx, userID, adminRole.ID, orgID); err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}

	return nil
}
