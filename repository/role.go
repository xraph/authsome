package repository

import (
	"context"
	"database/sql"
	"fmt"
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

// FindByNameAndApp finds a role by name within an app
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

// GetRoleTemplates gets all role templates for an app (templates have organization_id = NULL and is_template = true)
func (r *RoleRepository) GetRoleTemplates(ctx context.Context, appID xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role
	err := r.db.NewSelect().
		Model(&roles).
		Where("app_id = ?", appID).
		Where("organization_id IS NULL").
		Where("is_template = ?", true).
		Order("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetOwnerRole gets the role marked as the owner role for an app
func (r *RoleRepository) GetOwnerRole(ctx context.Context, appID xid.ID) (*schema.Role, error) {
	var role schema.Role
	err := r.db.NewSelect().
		Model(&role).
		Where("app_id = ?", appID).
		Where("organization_id IS NULL").
		Where("is_owner_role = ?", true).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("owner role not found for app %s", appID.String())
		}
		return nil, err
	}
	return &role, nil
}

// GetOrgRoles gets all roles specific to an organization
func (r *RoleRepository) GetOrgRoles(ctx context.Context, orgID xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role
	err := r.db.NewSelect().
		Model(&roles).
		Where("organization_id = ?", orgID).
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

	// Create the new role based on the template
	now := time.Now()
	newRole := &schema.Role{
		ID:             xid.New(),
		AppID:          template.AppID,
		OrganizationID: &orgID,
		Name:           template.Name,
		Description:    template.Description,
		IsTemplate:     false,
		IsOwnerRole:    false,
		TemplateID:     &templateID,
	}

	// Apply custom name if provided
	if customName != nil && *customName != "" {
		newRole.Name = *customName
	}

	// Set auditable fields
	newRole.CreatedAt = now
	newRole.UpdatedAt = now
	newRole.CreatedBy = orgID // Use org ID as creator for now
	newRole.UpdatedBy = orgID
	newRole.Version = 1

	// Insert the new role
	_, err = r.db.NewInsert().Model(newRole).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloned role: %w", err)
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
