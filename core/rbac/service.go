package rbac

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Service provides in-memory management of RBAC policies and role/permission operations.
// Storage-backed repositories can be added later via repository interfaces.
type Service struct {
	mu       sync.RWMutex
	policies []*Policy
	eval     *Evaluator

	// Repositories for role and permission management
	roleRepo           RoleRepository
	permissionRepo     PermissionRepository
	rolePermissionRepo RolePermissionRepository
	userRoleRepo       UserRoleRepository
}

func NewService() *Service {
	return &Service{eval: NewEvaluator()}
}

// NewServiceWithRepositories creates a service with repository dependencies
func NewServiceWithRepositories(
	roleRepo RoleRepository,
	permissionRepo PermissionRepository,
	rolePermissionRepo RolePermissionRepository,
	userRoleRepo UserRoleRepository,
) *Service {
	return &Service{
		eval:               NewEvaluator(),
		roleRepo:           roleRepo,
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
		userRoleRepo:       userRoleRepo,
	}
}

// SetRepositories sets the repository dependencies (for services created with NewService())
func (s *Service) SetRepositories(
	roleRepo RoleRepository,
	permissionRepo PermissionRepository,
	rolePermissionRepo RolePermissionRepository,
	userRoleRepo UserRoleRepository,
) {
	s.roleRepo = roleRepo
	s.permissionRepo = permissionRepo
	s.rolePermissionRepo = rolePermissionRepo
	s.userRoleRepo = userRoleRepo
}

func (s *Service) AddPolicy(p *Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = append(s.policies, p)
}

func (s *Service) AddExpression(expression string) error {
	parser := NewParser()
	p, err := parser.Parse(expression)
	if err != nil {
		return err
	}
	s.AddPolicy(p)
	return nil
}

// Allowed checks whether any registered policy allows the context.
func (s *Service) Allowed(ctx *Context) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.policies {
		if s.eval.Evaluate(p, ctx) {
			return true
		}
	}
	return false
}

// AllowedWithRoles checks policies against a subject plus assigned roles.
// If a policy subject is of form "role:<name>", it will be evaluated when
// that role is present in the provided roles slice.
func (s *Service) AllowedWithRoles(ctx *Context, roles []string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.policies {
		// Direct subject match
		if s.eval.Evaluate(p, ctx) {
			return true
		}
		// Role-based subject: evaluate using role subject when user has role
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(p.Subject)), "role:") {
			roleName := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(p.Subject), "role:"))
			for _, r := range roles {
				if strings.EqualFold(strings.TrimSpace(r), roleName) {
					// clone context with role subject
					rc := *ctx
					rc.Subject = p.Subject
					if s.eval.Evaluate(p, &rc) {
						return true
					}
				}
			}
		}
	}
	return false
}

// LoadPolicies loads and parses all stored policy expressions from a repository
func (s *Service) LoadPolicies(ctx context.Context, repo PolicyRepository) error {
	exprs, err := repo.ListAll(ctx)
	if err != nil {
		return err
	}
	parser := NewParser()
	for _, ex := range exprs {
		p, err := parser.Parse(ex)
		if err != nil {
			// skip invalid entries
			continue
		}
		s.AddPolicy(p)
	}
	return nil
}

// ====== Role Template Management ======

// GetRoleTemplates gets all role templates for an app and environment
func (s *Service) GetRoleTemplates(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error) {
	if s.roleRepo == nil {
		fmt.Printf("[DEBUG RBAC] GetRoleTemplates: roleRepo is nil\n")
		return nil, fmt.Errorf("role repository not initialized")
	}
	
	// Validate required parameters
	if appID.IsNil() {
		return nil, fmt.Errorf("app_id is required but was nil")
	}
	if envID.IsNil() {
		return nil, fmt.Errorf("environment_id is required but was nil when getting role templates for app %s", appID.String())
	}
	
	fmt.Printf("[DEBUG RBAC] GetRoleTemplates: calling roleRepo.GetRoleTemplates for app %s, env %s\n", appID.String(), envID.String())
	roles, err := s.roleRepo.GetRoleTemplates(ctx, appID, envID)
	if err != nil {
		fmt.Printf("[DEBUG RBAC] GetRoleTemplates: error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[DEBUG RBAC] GetRoleTemplates: found %d templates\n", len(roles))
	return roles, nil
}

// GetRoleTemplate gets a single role template by ID
func (s *Service) GetRoleTemplate(ctx context.Context, roleID xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}

	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to find role template: %w", err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return nil, fmt.Errorf("role %s is not a template", roleID.String())
	}

	return role, nil
}

