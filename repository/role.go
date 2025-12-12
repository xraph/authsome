package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// RoleRepository provides basic CRUD for roles
type RoleRepository struct{ db *bun.DB }

func NewRoleRepository(db *bun.DB) *RoleRepository { return &RoleRepository{db: db} }

func (r *RoleRepository) Create(ctx context.Context, role *schema.Role) error {
	// Populate required auditable fields to satisfy NOT NULL constraints
	if role.ID.IsNil() {
		role.ID = xid.New()
	}
	if role.AuditableModel.CreatedBy.IsNil() {
		role.AuditableModel.CreatedBy = xid.New()
	}
	if role.AuditableModel.UpdatedBy.IsNil() {
		role.AuditableModel.UpdatedBy = role.AuditableModel.CreatedBy
	}
	
	// Ensure environment_id is set
	if role.EnvironmentID == nil || role.EnvironmentID.IsNil() {
		return fmt.Errorf("environment_id is required but was nil for role '%s' (app_id: %v, org_id: %v)", 
			role.Name, 
			role.AppID,
			role.OrganizationID)
	}
	
	// Ensure app_id is set
	if role.AppID == nil || role.AppID.IsNil() {
		return fmt.Errorf("app_id is required but was nil for role '%s'", role.Name)
	}
	
	// Ensure display_name is set, default from name if empty
	if role.DisplayName == "" {
		role.DisplayName = toTitleCase(role.Name)
	}
	
	_, err := r.db.NewInsert().Model(role).Exec(ctx)
	return err
}

func (r *RoleRepository) Update(ctx context.Context, role *schema.Role) error {
	now := time.Now()
	role.UpdatedAt = now
	_, err := r.db.NewUpdate().
		Model(role).
		WherePK().
		Exec(ctx)
	return err
}

func (r *RoleRepository) Delete(ctx context.Context, roleID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Role)(nil)).
		Where("id = ?", roleID).
		Exec(ctx)
	return err
}

