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

// GetRoleTemplates gets all role templates for an app
func (s *Service) GetRoleTemplates(ctx context.Context, appID xid.ID) ([]*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}
	return s.roleRepo.GetRoleTemplates(ctx, appID)
}

// GetOwnerRole gets the role marked as the owner role for an app
func (s *Service) GetOwnerRole(ctx context.Context, appID xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}
	return s.roleRepo.GetOwnerRole(ctx, appID)
}

// ====== Organization Role Management ======

// BootstrapOrgRoles clones selected role templates for a new organization
func (s *Service) BootstrapOrgRoles(ctx context.Context, orgID xid.ID, templateIDs []xid.ID, customizations map[xid.ID]*RoleCustomization) error {
	if s.roleRepo == nil {
		return fmt.Errorf("role repository not initialized")
	}

	// If no template IDs provided, get all templates and clone them
	if len(templateIDs) == 0 {
		// Get platform app ID from first template
		templates, err := s.roleRepo.GetRoleTemplates(ctx, xid.ID{}) // TODO: pass correct appID
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

// GetOrgRoles gets all roles specific to an organization
func (s *Service) GetOrgRoles(ctx context.Context, orgID xid.ID) ([]*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, fmt.Errorf("role repository not initialized")
	}
	return s.roleRepo.GetOrgRoles(ctx, orgID)
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
func (s *Service) UpdateOrgRole(ctx context.Context, roleID xid.ID, name, description string, permissionIDs []xid.ID) error {
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
func (s *Service) AssignOwnerRole(ctx context.Context, userID xid.ID, orgID xid.ID) error {
	if s.roleRepo == nil || s.userRoleRepo == nil {
		return fmt.Errorf("repositories not initialized")
	}

	// Get all org roles
	roles, err := s.roleRepo.GetOrgRoles(ctx, orgID)
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
		return fmt.Errorf("owner role not found for organization %s", orgID.String())
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