// GetRoleTemplateWithPermissions gets a role template with its permissions loaded
func (s *Service) GetRoleTemplateWithPermissions(ctx context.Context, roleID xid.ID) (*RoleWithPermissions, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}

	role, err := s.roleRepo.GetOrgRoleWithPermissions(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to find role template: %w", err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return nil, fmt.Errorf("role %s is not a template", roleID.String())
	}

	// Convert []Permission to []*Permission
	permissions := make([]*schema.Permission, len(role.Permissions))
	for i := range role.Permissions {
		permissions[i] = &role.Permissions[i]
	}

	return &RoleWithPermissions{
		Role:        role,
		Permissions: permissions,
	}, nil
}

// CreateRoleTemplate creates a new role template for an app
func (s *Service) CreateRoleTemplate(ctx context.Context, appID, envID xid.ID, name, displayName, description string, isOwnerRole bool, permissionIDs []xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}

	// Validate required parameters
	if appID.IsNil() {
		return nil, fmt.Errorf("app_id is required but was nil")
	}
	if envID.IsNil() {
		return nil, fmt.Errorf("environment_id is required but was nil for role %s", name)
	}
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}

	// Default display name from name if not provided
	if displayName == "" {
		displayName = toTitleCase(name)
	}

	// Create the role template
	role := &schema.Role{
		ID:            xid.New(),
		AppID:         &appID,
		EnvironmentID: &envID,
		Name:          name,
		DisplayName:   displayName,
		Description:   description,
		IsTemplate:    true,
		IsOwnerRole:   isOwnerRole,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role template: %w", err)
	}

	// Assign permissions if provided
	if len(permissionIDs) > 0 && s.rolePermissionRepo != nil {
		if err := s.AssignPermissionsToRole(ctx, role.ID, permissionIDs); err != nil {
			// Rollback: delete the created role
			_ = s.roleRepo.Delete(ctx, role.ID)
			return nil, fmt.Errorf("failed to assign permissions: %w", err)
		}
	}

	return role, nil
}

// UpdateRoleTemplate updates an existing role template
func (s *Service) UpdateRoleTemplate(ctx context.Context, roleID xid.ID, name, displayName, description string, isOwnerRole bool, permissionIDs []xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}

	// Validate required parameters
	if roleID.IsNil() {
		return nil, fmt.Errorf("role_id is required but was nil")
	}
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}

	// Get the existing role
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to find role template: %w", err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return nil, fmt.Errorf("role %s is not a template", roleID.String())
	}
	
	// Verify it has an environment_id (should never be nil, but safety check)
	if role.EnvironmentID == nil || role.EnvironmentID.IsNil() {
		return nil, fmt.Errorf("role template %s has invalid environment_id", roleID.String())
	}

	// Update role fields
	role.Name = name
	if displayName != "" {
		role.DisplayName = displayName
	} else {
		role.DisplayName = toTitleCase(name)
	}
	role.Description = description
	role.IsOwnerRole = isOwnerRole

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to update role template: %w", err)
	}

	// Update permissions if provided
	if permissionIDs != nil && s.rolePermissionRepo != nil {
		if err := s.rolePermissionRepo.ReplaceRolePermissions(ctx, roleID, permissionIDs); err != nil {
			return nil, fmt.Errorf("failed to update role permissions: %w", err)
		}
	}

	return role, nil
}

// DeleteRoleTemplate deletes a role template
func (s *Service) DeleteRoleTemplate(ctx context.Context, roleID xid.ID) error {
	if s.roleRepo == nil {
		return fmt.Errorf("role repository not initialized")
	}

	// Get the role first to verify it's a template
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to find role template: %w", err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return fmt.Errorf("role %s is not a template", roleID.String())
	}

	return s.roleRepo.Delete(ctx, roleID)
}

