package rbac

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// RoleDefinition declares a role and its permissions
// Plugins register these during Init() to contribute to the platform RBAC system.
type RoleDefinition struct {
	Name         string   // Role name (e.g., "superadmin", "owner", "admin", "member")
	DisplayName  string   // Human-readable name (e.g., "Super Administrator")
	Description  string   // Human-readable description
	Permissions  []string // Permission expressions: "action on resource" or "* on *"
	IsPlatform   bool     // Platform-level role (superadmin) vs org-level (owner, admin, member)
	IsTemplate   bool     // Whether this role should be available as a template for organizations
	IsOwnerRole  bool     // Whether this is the default owner role for new organizations
	InheritsFrom string   // Parent role to inherit permissions from (for role hierarchy)
	Priority     int      // Higher priority roles override lower priority (superadmin=100, owner=80, admin=60, member=40)
}

// RoleRegistryInterface defines the contract for role registration and management
// This interface enables:
// - Mock implementations for testing
// - Alternative implementations (cached, remote, etc.)
// - Dependency injection and loose coupling.
type RoleRegistryInterface interface {
	// RegisterRole registers or updates a role definition
	// Override semantics: If a role with the same name exists, permissions are merged
	RegisterRole(role *RoleDefinition) error

	// GetRole retrieves a role definition by name
	GetRole(name string) (*RoleDefinition, bool)

	// ListRoles returns all registered role definitions
	ListRoles() []*RoleDefinition

	// Bootstrap applies all registered roles to the platform app
	// Called once during server startup after database migrations and plugin initialization
	Bootstrap(ctx context.Context, db *bun.DB, rbacService *Service, platformAppID xid.ID) error

	// ValidateRoleAssignment checks if a role can be assigned to a user in an app
	// Platform roles (IsPlatform=true) can only be assigned in the platform app
	ValidateRoleAssignment(roleName string, isPlatformApp bool) error

	// GetRoleHierarchy returns roles in descending priority order (highest first)
	GetRoleHierarchy() []*RoleDefinition
}

// RoleRegistry collects role definitions from core and plugins
// Supports:
// - Override semantics (later registrations override earlier ones)
// - Role inheritance (roles inherit from parent roles)
// - Cross-plugin modification (plugins can extend other plugins' roles).
type RoleRegistry struct {
	roles map[string]*RoleDefinition
}

// NewRoleRegistry creates a new role registry.
func NewRoleRegistry() *RoleRegistry {
	return &RoleRegistry{
		roles: make(map[string]*RoleDefinition),
	}
}

// RegisterRole registers or updates a role definition
// Override semantics: If a role with the same name exists, permissions are merged
// with the new permissions taking precedence.
func (r *RoleRegistry) RegisterRole(role *RoleDefinition) error {
	if role.Name == "" {
		return errs.RequiredField("name")
	}

	existingRole, exists := r.roles[role.Name]
	if exists {
		// Override semantics: merge permissions, with new ones taking precedence
		// Create a map of existing permissions for deduplication
		permMap := make(map[string]bool)
		for _, perm := range existingRole.Permissions {
			permMap[perm] = true
		}

		// Add new permissions (they override/extend existing)
		for _, perm := range role.Permissions {
			permMap[perm] = true
		}

		// Rebuild permissions list
		mergedPerms := make([]string, 0, len(permMap))
		for perm := range permMap {
			mergedPerms = append(mergedPerms, perm)
		}

		// Update the role with merged permissions and new metadata
		existingRole.Permissions = mergedPerms
		existingRole.Description = role.Description // Override description
		existingRole.IsPlatform = role.IsPlatform   // Override platform flag
		existingRole.InheritsFrom = role.InheritsFrom
		existingRole.Priority = role.Priority
	} else {
		// New role - register it
		r.roles[role.Name] = role
	}

	return nil
}

// GetRole retrieves a role definition by name.
func (r *RoleRegistry) GetRole(name string) (*RoleDefinition, bool) {
	role, exists := r.roles[name]

	return role, exists
}

// ListRoles returns all registered role definitions.
func (r *RoleRegistry) ListRoles() []*RoleDefinition {
	roles := make([]*RoleDefinition, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}

	return roles
}

