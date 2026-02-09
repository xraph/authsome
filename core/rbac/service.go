package rbac

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
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

// NewServiceWithRepositories creates a service with repository dependencies.
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

// SetRepositories sets the repository dependencies (for services created with NewService()).
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

// LoadPolicies loads and parses all stored policy expressions from a repository.
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

// GetRoleTemplates gets all role templates for an app and environment.
func (s *Service) GetRoleTemplates(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if appID.IsNil() {
		return nil, errs.RequiredField("app_id")
	}

	if envID.IsNil() {
		return nil, errs.RequiredField("environment_id")
	}

	roles, err := s.roleRepo.GetRoleTemplates(ctx, appID, envID)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

// GetRoleTemplate gets a single role template by ID.
func (s *Service) GetRoleTemplate(ctx context.Context, roleID xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, errs.RoleNotFound(roleID.String()).WithError(err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return nil, errs.BadRequest(fmt.Sprintf("role %s is not a template", roleID.String()))
	}

	return role, nil
}

// GetRoleTemplateWithPermissions gets a role template with its permissions loaded.
func (s *Service) GetRoleTemplateWithPermissions(ctx context.Context, roleID xid.ID) (*RoleWithPermissions, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	role, err := s.roleRepo.GetOrgRoleWithPermissions(ctx, roleID)
	if err != nil {
		return nil, errs.RoleNotFound(roleID.String()).WithError(err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return nil, errs.BadRequest(fmt.Sprintf("role %s is not a template", roleID.String()))
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

// CreateRoleTemplate creates a new role template for an app.
func (s *Service) CreateRoleTemplate(ctx context.Context, appID, envID xid.ID, name, displayName, description string, isOwnerRole bool, permissionIDs []xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if appID.IsNil() {
		return nil, errs.RequiredField("app_id")
	}

	if envID.IsNil() {
		return nil, errs.RequiredField("environment_id")
	}

	if name == "" {
		return nil, errs.RequiredField("name")
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
		return nil, errs.DatabaseError("create role template", err)
	}

	// Assign permissions if provided
	if len(permissionIDs) > 0 && s.rolePermissionRepo != nil {
		if err := s.AssignPermissionsToRole(ctx, role.ID, permissionIDs); err != nil {
			// Rollback: delete the created role
			_ = s.roleRepo.Delete(ctx, role.ID)

			return nil, errs.DatabaseError("assign permissions", err)
		}
	}

	return role, nil
}

// UpdateRoleTemplate updates an existing role template.
func (s *Service) UpdateRoleTemplate(ctx context.Context, roleID xid.ID, name, displayName, description string, isOwnerRole bool, permissionIDs []xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if roleID.IsNil() {
		return nil, errs.RequiredField("role_id")
	}

	if name == "" {
		return nil, errs.RequiredField("name")
	}

	// Get the existing role
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, errs.RoleNotFound(roleID.String()).WithError(err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return nil, errs.BadRequest(fmt.Sprintf("role %s is not a template", roleID.String()))
	}

	// Verify it has an environment_id (should never be nil, but safety check)
	if role.EnvironmentID == nil || role.EnvironmentID.IsNil() {
		return nil, errs.InvalidInput("environment_id", fmt.Sprintf("role template %s has invalid environment_id", roleID.String()))
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
		return nil, errs.DatabaseError("update role template", err)
	}

	// Update permissions if provided
	if permissionIDs != nil && s.rolePermissionRepo != nil {
		if err := s.rolePermissionRepo.ReplaceRolePermissions(ctx, roleID, permissionIDs); err != nil {
			return nil, errs.DatabaseError("update role permissions", err)
		}
	}

	return role, nil
}

// DeleteRoleTemplate deletes a role template.
func (s *Service) DeleteRoleTemplate(ctx context.Context, roleID xid.ID) error {
	if s.roleRepo == nil {
		return errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Get the role first to verify it's a template
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return errs.RoleNotFound(roleID.String()).WithError(err)
	}

	// Verify it's a template
	if !role.IsTemplate {
		return errs.BadRequest(fmt.Sprintf("role %s is not a template", roleID.String()))
	}

	return s.roleRepo.Delete(ctx, roleID)
}

// GetOwnerRole gets the role marked as the owner role for an app and environment.
func (s *Service) GetOwnerRole(ctx context.Context, appID, envID xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if appID.IsNil() {
		return nil, errs.RequiredField("app_id")
	}

	if envID.IsNil() {
		return nil, errs.RequiredField("environment_id")
	}

	return s.roleRepo.GetOwnerRole(ctx, appID, envID)
}

// ====== Organization Role Management ======

// BootstrapOrgRoles clones selected role templates for a new organization.
func (s *Service) BootstrapOrgRoles(ctx context.Context, orgID, appID, envID xid.ID, templateIDs []xid.ID, customizations map[xid.ID]*RoleCustomization) error {
	if s.roleRepo == nil {
		return errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if orgID.IsNil() {
		return errs.RequiredField("organization_id")
	}

	if appID.IsNil() {
		return errs.RequiredField("app_id")
	}

	if envID.IsNil() {
		return errs.RequiredField("environment_id")
	}

	// If no template IDs provided, get all templates and clone them
	if len(templateIDs) == 0 {
		templates, err := s.roleRepo.GetRoleTemplates(ctx, appID, envID)
		if err != nil {
			return errs.DatabaseError("get role templates", err)
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
			return errs.DatabaseError("clone role template", err).WithContext("template_id", templateID.String())
		}
	}

	return nil
}

// GetOrgRoles gets all roles specific to an organization and environment.
func (s *Service) GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if orgID.IsNil() {
		return nil, errs.RequiredField("organization_id")
	}

	if envID.IsNil() {
		return nil, errs.RequiredField("environment_id")
	}

	return s.roleRepo.GetOrgRoles(ctx, orgID, envID)
}

// CreateOrgRole creates a new organization-specific role.
func (s *Service) CreateOrgRole(ctx context.Context, appID, orgID, envID xid.ID, name, displayName, description string, permissionIDs []xid.ID) (*schema.Role, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Validate required parameters
	if appID.IsNil() {
		return nil, errs.RequiredField("app_id")
	}

	if orgID.IsNil() {
		return nil, errs.RequiredField("organization_id")
	}

	if envID.IsNil() {
		return nil, errs.RequiredField("environment_id")
	}

	if name == "" {
		return nil, errs.RequiredField("name")
	}

	// Default display name from name if not provided
	if displayName == "" {
		displayName = toTitleCase(name)
	}

	// Create the organization-specific role
	role := &schema.Role{
		ID:             xid.New(),
		AppID:          &appID,
		OrganizationID: &orgID,
		EnvironmentID:  &envID,
		Name:           name,
		DisplayName:    displayName,
		Description:    description,
		IsTemplate:     false,
		IsOwnerRole:    false,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, errs.DatabaseError("create organization role", err)
	}

	// Assign permissions if provided
	if len(permissionIDs) > 0 && s.rolePermissionRepo != nil {
		if err := s.AssignPermissionsToRole(ctx, role.ID, permissionIDs); err != nil {
			// Rollback: delete the created role
			_ = s.roleRepo.Delete(ctx, role.ID)

			return nil, errs.DatabaseError("assign permissions", err)
		}
	}

	return role, nil
}

// GetOrgRoleWithPermissions gets a role with its permissions loaded.
func (s *Service) GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*RoleWithPermissions, error) {
	if s.roleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role repository not initialized")
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

// UpdateOrgRole updates an organization-specific role.
func (s *Service) UpdateOrgRole(ctx context.Context, roleID xid.ID, name, displayName, description string, permissionIDs []xid.ID) error {
	if s.roleRepo == nil {
		return errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Get the role first
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return errs.RoleNotFound(roleID.String()).WithError(err)
	}

	// Ensure this is an org-scoped role, not a template
	if role.OrganizationID == nil {
		return errs.BadRequest("cannot update template roles through this method")
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
		return errs.DatabaseError("update role", err)
	}

	// Update permissions if provided
	if permissionIDs != nil && s.rolePermissionRepo != nil {
		err = s.rolePermissionRepo.ReplaceRolePermissions(ctx, roleID, permissionIDs)
		if err != nil {
			return errs.DatabaseError("update role permissions", err)
		}
	}

	return nil
}

// DeleteOrgRole deletes an organization-specific role.
func (s *Service) DeleteOrgRole(ctx context.Context, roleID xid.ID) error {
	if s.roleRepo == nil {
		return errs.InternalServerErrorWithMessage("role repository not initialized")
	}

	// Get the role first to verify it's org-scoped
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return errs.RoleNotFound(roleID.String()).WithError(err)
	}

	// Ensure this is an org-scoped role, not a template
	if role.OrganizationID == nil {
		return errs.BadRequest("cannot delete template roles through this method")
	}

	return s.roleRepo.Delete(ctx, roleID)
}

// AssignOwnerRole assigns the owner role to a user in an organization.
func (s *Service) AssignOwnerRole(ctx context.Context, userID xid.ID, orgID xid.ID, envID xid.ID) error {
	if s.roleRepo == nil || s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("repositories not initialized")
	}

	// Get all org roles for this environment
	roles, err := s.roleRepo.GetOrgRoles(ctx, orgID, envID)
	if err != nil {
		return errs.DatabaseError("get org roles", err)
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
		return errs.RoleNotFound("owner").WithContext("organization_id", orgID.String()).WithContext("environment_id", envID.String())
	}

	// Assign the role to the user
	err = s.userRoleRepo.Assign(ctx, userID, ownerRole.ID, orgID)
	if err != nil {
		return errs.DatabaseError("assign owner role", err)
	}

	return nil
}

// ====== Permission Management ======

// GetAppPermissions gets all app-level permissions.
func (s *Service) GetAppPermissions(ctx context.Context, appID xid.ID) ([]*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("permission repository not initialized")
	}

	return s.permissionRepo.ListByApp(ctx, appID)
}

// GetOrgPermissions gets all org-specific permissions.
func (s *Service) GetOrgPermissions(ctx context.Context, orgID xid.ID) ([]*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("permission repository not initialized")
	}

	return s.permissionRepo.ListByOrg(ctx, orgID)
}