// GetOwnerRole gets the role marked as the owner role for an app and environment
func (s *Service) GetOwnerRole(ctx context.Context, appID, envID xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}
	
	// Validate required parameters
	if appID.IsNil() {
		return nil, fmt.Errorf("app_id is required but was nil")
	}
	if envID.IsNil() {
		return nil, fmt.Errorf("environment_id is required but was nil when getting owner role for app %s", appID.String())
	}
	
	return s.roleRepo.GetOwnerRole(ctx, appID, envID)
}

// ====== Organization Role Management ======

// BootstrapOrgRoles clones selected role templates for a new organization
func (s *Service) BootstrapOrgRoles(ctx context.Context, orgID, appID, envID xid.ID, templateIDs []xid.ID, customizations map[xid.ID]*RoleCustomization) error {
	if s.roleRepo == nil {
		return fmt.Errorf("role repository not initialized")
	}

	// Validate required parameters
	if orgID.IsNil() {
		return fmt.Errorf("organization_id is required but was nil")
	}
	if appID.IsNil() {
		return fmt.Errorf("app_id is required but was nil")
	}
	if envID.IsNil() {
		return fmt.Errorf("environment_id is required but was nil when bootstrapping roles for org %s", orgID.String())
	}

	// If no template IDs provided, get all templates and clone them
	if len(templateIDs) == 0 {
		templates, err := s.roleRepo.GetRoleTemplates(ctx, appID, envID)
		if err != nil {
			return fmt.Errorf("failed to get role templates: %w", err)
		}

		for _, template := range templates {
			templateIDs = append(templateIDs, template.ID)
		}
	}

	// Clone each template
	for _, templateID := range templateIDs {
		customization := customizations[templateID]
		var customName *string
		if customization != nil && customization.Name != nil {
			customName = customization.Name
		}

		_, err := s.roleRepo.CloneRole(ctx, templateID, orgID, customName)
		if err != nil {
			return fmt.Errorf("failed to clone role template %s: %w", templateID.String(), err)
		}
	}

	return nil
}

// GetOrgRoles gets all roles specific to an organization and environment
func (s *Service) GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}
	
	// Validate required parameters
	if orgID.IsNil() {
		return nil, fmt.Errorf("organization_id is required but was nil")
	}
	if envID.IsNil() {
		return nil, fmt.Errorf("environment_id is required but was nil when getting roles for org %s", orgID.String())
	}
	
	return s.roleRepo.GetOrgRoles(ctx, orgID, envID)
}

// GetOrgRoleWithPermissions gets a role with its permissions loaded
func (s *Service) GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*RoleWithPermissions, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}

	role, err := s.roleRepo.GetOrgRoleWithPermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	// Convert []Permission to []*Permission
	permissions := make([]*schema.Permission, len(role.Permissions))
	for i := range role.Permissions {
		permissions[i] = &role.Permissions[i]
	}

	return &RoleWithPermissions{
		Role:        role,
		Permissions: permissions,
	}, nil
}

// UpdateOrgRole updates an organization-specific role
func (s *Service) UpdateOrgRole(ctx context.Context, roleID xid.ID, name, displayName, description string, permissionIDs []xid.ID) error {
	if s.roleRepo == nil {
		return fmt.Errorf("role repository not initialized")
	}

	// Get the role first
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to find role: %w", err)
	}

	// Ensure this is an org-scoped role, not a template
	if role.OrganizationID == nil {
		return fmt.Errorf("cannot update template roles through this method")
	}

	// Update role fields
	role.Name = name
	if displayName != "" {
		role.DisplayName = displayName
	} else {
		role.DisplayName = toTitleCase(name)
	}
	role.Description = description

	err = s.roleRepo.Update(ctx, role)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	// Update permissions if provided
	if permissionIDs != nil && s.rolePermissionRepo != nil {
		err = s.rolePermissionRepo.ReplaceRolePermissions(ctx, roleID, permissionIDs)
		if err != nil {
			return fmt.Errorf("failed to update role permissions: %w", err)
		}
	}

	return nil
}

// DeleteOrgRole deletes an organization-specific role
func (s *Service) DeleteOrgRole(ctx context.Context, roleID xid.ID) error {
	if s.roleRepo == nil {
		return fmt.Errorf("role repository not initialized")
	}

	// Get the role first to verify it's org-scoped
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to find role: %w", err)
	}

	// Ensure this is an org-scoped role, not a template
	if role.OrganizationID == nil {
		return fmt.Errorf("cannot delete template roles through this method")
	}

	return s.roleRepo.Delete(ctx, roleID)
}