// Bootstrap applies all registered roles to the platform app
// Called once during server startup AFTER:
// - Database migrations have run
// - Plugins have initialized and registered their roles
//
// This creates/updates:
// 1. Role records in the database
// 2. Permission records in the database
// 3. RBAC policy expressions in the policy engine.
func (r *RoleRegistry) Bootstrap(ctx context.Context, db *bun.DB, rbacService *Service, platformAppID xid.ID) error {
	// Get the default environment for this app
	var defaultEnvID xid.ID

	err := db.NewSelect().
		Table("environments").
		Column("id").
		Where("app_id = ?", platformAppID).
		Where("is_default = ?", true).
		Limit(1).
		Scan(ctx, &defaultEnvID)
	if err != nil {
		// If no default environment found, get the first environment
		err = db.NewSelect().
			Table("environments").
			Column("id").
			Where("app_id = ?", platformAppID).
			Order("created_at ASC").
			Limit(1).
			Scan(ctx, &defaultEnvID)
		if err != nil {
			return fmt.Errorf("no environment found for app %s: %w", platformAppID.String(), err)
		}
	}

	// Resolve role inheritance and build final permission sets
	resolvedRoles, err := r.resolveInheritance()
	if err != nil {
		return fmt.Errorf("failed to resolve role inheritance: %w", err)
	}

	// Collect all unique permissions across all roles
	permissionMap := make(map[string]bool)

	for _, roleDef := range resolvedRoles {
		for _, perm := range roleDef.Permissions {
			permissionMap[perm] = true
		}
	}

	// Upsert permissions in database
	for permName := range permissionMap {
		if err := r.upsertPermission(ctx, db, platformAppID, permName); err != nil {
			return fmt.Errorf("failed to upsert permission %s: %w", permName, err)
		}
	}

	// Upsert roles in database
	for _, roleDef := range resolvedRoles {
		if err := r.upsertRole(ctx, db, platformAppID, defaultEnvID, roleDef); err != nil {
			return fmt.Errorf("failed to upsert role %s: %w", roleDef.Name, err)
		}
	}

	// Note: App-level roles are NOT templates - they are the actual roles for the app
	// Templates are separate entities that organizations can clone if needed
	// The is_template, is_owner_role fields are for future use with explicit template creation

	// Apply RBAC policies to the policy engine
	if rbacService != nil {
		for _, roleDef := range resolvedRoles {
			for _, perm := range roleDef.Permissions {
				// Convert permission to policy expression: "role:name can action on resource"
				policyExpr := fmt.Sprintf("role:%s can %s", roleDef.Name, perm)
				if err := rbacService.AddExpression(policyExpr); err != nil {
					// Log error but don't fail bootstrap - some expressions may be invalid
					_ = err
				}
			}
		}
	}

	return nil
}

// resolveInheritance builds the final permission set for each role by resolving inheritance chains
// Handles:
// - Single inheritance (InheritsFrom)
// - Circular dependency detection
// - Priority-based ordering.
func (r *RoleRegistry) resolveInheritance() ([]*RoleDefinition, error) {
	resolved := make(map[string]*RoleDefinition)
	visiting := make(map[string]bool) // For cycle detection

	var resolve func(name string) (*RoleDefinition, error)

	resolve = func(name string) (*RoleDefinition, error) {
		// Already resolved
		if resolved[name] != nil {
			return resolved[name], nil
		}

		// Cycle detection
		if visiting[name] {
			return nil, fmt.Errorf("circular role inheritance detected: %s", name)
		}

		role, exists := r.roles[name]
		if !exists {
			return nil, fmt.Errorf("role %s not found", name)
		}

		visiting[name] = true
		defer delete(visiting, name)

		// Create a copy to avoid modifying the original
		resolvedRole := &RoleDefinition{
			Name:         role.Name,
			Description:  role.Description,
			IsPlatform:   role.IsPlatform,
			InheritsFrom: role.InheritsFrom,
			Priority:     role.Priority,
			Permissions:  make([]string, len(role.Permissions)),
		}
		copy(resolvedRole.Permissions, role.Permissions)

		// Resolve parent if specified
		if role.InheritsFrom != "" {
			parent, err := resolve(role.InheritsFrom)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve parent role %s for %s: %w", role.InheritsFrom, name, err)
			}

			// Inherit permissions from parent (child permissions override parent)
			parentPerms := make(map[string]bool)
			for _, perm := range parent.Permissions {
				parentPerms[perm] = true
			}

			// Add child permissions (they override/extend parent)
			for _, perm := range resolvedRole.Permissions {
				parentPerms[perm] = true
			}

			// Rebuild permissions list
			resolvedRole.Permissions = make([]string, 0, len(parentPerms))
			for perm := range parentPerms {
				resolvedRole.Permissions = append(resolvedRole.Permissions, perm)
			}
		}

		resolved[name] = resolvedRole

		return resolvedRole, nil
	}

	// Resolve all roles
	for name := range r.roles {
		if _, err := resolve(name); err != nil {
			return nil, err
		}
	}

	// Convert map to slice
	result := make([]*RoleDefinition, 0, len(resolved))
	for _, role := range resolved {
		result = append(result, role)
	}

	return result, nil
}