// GetUserPermissions gets all permissions for a user.
func (s *Service) GetPermission(ctx context.Context, permissionID xid.ID) (*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("permission repository not initialized")
	}

	permission, err := s.permissionRepo.FindByID(ctx, permissionID)
	if err != nil {
		return nil, errs.DatabaseError("get permission", err)
	}

	return permission, nil
}

// GetPermissionsByCategory gets permissions by category.
func (s *Service) GetPermissionsByCategory(ctx context.Context, category string, appID xid.ID) ([]*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("permission repository not initialized")
	}

	return s.permissionRepo.ListByCategory(ctx, category, appID)
}

// CreateCustomPermission creates a custom permission for an organization.
func (s *Service) CreateCustomPermission(ctx context.Context, name, description, category string, orgID xid.ID) (*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("permission repository not initialized")
	}

	return s.permissionRepo.CreateCustomPermission(ctx, name, description, category, orgID)
}

// CreateAppPermission creates an app-level custom permission
// Used for registering application-specific permissions during bootstrap
// These are marked as custom to distinguish them from Authsome's internal system permissions.
func (s *Service) CreateAppPermission(ctx context.Context, appID xid.ID, name, description, category string) (*schema.Permission, error) {
	if s.permissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("permission repository not initialized")
	}

	permission := &schema.Permission{
		ID:             xid.New(),
		AppID:          &appID,
		OrganizationID: nil, // App-level permission
		Name:           name,
		Description:    description,
		IsCustom:       true, // Mark as custom to distinguish from Authsome's internal permissions
		Category:       category,
	}

	if err := s.permissionRepo.Create(ctx, permission); err != nil {
		return nil, errs.DatabaseError("create app permission", err)
	}

	return permission, nil
}