// AssignOwnerRole assigns the owner role to a user in an organization
func (s *Service) AssignOwnerRole(ctx context.Context, userID xid.ID, orgID xid.ID, envID xid.ID) error {
	if s.roleRepo == nil || s.userRoleRepo == nil {
		return fmt.Errorf("repositories not initialized")
	}

	// Get all org roles for this environment
	roles, err := s.roleRepo.GetOrgRoles(ctx, orgID, envID)
	if err != nil {
		return fmt.Errorf("failed to get org roles: %w", err)
	}

	// Find the owner role (cloned from template with is_owner_role = true)
	var ownerRole *schema.Role
	for _, role := range roles {
		// Check if this role was cloned from an owner template
		if role.TemplateID != nil {
			template, err := s.roleRepo.FindByID(ctx, *role.TemplateID)
			if err == nil && template.IsOwnerRole {
				ownerRole = role
				break
			}
		}
	}

	if ownerRole == nil {
		return fmt.Errorf("owner role not found for organization %s environment %s", orgID.String(), envID.String())
	}

	// Assign the role to the user
	err = s.userRoleRepo.Assign(ctx, userID, ownerRole.ID, orgID)
	if err != nil {
		return fmt.Errorf("failed to assign owner role: %w", err)
	}

	return nil
}

// ====== Permission Management ======

// GetAppPermissions gets all app-level permissions
func (s *Service) GetAppPermissions(ctx context.Context, appID xid.ID) ([]*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, fmt.Errorf("permission repository not initialized")
	}
	return s.permissionRepo.ListByApp(ctx, appID)
}

// GetOrgPermissions gets all org-specific permissions
func (s *Service) GetOrgPermissions(ctx context.Context, orgID xid.ID) ([]*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, fmt.Errorf("permission repository not initialized")
	}
	return s.permissionRepo.ListByOrg(ctx, orgID)
}

// GetUserPermissions gets all permissions for a user
func (s *Service) GetPermission(ctx context.Context, permissionID xid.ID) (*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, fmt.Errorf("permission repository not initialized")
	}
	permission, err := s.permissionRepo.FindByID(ctx, permissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	return permission, nil
}

// GetPermissionsByCategory gets permissions by category
func (s *Service) GetPermissionsByCategory(ctx context.Context, category string, appID xid.ID) ([]*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, fmt.Errorf("permission repository not initialized")
	}
	return s.permissionRepo.ListByCategory(ctx, category, appID)
}

// CreateCustomPermission creates a custom permission for an organization
func (s *Service) CreateCustomPermission(ctx context.Context, name, description, category string, orgID xid.ID) (*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, fmt.Errorf("permission repository not initialized")
	}
	return s.permissionRepo.CreateCustomPermission(ctx, name, description, category, orgID)
}

// ====== Role-Permission Management ======

// AssignPermissionsToRole assigns permissions to a role
func (s *Service) AssignPermissionsToRole(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	if s.rolePermissionRepo == nil {
		return fmt.Errorf("role permission repository not initialized")
	}

	for _, permID := range permissionIDs {
		err := s.rolePermissionRepo.AssignPermission(ctx, roleID, permID)
		if err != nil {
			return fmt.Errorf("failed to assign permission %s: %w", permID.String(), err)
		}
	}

	return nil
}

// RemovePermissionsFromRole removes permissions from a role
func (s *Service) RemovePermissionsFromRole(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	if s.rolePermissionRepo == nil {
		return fmt.Errorf("role permission repository not initialized")
	}

	for _, permID := range permissionIDs {
		err := s.rolePermissionRepo.UnassignPermission(ctx, roleID, permID)
		if err != nil {
			return fmt.Errorf("failed to remove permission %s: %w", permID.String(), err)
		}
	}

	return nil
}

// GetRolePermissions gets all permissions for a role
func (s *Service) GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error) {
	if s.rolePermissionRepo == nil {
		return nil, fmt.Errorf("role permission repository not initialized")
	}
	return s.rolePermissionRepo.GetRolePermissions(ctx, roleID)
}

// toTitleCase converts a snake_case string to Title Case
// Example: "workspace_owner" -> "Workspace Owner"
func toTitleCase(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
