package dashboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/rbac"
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
	userRoleRepo rbac.UserRoleRepository
	roleCache    *permissionCache
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(rbacSvc *rbac.Service, userRoleRepo rbac.UserRoleRepository) *PermissionChecker {
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
	fmt.Println("CanAccess", d.Can("dashboard.view", "dashboard"), d.IsAdmin(), d.IsSuperAdmin())
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

// RegisterDashboardRoles registers dashboard-specific roles in the RoleRegistry
// This extends the default platform roles with dashboard-specific permissions
// Supports override semantics - plugins can modify other plugins' roles
func RegisterDashboardRoles(registry *rbac.RoleRegistry) error {
	// Dashboard plugin extends the default roles with dashboard-specific permissions
	// These will be merged with existing role definitions (override semantics)

	// Extend superadmin with dashboard permissions (redundant since * on * covers everything)
	// But explicit for documentation purposes
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleSuperAdmin,
		Description: rbac.RoleDescSuperAdmin,
		IsPlatform:  rbac.RoleIsPlatformSuperAdmin,
		Priority:    rbac.RolePrioritySuperAdmin,
		Permissions: []string{
			"* on *", // Unrestricted access
		},
	}); err != nil {
		return fmt.Errorf("failed to register superadmin role: %w", err)
	}

	// Extend owner with dashboard management permissions
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleOwner,
		Description: rbac.RoleDescOwner,
		IsPlatform:  rbac.RoleIsPlatformOwner,
		Priority:    rbac.RolePriorityOwner,
		Permissions: []string{
			"* on organization.*",
			"dashboard.view on dashboard",
			"view,edit,delete,create on users",
			"view,delete on sessions",
			"view on audit_logs",
			"manage on apikeys",
			"manage on settings",
		},
	}); err != nil {
		return fmt.Errorf("failed to register owner role: %w", err)
	}

	// Extend admin with dashboard permissions (inherits from member)
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:         rbac.RoleAdmin,
		Description:  rbac.RoleDescAdmin,
		IsPlatform:   rbac.RoleIsPlatformAdmin,
		InheritsFrom: rbac.RoleMember, // Inherits member permissions
		Priority:     rbac.RolePriorityAdmin,
		Permissions: []string{
			"dashboard.view on dashboard",
			"view,edit,delete,create on users",
			"view,delete on sessions",
			"view on audit_logs",
			"view,create on apikeys",
			"view on settings",
		},
	}); err != nil {
		return fmt.Errorf("failed to register admin role: %w", err)
	}

	// Extend member with basic dashboard access
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleMember,
		Description: rbac.RoleDescMember,
		IsPlatform:  rbac.RoleIsPlatformMember,
		Priority:    rbac.RolePriorityMember,
		Permissions: []string{
			"dashboard.view on dashboard",
			"view on profile",
			"edit on profile",
		},
	}); err != nil {
		return fmt.Errorf("failed to register member role: %w", err)
	}

	return nil
}

// SetupDefaultPolicies creates default RBAC policies for the dashboard
// Role hierarchy: superadmin > owner > admin > member
// This is kept for backward compatibility and immediate policy loading
// The role bootstrap system will persist these roles to the database
func SetupDefaultPolicies(rbacSvc *rbac.Service) error {
	policies := []string{
		// Superadmin role (Platform Owner - First User)
		// Has unrestricted access to everything across all organizations
		"role:" + rbac.RoleSuperAdmin + " can * on *",

		// Owner role (Organization Owner)
		// Full control over their organization and its resources
		"role:" + rbac.RoleOwner + " can * on organization.*",
		"role:" + rbac.RoleOwner + " can dashboard.view on dashboard",
		"role:" + rbac.RoleOwner + " can view,edit,delete,create on users",
		"role:" + rbac.RoleOwner + " can view,delete on sessions",
		"role:" + rbac.RoleOwner + " can view on audit_logs",
		"role:" + rbac.RoleOwner + " can manage on apikeys",
		"role:" + rbac.RoleOwner + " can manage on settings",

		// Admin role (Organization Administrator)
		// Can manage users and resources but not organization settings
		"role:" + rbac.RoleAdmin + " can dashboard.view on dashboard",
		"role:" + rbac.RoleAdmin + " can view,edit,delete,create on users",
		"role:" + rbac.RoleAdmin + " can view,delete on sessions",
		"role:" + rbac.RoleAdmin + " can view on audit_logs",
		"role:" + rbac.RoleAdmin + " can view,create on apikeys",
		"role:" + rbac.RoleAdmin + " can view on settings",

		// Member role (Regular User)
		// Basic dashboard access and self-management
		"role:" + rbac.RoleMember + " can dashboard.view on dashboard",
		"role:" + rbac.RoleMember + " can view on profile",
		"role:" + rbac.RoleMember + " can edit on profile",
	}

	for _, policy := range policies {
		if err := rbacSvc.AddExpression(policy); err != nil {
			return fmt.Errorf("failed to add policy %q: %w", policy, err)
		}
	}

	return nil
}

// EnsureFirstUserIsAdmin assigns admin role to the first user
// DEPRECATED: Use EnsureFirstUserIsSuperAdmin for first user setup
func EnsureFirstUserIsAdmin(ctx context.Context, userID, orgID xid.ID, userRoleRepo rbac.UserRoleRepository, roleRepo rbac.RoleRepository) error {
	// Check if admin role exists in the platform organization
	orgIDStr := orgID.String()
	roles, err := roleRepo.ListByOrg(ctx, &orgIDStr)
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	var adminRole *schema.Role
	for i := range roles {
		if roles[i].Name == rbac.RoleAdmin {
			adminRole = &roles[i]
			break
		}
	}

	// Create admin role if it doesn't exist
	if adminRole == nil {
		adminRole = &schema.Role{
			ID:          xid.New(),
			AppID:       &orgID,
			Name:        rbac.RoleAdmin,
			Description: rbac.RoleDescAdmin,
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

// EnsureFirstUserIsSuperAdmin assigns superadmin role to the first user
// This makes them the platform owner with full system access
func EnsureFirstUserIsSuperAdmin(ctx context.Context, userID, orgID xid.ID, userRoleRepo rbac.UserRoleRepository, roleRepo rbac.RoleRepository) error {

	// Check if superadmin role exists in the platform organization
	orgIDStr := orgID.String()
	roles, err := roleRepo.ListByOrg(ctx, &orgIDStr)
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	var superadminRole *schema.Role
	for i := range roles {
		if roles[i].Name == rbac.RoleSuperAdmin {
			superadminRole = &roles[i]
			break
		}
	}

	// Create superadmin role if it doesn't exist
	if superadminRole == nil {
		superadminRole = &schema.Role{
			ID:          xid.New(),
			AppID:       &orgID,
			Name:        rbac.RoleSuperAdmin,
			Description: rbac.RoleDescSuperAdmin,
		}
		superadminRole.CreatedBy = userID
		superadminRole.UpdatedBy = userID

		if err := roleRepo.Create(ctx, superadminRole); err != nil {
			return fmt.Errorf("failed to create superadmin role: %w", err)
		}
	}

	// Assign superadmin role to user
	if err := userRoleRepo.Assign(ctx, userID, superadminRole.ID, orgID); err != nil {
		return fmt.Errorf("failed to assign superadmin role: %w", err)
	}

	return nil
}