// ====== Role-Permission Management ======

// AssignPermissionsToRole assigns permissions to a role.
func (s *Service) AssignPermissionsToRole(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	if s.rolePermissionRepo == nil {
		return errs.InternalServerErrorWithMessage("role permission repository not initialized")
	}

	for _, permID := range permissionIDs {
		err := s.rolePermissionRepo.AssignPermission(ctx, roleID, permID)
		if err != nil {
			return errs.DatabaseError("assign permission", err).WithContext("permission_id", permID.String())
		}
	}

	return nil
}

// RemovePermissionsFromRole removes permissions from a role.
func (s *Service) RemovePermissionsFromRole(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	if s.rolePermissionRepo == nil {
		return errs.InternalServerErrorWithMessage("role permission repository not initialized")
	}

	for _, permID := range permissionIDs {
		err := s.rolePermissionRepo.UnassignPermission(ctx, roleID, permID)
		if err != nil {
			return errs.DatabaseError("remove permission", err).WithContext("permission_id", permID.String())
		}
	}

	return nil
}

// GetRolePermissions gets all permissions for a role.
func (s *Service) GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error) {
	if s.rolePermissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("role permission repository not initialized")
	}

	return s.rolePermissionRepo.GetRolePermissions(ctx, roleID)
}

// toTitleCase converts a snake_case string to Title Case
// Example: "workspace_owner" -> "Workspace Owner".
func toTitleCase(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// ====== Role Assignment Methods ======

// AssignRoleToUser assigns a single role to a user in an organization.
func (s *Service) AssignRoleToUser(ctx context.Context, userID, roleID, orgID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateRoleID(roleID); err != nil {
		return err
	}

	if err := validateOrgID(orgID); err != nil {
		return err
	}

	return s.userRoleRepo.Assign(ctx, userID, roleID, orgID)
}

// AssignRolesToUser assigns multiple roles to a user in an organization.
func (s *Service) AssignRolesToUser(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateRoleIDs(roleIDs); err != nil {
		return err
	}

	if err := validateOrgID(orgID); err != nil {
		return err
	}

	return s.userRoleRepo.AssignBatch(ctx, userID, roleIDs, orgID)
}

// AssignRoleToUsers assigns a single role to multiple users in an organization.
func (s *Service) AssignRoleToUsers(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (*BulkAssignmentResult, error) {
	if s.userRoleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserIDs(userIDs); err != nil {
		return nil, err
	}

	if err := validateRoleID(roleID); err != nil {
		return nil, err
	}

	if err := validateOrgID(orgID); err != nil {
		return nil, err
	}

	errors, err := s.userRoleRepo.AssignBulk(ctx, userIDs, roleID, orgID)
	if err != nil {
		return nil, err
	}

	result := &BulkAssignmentResult{
		SuccessCount: len(userIDs) - len(errors),
		FailureCount: len(errors),
		Errors:       errors,
	}

	return result, nil
}

// AssignAppLevelRole assigns a role at app-level (not org-scoped).
func (s *Service) AssignAppLevelRole(ctx context.Context, userID, roleID, appID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateRoleID(roleID); err != nil {
		return err
	}

	if err := validateAppID(appID); err != nil {
		return err
	}

	return s.userRoleRepo.AssignAppLevel(ctx, userID, roleID, appID)
}

// ====== Role Unassignment Methods ======

// UnassignRoleFromUser removes a single role from a user in an organization.
func (s *Service) UnassignRoleFromUser(ctx context.Context, userID, roleID, orgID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateRoleID(roleID); err != nil {
		return err
	}

	if err := validateOrgID(orgID); err != nil {
		return err
	}

	return s.userRoleRepo.Unassign(ctx, userID, roleID, orgID)
}

// UnassignRolesFromUser removes multiple roles from a user in an organization.
func (s *Service) UnassignRolesFromUser(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateRoleIDs(roleIDs); err != nil {
		return err
	}

	if err := validateOrgID(orgID); err != nil {
		return err
	}

	return s.userRoleRepo.UnassignBatch(ctx, userID, roleIDs, orgID)
}

// UnassignRoleFromUsers removes a single role from multiple users in an organization.
func (s *Service) UnassignRoleFromUsers(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (*BulkAssignmentResult, error) {
	if s.userRoleRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserIDs(userIDs); err != nil {
		return nil, err
	}

	if err := validateRoleID(roleID); err != nil {
		return nil, err
	}

	if err := validateOrgID(orgID); err != nil {
		return nil, err
	}

	errors, err := s.userRoleRepo.UnassignBulk(ctx, userIDs, roleID, orgID)
	if err != nil {
		return nil, err
	}

	result := &BulkAssignmentResult{
		SuccessCount: len(userIDs) - len(errors),
		FailureCount: len(errors),
		Errors:       errors,
	}

	return result, nil
}

// ClearUserRolesInOrg removes all roles from a user in an organization.
func (s *Service) ClearUserRolesInOrg(ctx context.Context, userID, orgID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateOrgID(orgID); err != nil {
		return err
	}

	return s.userRoleRepo.ClearUserRolesInOrg(ctx, userID, orgID)
}

// ClearUserRolesInApp removes all roles from a user in an app.
func (s *Service) ClearUserRolesInApp(ctx context.Context, userID, appID xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateAppID(appID); err != nil {
		return err
	}

	return s.userRoleRepo.ClearUserRolesInApp(ctx, userID, appID)
}

// ====== Role Transfer Methods ======

// TransferUserRoles moves roles from one org to another.
func (s *Service) TransferUserRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateOrgID(sourceOrgID); err != nil {
		return err
	}

	if err := validateOrgID(targetOrgID); err != nil {
		return err
	}

	if len(roleIDs) > 0 {
		if err := validateRoleIDs(roleIDs); err != nil {
			return err
		}
	}

	return s.userRoleRepo.TransferRoles(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
}

// CopyUserRoles duplicates roles from one org to another.
func (s *Service) CopyUserRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateOrgID(sourceOrgID); err != nil {
		return err
	}

	if err := validateOrgID(targetOrgID); err != nil {
		return err
	}

	if len(roleIDs) > 0 {
		if err := validateRoleIDs(roleIDs); err != nil {
			return err
		}
	}

	return s.userRoleRepo.CopyRoles(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
}

// ReplaceUserRoles atomically replaces all user roles in an org with a new set.
func (s *Service) ReplaceUserRoles(ctx context.Context, userID, orgID xid.ID, newRoleIDs []xid.ID) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateOrgID(orgID); err != nil {
		return err
	}

	if len(newRoleIDs) > 0 {
		if err := validateRoleIDs(newRoleIDs); err != nil {
			return err
		}
	}

	return s.userRoleRepo.ReplaceUserRoles(ctx, userID, orgID, newRoleIDs)
}

// SyncRolesBetweenOrgs synchronizes roles between organizations.
func (s *Service) SyncRolesBetweenOrgs(ctx context.Context, userID xid.ID, config *RoleSyncConfig) error {
	if s.userRoleRepo == nil {
		return errs.InternalServerErrorWithMessage("user role repository not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	if config == nil {
		return errs.RequiredField("sync_config")
	}

	if err := validateOrgID(config.SourceOrgID); err != nil {
		return err
	}

	if err := validateOrgID(config.TargetOrgID); err != nil {
		return err
	}

	// Get roles from source org
	sourceRoles, err := s.userRoleRepo.ListRolesForUser(ctx, userID, &config.SourceOrgID)
	if err != nil {
		return errs.DatabaseError("list source roles", err)
	}

	// Filter by specific role IDs if provided
	var roleIDsToSync []xid.ID

	if len(config.RoleIDs) > 0 {
		roleIDMap := make(map[xid.ID]bool)
		for _, roleID := range config.RoleIDs {
			roleIDMap[roleID] = true
		}

		for _, role := range sourceRoles {
			if roleIDMap[role.ID] {
				roleIDsToSync = append(roleIDsToSync, role.ID)
			}
		}
	} else {
		for _, role := range sourceRoles {
			roleIDsToSync = append(roleIDsToSync, role.ID)
		}
	}

	if len(roleIDsToSync) == 0 {
		return nil
	}

	switch config.Mode {
	case "mirror":
		// Mirror: make target identical to source (replace all)
		return s.userRoleRepo.ReplaceUserRoles(ctx, userID, config.TargetOrgID, roleIDsToSync)
	case "merge":
		// Merge: add missing roles from source to target
		return s.userRoleRepo.CopyRoles(ctx, userID, config.SourceOrgID, config.TargetOrgID, roleIDsToSync)
	default:
		return errs.InvalidInput("mode", fmt.Sprintf("invalid sync mode: %s (must be 'mirror' or 'merge')", config.Mode))
	}
}

// ====== Access Control Helper Methods ======

// matchesPermission checks if a permission matches the requested action/resource
// Supports wildcards: "manage on *", "* on users", "* on *"
// Returns (matches, isWildcard).
func matchesPermission(perm *schema.Permission, action, resource string) (bool, bool) {
	// Try exact name match first: "view on users"
	expectedName := action + " on " + resource
	if perm.Name == expectedName {
		return true, false // exact match, not wildcard
	}

	// Try wildcard matching
	// Format: "action on resource" where either can be "*"
	parts := strings.Split(perm.Name, " on ")
	if len(parts) != 2 {
		return false, false
	}

	permAction := strings.TrimSpace(parts[0])
	permResource := strings.TrimSpace(parts[1])

	actionMatch := permAction == "*" || permAction == action
	resourceMatch := permResource == "*" || permResource == resource

	if actionMatch && resourceMatch {
		return true, true // wildcard match
	}

	return false, false
}

// ====== Role Listing Methods ======

// GetUserRolesInOrg gets all roles (with permissions) for a specific user in an organization.
func (s *Service) GetUserRolesInOrg(ctx context.Context, userID, orgID, envID xid.ID) ([]*RoleWithPermissions, error) {
	if s.userRoleRepo == nil || s.rolePermissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("repositories not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	if err := validateOrgID(orgID); err != nil {
		return nil, err
	}

	if err := validateEnvID(envID); err != nil {
		return nil, err
	}

	// Get roles from repository
	roles, err := s.userRoleRepo.ListRolesForUserInOrg(ctx, userID, orgID, envID)
	if err != nil {
		return nil, errs.DatabaseError("list roles", err)
	}

	// Load permissions for each role
	rolesWithPermissions := make([]*RoleWithPermissions, len(roles))
	for i, role := range roles {
		permissions, err := s.rolePermissionRepo.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, errs.DatabaseError("load permissions for role", err).WithContext("role_id", role.ID.String())
		}

		rolesWithPermissions[i] = &RoleWithPermissions{
			Role:        &role,
			Permissions: permissions,
		}
	}

	return rolesWithPermissions, nil
}

// GetUserRolesInApp gets all roles (with permissions) for a specific user across all orgs in an app.
func (s *Service) GetUserRolesInApp(ctx context.Context, userID, appID, envID xid.ID) ([]*RoleWithPermissions, error) {
	if s.userRoleRepo == nil || s.rolePermissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("repositories not initialized")
	}

	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	if err := validateAppID(appID); err != nil {
		return nil, err
	}

	if err := validateEnvID(envID); err != nil {
		return nil, err
	}

	// Get roles from repository
	roles, err := s.userRoleRepo.ListRolesForUserInApp(ctx, userID, appID, envID)
	if err != nil {
		return nil, errs.DatabaseError("list roles", err)
	}

	// Load permissions for each role
	rolesWithPermissions := make([]*RoleWithPermissions, len(roles))
	for i, role := range roles {
		permissions, err := s.rolePermissionRepo.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, errs.DatabaseError("load permissions for role", err).WithContext("role_id", role.ID.String())
		}

		rolesWithPermissions[i] = &RoleWithPermissions{
			Role:        &role,
			Permissions: permissions,
		}
	}

	return rolesWithPermissions, nil
}