// upsertPermission creates or updates a permission in the database.
func (r *RoleRegistry) upsertPermission(ctx context.Context, db *bun.DB, appID xid.ID, permissionExpr string) error {
	// Find existing permission
	var existingPerm schema.Permission

	err := db.NewSelect().
		Model(&existingPerm).
		Where("name = ?", permissionExpr).
		Where("app_id IS NULL OR app_id = ?", appID).
		Scan(ctx)

	now := time.Now()

	if err != nil {
		// Permission doesn't exist - create it
		newPerm := &schema.Permission{
			ID:          xid.New(),
			AppID:       &appID,
			Name:        permissionExpr,
			Description: "Permission: " + permissionExpr,
		}
		newPerm.CreatedAt = now
		newPerm.UpdatedAt = now
		newPerm.CreatedBy = appID
		newPerm.UpdatedBy = appID
		newPerm.Version = 1

		_, err = db.NewInsert().Model(newPerm).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert permission: %w", err)
		}
	}

	return nil
}

// upsertRole creates or updates a role in the database.
func (r *RoleRegistry) upsertRole(ctx context.Context, db *bun.DB, appID, envID xid.ID, def *RoleDefinition) error {
	// Validate environment ID
	if envID.IsNil() {
		return fmt.Errorf("environment_id is required but was nil for role %s", def.Name)
	}

	// Find existing role
	var existingRole schema.Role

	err := db.NewSelect().
		Model(&existingRole).
		Where("name = ?", def.Name).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id IS NULL").
		Scan(ctx)

	now := time.Now()

	// Default display name from name if not provided
	displayName := def.DisplayName
	if displayName == "" {
		displayName = toTitleCase(def.Name)
	}

	if err != nil {
		// Role doesn't exist - create it
		newRole := &schema.Role{
			ID:            xid.New(),
			AppID:         &appID,
			EnvironmentID: &envID,
			Name:          def.Name,
			DisplayName:   displayName,
			Description:   def.Description,
			IsTemplate:    def.IsTemplate,
			IsOwnerRole:   def.IsOwnerRole,
		}
		newRole.CreatedAt = now
		newRole.UpdatedAt = now
		newRole.CreatedBy = appID // Platform app is the creator
		newRole.UpdatedBy = appID
		newRole.Version = 1

		_, err = db.NewInsert().Model(newRole).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert role: %w", err)
		}
	} else {
		// Role exists - update it
		existingRole.DisplayName = displayName
		existingRole.Description = def.Description
		existingRole.IsTemplate = def.IsTemplate
		existingRole.IsOwnerRole = def.IsOwnerRole
		existingRole.UpdatedAt = now
		existingRole.UpdatedBy = appID
		existingRole.Version++

		_, err = db.NewUpdate().
			Model(&existingRole).
			Column("display_name", "description", "is_template", "is_owner_role", "updated_at", "updated_by", "version").
			Where("id = ?", existingRole.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}
	}

	return nil
}