func (r *RoleRepository) FindByID(ctx context.Context, roleID xid.ID) (*schema.Role, error) {
	var role schema.Role
	err := r.db.NewSelect().
		Model(&role).
		Where("id = ?", roleID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) ListByOrg(ctx context.Context, orgID *string) ([]schema.Role, error) {
	var rows []schema.Role
	q := r.db.NewSelect().Model(&rows)
	if orgID != nil {
		q = q.Where("app_id = ?", *orgID)
	}
	err := q.Scan(ctx)
	return rows, err
}

// FindByNameAndApp finds a role by name within an app (deprecated, use FindByNameAppEnv)
func (r *RoleRepository) FindByNameAndApp(ctx context.Context, name string, appID xid.ID) (*schema.Role, error) {
	var role schema.Role
	err := r.db.NewSelect().
		Model(&role).
		Where("name = ?", name).
		Where("app_id = ?", appID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindByNameAppEnv finds a role by name, app, and environment
func (r *RoleRepository) FindByNameAppEnv(ctx context.Context, name string, appID, envID xid.ID) (*schema.Role, error) {
	var role schema.Role
	err := r.db.NewSelect().
		Model(&role).
		Where("name = ?", name).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetRoleTemplates gets all role templates for an app (templates have organization_id = NULL and is_template = true)
func (r *RoleRepository) GetRoleTemplates(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role
	query := r.db.NewSelect().
		Model(&roles).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id IS NULL").
		Where("is_template = ?", true).
		Order("name ASC")

	fmt.Printf("[DEBUG REPO] GetRoleTemplates SQL: %s, appID: %s, envID: %s\n", query.String(), appID.String(), envID.String())

	err := query.Scan(ctx)
	if err != nil {
		fmt.Printf("[DEBUG REPO] GetRoleTemplates error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[DEBUG REPO] GetRoleTemplates found %d roles\n", len(roles))
	return roles, nil
}

// GetOwnerRole gets the role marked as the owner role for an app
func (r *RoleRepository) GetOwnerRole(ctx context.Context, appID, envID xid.ID) (*schema.Role, error) {
	var role schema.Role
	err := r.db.NewSelect().
		Model(&role).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id IS NULL").
		Where("is_owner_role = ?", true).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("owner role not found for app %s environment %s", appID.String(), envID.String())
		}
		return nil, err
	}
	return &role, nil
}

// GetOrgRoles gets all roles specific to an organization
func (r *RoleRepository) GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role
	err := r.db.NewSelect().
		Model(&roles).
		Where("organization_id = ?", orgID).
		Where("environment_id = ?", envID).
		Order("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetOrgRoleWithPermissions gets a role with its permissions loaded
func (r *RoleRepository) GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*schema.Role, error) {
	var role schema.Role
	err := r.db.NewSelect().
		Model(&role).
		Where("id = ?", roleID).
		Relation("Permissions").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CloneRole clones a role template for an organization
func (r *RoleRepository) CloneRole(ctx context.Context, templateID xid.ID, orgID xid.ID, customName *string) (*schema.Role, error) {
	// Get the template role with its permissions
	var template schema.Role
	err := r.db.NewSelect().
		Model(&template).
		Where("id = ?", templateID).
		Relation("Permissions").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find template role: %w", err)
	}

	// Determine the name for the cloned role
	roleName := template.Name
	if customName != nil && *customName != "" {
		roleName = *customName
	}

	// Check if role already exists for this organization (idempotent behavior)
	var existingRole schema.Role
	err = r.db.NewSelect().
		Model(&existingRole).
		Where("app_id = ?", template.AppID).
		Where("environment_id = ?", template.EnvironmentID).
		Where("organization_id = ?", orgID).
		Where("name = ?", roleName).
		Where("is_template = ?", false).
		Scan(ctx)

	if err == nil {
		// Role already exists, return it (idempotent)
		fmt.Printf("[RoleRepository] Role '%s' already exists for org %s, returning existing role\n", roleName, orgID.String())
		
		// Load permissions for the existing role
		err = r.db.NewSelect().
			Model(&existingRole).
			Where("id = ?", existingRole.ID).
			Relation("Permissions").
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load existing role permissions: %w", err)
		}
		
		return &existingRole, nil
	}

	// Create the new role based on the template
	now := time.Now()
	newRole := &schema.Role{
		ID:             xid.New(),
		AppID:          template.AppID,
		EnvironmentID:  template.EnvironmentID, // Copy environment from template
		OrganizationID: &orgID,
		Name:           roleName,
		DisplayName:    template.DisplayName, // Copy display name from template
		Description:    template.Description,
		IsTemplate:     false,
		IsOwnerRole:    false,
		TemplateID:     &templateID,
	}

	// Apply custom display name if custom name was provided
	if customName != nil && *customName != "" {
		newRole.DisplayName = toTitleCase(*customName)
	}

	// Set auditable fields
	newRole.CreatedAt = now
	newRole.UpdatedAt = now
	newRole.CreatedBy = orgID // Use org ID as creator for now
	newRole.UpdatedBy = orgID
	newRole.Version = 1

	// Insert the new role
	fmt.Printf("[RoleRepository] Creating new role '%s' for org %s (template: %s)\n", roleName, orgID.String(), templateID.String())
	_, err = r.db.NewInsert().Model(newRole).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloned role '%s' for org %s: %w", roleName, orgID.String(), err)
	}

	// Clone the permissions by creating role_permission entries
	if len(template.Permissions) > 0 {
		rolePermissions := make([]*schema.RolePermission, 0, len(template.Permissions))
		for _, perm := range template.Permissions {
			rolePermissions = append(rolePermissions, &schema.RolePermission{
				ID:           xid.New(),
				RoleID:       newRole.ID,
				PermissionID: perm.ID,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}

		_, err = r.db.NewInsert().
			Model(&rolePermissions).
			Exec(ctx)
		if err != nil {
			// Rollback: delete the created role
			_, _ = r.db.NewDelete().Model(newRole).WherePK().Exec(ctx)
			return nil, fmt.Errorf("failed to clone permissions: %w", err)
		}
	}

	return newRole, nil
}

// FindDuplicateRoles identifies roles that would violate the new uniqueness constraints
func (r *RoleRepository) FindDuplicateRoles(ctx context.Context) ([]schema.Role, error) {
	var duplicates []schema.Role
	
	// Find app-level duplicates
	err := r.db.NewSelect().
		Model(&duplicates).
		Where("organization_id IS NULL").
		Where(`(app_id, environment_id, name, is_template) IN (
			SELECT app_id, environment_id, name, is_template
			FROM roles
			WHERE organization_id IS NULL
			GROUP BY app_id, environment_id, name, is_template
			HAVING COUNT(*) > 1
		)`).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Find org-level duplicates
	var orgDuplicates []schema.Role
	err = r.db.NewSelect().
		Model(&orgDuplicates).
		Where("organization_id IS NOT NULL").
		Where(`(app_id, environment_id, organization_id, name, is_template) IN (
			SELECT app_id, environment_id, organization_id, name, is_template
			FROM roles
			WHERE organization_id IS NOT NULL
			GROUP BY app_id, environment_id, organization_id, name, is_template
			HAVING COUNT(*) > 1
		)`).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	duplicates = append(duplicates, orgDuplicates...)
	return duplicates, nil
}

// toTitleCase converts a snake_case string to Title Case
func toTitleCase(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