// ListAllUserRolesInOrg lists all user-role assignments with permissions in an organization (admin view).
func (s *Service) ListAllUserRolesInOrg(ctx context.Context, orgID, envID xid.ID) ([]*UserRoleAssignment, error) {
	if s.userRoleRepo == nil || s.rolePermissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("repositories not initialized")
	}

	if err := validateOrgID(orgID); err != nil {
		return nil, err
	}

	if err := validateEnvID(envID); err != nil {
		return nil, err
	}

	// Get all user-role assignments
	userRoles, err := s.userRoleRepo.ListAllUserRolesInOrg(ctx, orgID, envID)
	if err != nil {
		return nil, errs.DatabaseError("list user roles", err)
	}

	// Group by user and load permissions
	userRoleMap := make(map[xid.ID]*UserRoleAssignment)
	for _, ur := range userRoles {
		if _, exists := userRoleMap[ur.UserID]; !exists {
			userRoleMap[ur.UserID] = &UserRoleAssignment{
				UserID:         ur.UserID,
				OrganizationID: &orgID,
				Roles:          []*RoleWithPermissions{},
			}
		}

		// Load permissions for this role
		permissions, err := s.rolePermissionRepo.GetRolePermissions(ctx, ur.RoleID)
		if err != nil {
			return nil, errs.DatabaseError("load permissions for role", err).WithContext("role_id", ur.RoleID.String())
		}

		userRoleMap[ur.UserID].Roles = append(userRoleMap[ur.UserID].Roles, &RoleWithPermissions{
			Role:        ur.Role,
			Permissions: permissions,
		})
	}

	// Convert map to slice
	result := make([]*UserRoleAssignment, 0, len(userRoleMap))
	for _, assignment := range userRoleMap {
		result = append(result, assignment)
	}

	return result, nil
}

