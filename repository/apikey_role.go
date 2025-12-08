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

// APIKeyRoleRepository handles API key to role assignments
type APIKeyRoleRepository struct {
	db *bun.DB
}

// NewAPIKeyRoleRepository creates a new API key role repository
func NewAPIKeyRoleRepository(db *bun.DB) *APIKeyRoleRepository {
	return &APIKeyRoleRepository{db: db}
}

// AssignRole assigns a role to an API key
func (r *APIKeyRoleRepository) AssignRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID, createdBy *xid.ID) error {
	assignment := &schema.APIKeyRole{
		ID:             xid.New(),
		APIKeyID:       apiKeyID,
		RoleID:         roleID,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
	}

	_, err := r.db.NewInsert().
		Model(assignment).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to assign role to API key: %w", err)
	}

	return nil
}

// UnassignRole removes a role from an API key (soft delete)
func (r *APIKeyRoleRepository) UnassignRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID) error {
	query := r.db.NewUpdate().
		Model((*schema.APIKeyRole)(nil)).
		Set("deleted_at = ?", time.Now()).
		Where("api_key_id = ?", apiKeyID).
		Where("role_id = ?", roleID).
		Where("deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("organization_id = ?", *orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unassign role from API key: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("role assignment not found")
	}

	return nil
}

// GetRoles retrieves all roles assigned to an API key
func (r *APIKeyRoleRepository) GetRoles(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role

	query := r.db.NewSelect().
		Model(&roles).
		Join("INNER JOIN apikey_roles akr ON akr.role_id = role.id").
		Where("akr.api_key_id = ?", apiKeyID).
		Where("akr.deleted_at IS NULL").
		Where("role.deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("akr.organization_id = ?", *orgID)
	} else {
		query = query.Where("akr.organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key roles: %w", err)
	}

	return roles, nil
}

// GetPermissions retrieves all permissions for an API key through its roles
func (r *APIKeyRoleRepository) GetPermissions(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) ([]*schema.Permission, error) {
	var permissions []*schema.Permission

	query := r.db.NewSelect().
		Model(&permissions).
		Distinct().
		Join("INNER JOIN role_permissions rp ON rp.permission_id = permission.id").
		Join("INNER JOIN apikey_roles akr ON akr.role_id = rp.role_id").
		Where("akr.api_key_id = ?", apiKeyID).
		Where("akr.deleted_at IS NULL").
		Where("rp.deleted_at IS NULL").
		Where("permission.deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("akr.organization_id = ?", *orgID)
	} else {
		query = query.Where("akr.organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key permissions: %w", err)
	}

	return permissions, nil
}

// HasRole checks if an API key has a specific role
func (r *APIKeyRoleRepository) HasRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID) (bool, error) {
	query := r.db.NewSelect().
		Model((*schema.APIKeyRole)(nil)).
		Where("api_key_id = ?", apiKeyID).
		Where("role_id = ?", roleID).
		Where("deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("organization_id = ?", *orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	exists, err := query.Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check API key role: %w", err)
	}

	return exists, nil
}

// GetAPIKeysWithRole retrieves all API keys that have a specific role
func (r *APIKeyRoleRepository) GetAPIKeysWithRole(ctx context.Context, roleID xid.ID, orgID *xid.ID) ([]*schema.APIKey, error) {
	var apiKeys []*schema.APIKey

	query := r.db.NewSelect().
		Model(&apiKeys).
		Join("INNER JOIN apikey_roles akr ON akr.api_key_id = api_keys.id").
		Where("akr.role_id = ?", roleID).
		Where("akr.deleted_at IS NULL").
		Where("api_keys.deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("akr.organization_id = ?", *orgID)
	} else {
		query = query.Where("akr.organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys with role: %w", err)
	}

	return apiKeys, nil
}

// GetCreatorPermissions retrieves the permissions of the user who created the API key
func (r *APIKeyRoleRepository) GetCreatorPermissions(ctx context.Context, creatorID xid.ID, orgID *xid.ID) ([]*schema.Permission, error) {
	var permissions []*schema.Permission

	query := r.db.NewSelect().
		Model(&permissions).
		Distinct().
		Join("INNER JOIN role_permissions rp ON rp.permission_id = permission.id").
		Join("INNER JOIN user_roles ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", creatorID).
		Where("ur.deleted_at IS NULL").
		Where("rp.deleted_at IS NULL").
		Where("permission.deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("ur.organization_id = ?", *orgID)
	} else {
		query = query.Where("ur.organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator permissions: %w", err)
	}

	return permissions, nil
}

// GetCreatorRoles retrieves the roles of the user who created the API key
func (r *APIKeyRoleRepository) GetCreatorRoles(ctx context.Context, creatorID xid.ID, orgID *xid.ID) ([]*schema.Role, error) {
	var roles []*schema.Role

	query := r.db.NewSelect().
		Model(&roles).
		Join("INNER JOIN user_roles ur ON ur.role_id = role.id").
		Where("ur.user_id = ?", creatorID).
		Where("ur.deleted_at IS NULL").
		Where("role.deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("ur.organization_id = ?", *orgID)
	} else {
		query = query.Where("ur.organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator roles: %w", err)
	}

	return roles, nil
}

// BulkAssignRoles assigns multiple roles to an API key in a single transaction
func (r *APIKeyRoleRepository) BulkAssignRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID, createdBy *xid.ID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		for _, roleID := range roleIDs {
			assignment := &schema.APIKeyRole{
				ID:             xid.New(),
				APIKeyID:       apiKeyID,
				RoleID:         roleID,
				OrganizationID: orgID,
				CreatedBy:      createdBy,
				CreatedAt:      time.Now(),
			}

			_, err := tx.NewInsert().
				Model(assignment).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to assign role %s: %w", roleID, err)
			}
		}
		return nil
	})
}

// BulkUnassignRoles removes multiple roles from an API key in a single transaction
func (r *APIKeyRoleRepository) BulkUnassignRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	query := r.db.NewUpdate().
		Model((*schema.APIKeyRole)(nil)).
		Set("deleted_at = ?", time.Now()).
		Where("api_key_id = ?", apiKeyID).
		Where("role_id IN (?)", bun.In(roleIDs)).
		Where("deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("organization_id = ?", *orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to bulk unassign roles: %w", err)
	}

	return nil
}

// ReplaceRoles replaces all roles for an API key with a new set (in a transaction)
func (r *APIKeyRoleRepository) ReplaceRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID, createdBy *xid.ID) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// First, soft delete all existing assignments
		query := tx.NewUpdate().
			Model((*schema.APIKeyRole)(nil)).
			Set("deleted_at = ?", time.Now()).
			Where("api_key_id = ?", apiKeyID).
			Where("deleted_at IS NULL")

		if orgID != nil {
			query = query.Where("organization_id = ?", *orgID)
		} else {
			query = query.Where("organization_id IS NULL")
		}

		_, err := query.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to remove existing roles: %w", err)
		}

		// Then, assign new roles
		if len(roleIDs) > 0 {
			for _, roleID := range roleIDs {
				assignment := &schema.APIKeyRole{
					ID:             xid.New(),
					APIKeyID:       apiKeyID,
					RoleID:         roleID,
					OrganizationID: orgID,
					CreatedBy:      createdBy,
					CreatedAt:      time.Now(),
				}

				_, err := tx.NewInsert().
					Model(assignment).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to assign role %s: %w", roleID, err)
				}
			}
		}

		return nil
	})
}