// ValidateRoleAssignment checks if a role can be assigned to a user in an app
// Platform roles (IsPlatform=true) can only be assigned in the platform app.
func (r *RoleRegistry) ValidateRoleAssignment(roleName string, isPlatformApp bool) error {
	role, exists := r.roles[roleName]
	if !exists {
		return fmt.Errorf("role %s not registered", roleName)
	}

	if role.IsPlatform && !isPlatformApp {
		return fmt.Errorf("role %s is a platform role and can only be assigned in the platform app", roleName)
	}

	return nil
}

// GetRoleHierarchy returns roles in descending priority order (highest first).
func (r *RoleRegistry) GetRoleHierarchy() []*RoleDefinition {
	roles := r.ListRoles()

	// Sort by priority (descending)
	for i := range roles {
		for j := i + 1; j < len(roles); j++ {
			if roles[j].Priority > roles[i].Priority {
				roles[i], roles[j] = roles[j], roles[i]
			}
		}
	}

	return roles
}

// RegisterDefaultPlatformRoles registers the default platform-wide roles
// This is called during AuthSome initialization before plugins register their roles
// Plugins can then extend or override these default roles.
func RegisterDefaultPlatformRoles(registry *RoleRegistry) error {
	// Superadmin - Platform owner with unrestricted access
	// NOT a template - this is platform-only and cannot be cloned to organizations
	if err := registry.RegisterRole(&RoleDefinition{
		Name:        RoleSuperAdmin,
		DisplayName: "Super Administrator",
		Description: RoleDescSuperAdmin,
		IsPlatform:  RoleIsPlatformSuperAdmin,
		IsTemplate:  false, // Platform-only, not a template
		IsOwnerRole: false,
		Priority:    RolePrioritySuperAdmin,
		Permissions: []string{
			"* on *", // Unrestricted access to everything
		},
	}); err != nil {
		return err
	}

	// Owner - Organization owner with full org control
	// This IS a template that can be cloned to organizations
	if err := registry.RegisterRole(&RoleDefinition{
		Name:        RoleOwner,
		DisplayName: "Owner",
		Description: RoleDescOwner,
		IsPlatform:  RoleIsPlatformOwner,
		IsTemplate:  true, // Available as template for organizations
		IsOwnerRole: true, // This is the default owner role for new organizations
		Priority:    RolePriorityOwner,
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
		return err
	}

	// Admin - Organization administrator
	// This IS a template that can be cloned to organizations
	if err := registry.RegisterRole(&RoleDefinition{
		Name:         RoleAdmin,
		DisplayName:  "Administrator",
		Description:  RoleDescAdmin,
		IsPlatform:   RoleIsPlatformAdmin,
		IsTemplate:   true, // Available as template for organizations
		IsOwnerRole:  false,
		InheritsFrom: RoleMember, // Inherits member permissions
		Priority:     RolePriorityAdmin,
		Permissions: []string{
			"dashboard.view on dashboard",
			"view,edit,delete,create on users",
			"view,delete on sessions",
			"view on audit_logs",
			"view,create on apikeys",
			"view on settings",
		},
	}); err != nil {
		return err
	}

	// Member - Regular user
	// This IS a template that can be cloned to organizations
	if err := registry.RegisterRole(&RoleDefinition{
		Name:        RoleMember,
		DisplayName: "Member",
		Description: RoleDescMember,
		IsPlatform:  RoleIsPlatformMember,
		IsTemplate:  true, // Available as template for organizations
		IsOwnerRole: false,
		Priority:    RolePriorityMember,
		Permissions: []string{
			"dashboard.view on dashboard",
			"view on profile",
			"edit on profile",
		},
	}); err != nil {
		return err
	}

	return nil
}

// expandPermissions is a helper function to parse permission strings like "view,edit,delete on resource"
// Returns individual permission expressions
// This is currently unused but may be needed for future permission expansion features.
func expandPermissions(permExpr string) []string {
	parts := strings.Split(permExpr, " on ")
	if len(parts) != 2 {
		return []string{permExpr} // Return as-is if not in expected format
	}

	actions := strings.Split(parts[0], ",")
	resource := strings.TrimSpace(parts[1])

	expanded := make([]string, 0, len(actions))
	for _, action := range actions {
		action = strings.TrimSpace(action)
		expanded = append(expanded, fmt.Sprintf("%s on %s", action, resource))
	}

	return expanded
}