// ListAllUserRolesInApp lists all user-role assignments with permissions across all orgs in an app (admin view).
func (s *Service) ListAllUserRolesInApp(ctx context.Context, appID, envID xid.ID) ([]*UserRoleAssignment, error) {
	if s.userRoleRepo == nil || s.rolePermissionRepo == nil {
		return nil, errs.InternalServerErrorWithMessage("repositories not initialized")
	}

	if err := validateAppID(appID); err != nil {
		return nil, err
	}

	if err := validateEnvID(envID); err != nil {
		return nil, err
	}

	// Get all user-role assignments
	userRoles, err := s.userRoleRepo.ListAllUserRolesInApp(ctx, appID, envID)
	if err != nil {
		return nil, errs.DatabaseError("list user roles", err)
	}

	// Group by user and org, load permissions
	type userOrgKey struct {
		UserID xid.ID
		OrgID  xid.ID
	}

	userRoleMap := make(map[userOrgKey]*UserRoleAssignment)

	for _, ur := range userRoles {
		key := userOrgKey{UserID: ur.UserID, OrgID: ur.AppID}
		if _, exists := userRoleMap[key]; !exists {
			orgID := ur.AppID
			userRoleMap[key] = &UserRoleAssignment{
				UserID:         ur.UserID,
				OrganizationID: &orgID,
				Roles:          []*RoleWithPermissions{},
			}
		}

		// Load permissions for this role
		permissions, err := s.rolePermissionRepo.GetRolePermissions(ctx, ur.RoleID)
		if err != nil {
			return nil, errs.DatabaseError("load permissions for role", err).WithContext("role_id", ur.RoleID.String())
		}

		userRoleMap[key].Roles = append(userRoleMap[key].Roles, &RoleWithPermissions{
			Role:        ur.Role,
			Permissions: permissions,
		})
	}

	// Convert map to slice
	result := make([]*UserRoleAssignment, 0, len(userRoleMap))
	for _, assignment := range userRoleMap {
		result = append(result, assignment)
	}

	return result, nil
}

