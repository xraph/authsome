package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// RolePermissionRepository handles role-permission relationships
type RolePermissionRepository struct{ db *bun.DB }

func NewRolePermissionRepository(db *bun.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

func (r *RolePermissionRepository) AssignPermission(ctx context.Context, roleID, permissionID xid.ID) error {
	now := time.Now()
	rp := &schema.RolePermission{
		ID:           xid.New(),
		RoleID:       roleID,
		PermissionID: permissionID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	
	_, err := r.db.NewInsert().
		Model(rp).
		On("CONFLICT (role_id, permission_id) DO NOTHING").
		Exec(ctx)
	return err
}

func (r *RolePermissionRepository) UnassignPermission(ctx context.Context, roleID, permissionID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.RolePermission)(nil)).
		Where("role_id = ?", roleID).
		Where("permission_id = ?", permissionID).
		Exec(ctx)
	return err
}

func (r *RolePermissionRepository) GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error) {
	var permissions []*schema.Permission
	err := r.db.NewSelect().
		Model(&permissions).
		Join("INNER JOIN role_permissions AS rp ON rp.permission_id = permission.id").
		Where("rp.role_id = ?", roleID).
		Order("permission.category ASC, permission.name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *RolePermissionRepository) GetPermissionRoles(ctx context.Context, permissionID xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role
	err := r.db.NewSelect().
		Model(&roles).
		Join("INNER JOIN role_permissions AS rp ON rp.role_id = r.id").
		Where("rp.permission_id = ?", permissionID).
		Order("r.name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RolePermissionRepository) ReplaceRolePermissions(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	// Start a transaction
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete all existing role-permission associations
		_, err := tx.NewDelete().
			Model((*schema.RolePermission)(nil)).
			Where("role_id = ?", roleID).
			Exec(ctx)
		if err != nil {
			return err
		}
		
		// Insert new associations
		if len(permissionIDs) > 0 {
			now := time.Now()
			rolePermissions := make([]*schema.RolePermission, 0, len(permissionIDs))
			for _, permID := range permissionIDs {
				rolePermissions = append(rolePermissions, &schema.RolePermission{
					ID:           xid.New(),
					RoleID:       roleID,
					PermissionID: permID,
					CreatedAt:    now,
					UpdatedAt:    now,
				})
			}
			
			_, err = tx.NewInsert().
				Model(&rolePermissions).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
		
		return nil
	})
}