// ====== Access Control Check Methods ======

// CheckUserAccessInOrg checks if a user has permission to perform an action on a resource in an organization
// Accepts optional pre-loaded roles/permissions for performance optimization.
func (s *Service) CheckUserAccessInOrg(
	ctx context.Context,
	userID, orgID, envID xid.ID,
	action, resource string,
	cachedRoles []*RoleWithPermissions,
) (*AccessCheckResult, error) {
	var (
		roles []*RoleWithPermissions
		err   error
	)

	// Use cached roles if provided, otherwise fetch

	if cachedRoles != nil {
		roles = cachedRoles
	} else {
		roles, err = s.GetUserRolesInOrg(ctx, userID, orgID, envID)
		if err != nil {
			return nil, errs.DatabaseError("get user roles", err)
		}
	}

	// Check each role's permissions
	for _, roleWithPerms := range roles {
		for _, perm := range roleWithPerms.Permissions {
			if matches, isWildcard := matchesPermission(perm, action, resource); matches {
				return &AccessCheckResult{
					Allowed:           true,
					Reason:            fmt.Sprintf("User has '%s' permission via role '%s'", perm.Name, roleWithPerms.Name),
					MatchedPermission: perm,
					MatchedRole:       roleWithPerms.Role,
					IsWildcard:        isWildcard,
				}, nil
			}
		}
	}

	return &AccessCheckResult{
		Allowed: false,
		Reason:  fmt.Sprintf("User does not have permission to '%s' on '%s'", action, resource),
	}, nil
}

// CheckUserAccessInApp checks if a user has permission to perform an action on a resource at app level
// Accepts optional pre-loaded roles/permissions for performance optimization.
func (s *Service) CheckUserAccessInApp(
	ctx context.Context,
	userID, appID, envID xid.ID,
	action, resource string,
	cachedRoles []*RoleWithPermissions,
) (*AccessCheckResult, error) {
	var (
		roles []*RoleWithPermissions
		err   error
	)

	// Use cached roles if provided, otherwise fetch

	if cachedRoles != nil {
		roles = cachedRoles
	} else {
		roles, err = s.GetUserRolesInApp(ctx, userID, appID, envID)
		if err != nil {
			return nil, errs.DatabaseError("get user roles", err)
		}
	}

	// Check each role's permissions
	for _, roleWithPerms := range roles {
		for _, perm := range roleWithPerms.Permissions {
			if matches, isWildcard := matchesPermission(perm, action, resource); matches {
				return &AccessCheckResult{
					Allowed:           true,
					Reason:            fmt.Sprintf("User has '%s' permission via role '%s'", perm.Name, roleWithPerms.Name),
					MatchedPermission: perm,
					MatchedRole:       roleWithPerms.Role,
					IsWildcard:        isWildcard,
				}, nil
			}
		}
	}

	return &AccessCheckResult{
		Allowed: false,
		Reason:  fmt.Sprintf("User does not have permission to '%s' on '%s'", action, resource),
	}, nil
}
