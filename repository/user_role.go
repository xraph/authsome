package repository

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// UserRoleRepository manages user-role assignments.
type UserRoleRepository struct{ db *bun.DB }

func NewUserRoleRepository(db *bun.DB) *UserRoleRepository { return &UserRoleRepository{db: db} }

// Assign links a user to a role within an organization.
func (r *UserRoleRepository) Assign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	ur := &schema.UserRole{UserID: userID, RoleID: roleID, AppID: orgID}
	// Populate required auditable fields
	ur.ID = xid.New()
	ur.CreatedBy = xid.New()
	ur.UpdatedBy = ur.CreatedBy
	_, err := r.db.NewInsert().Model(ur).Exec(ctx)

	return err
}

// ====== Assignment Methods ======

// AssignBatch assigns multiple roles to a single user in an organization.
func (r *UserRoleRepository) AssignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	userRoles := make([]*schema.UserRole, len(roleIDs))
	createdBy := xid.New()

	for i, roleID := range roleIDs {
		userRoles[i] = &schema.UserRole{
			ID:     xid.New(),
			UserID: userID,
			RoleID: roleID,
			AppID:  orgID,
		}
		userRoles[i].CreatedBy = createdBy
		userRoles[i].UpdatedBy = createdBy
	}

	_, err := r.db.NewInsert().Model(&userRoles).Exec(ctx)

	return err
}

// AssignBulk assigns a single role to multiple users in an organization.
func (r *UserRoleRepository) AssignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	errors := make(map[xid.ID]error)
	createdBy := xid.New()

	for _, userID := range userIDs {
		ur := &schema.UserRole{
			ID:     xid.New(),
			UserID: userID,
			RoleID: roleID,
			AppID:  orgID,
		}
		ur.CreatedBy = createdBy
		ur.UpdatedBy = createdBy

		_, err := r.db.NewInsert().Model(ur).Exec(ctx)
		if err != nil {
			errors[userID] = err
		}
	}

	if len(errors) > 0 {
		return errors, nil
	}

	return nil, nil
}

// AssignAppLevel assigns a role at app-level (not org-scoped).
func (r *UserRoleRepository) AssignAppLevel(ctx context.Context, userID, roleID, appID xid.ID) error {
	ur := &schema.UserRole{
		ID:     xid.New(),
		UserID: userID,
		RoleID: roleID,
		AppID:  appID,
	}
	// Populate required auditable fields
	ur.CreatedBy = xid.New()
	ur.UpdatedBy = ur.CreatedBy

	_, err := r.db.NewInsert().Model(ur).Exec(ctx)

	return err
}

// ListRolesForUser returns roles assigned to a user, optionally filtered by org.
func (r *UserRoleRepository) ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error) {
	var roles []schema.Role

	q := r.db.NewSelect().Model(&roles).
		Join("JOIN user_roles AS ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID)
	if orgID != nil && !orgID.IsNil() {
		q = q.Where("ur.app_id = ?", *orgID)
	}

	err := q.Scan(ctx)

	return roles, err
}

// ====== Listing Methods ======

// ListRolesForUserInOrg gets roles for a specific user in an organization with environment filter.
func (r *UserRoleRepository) ListRolesForUserInOrg(ctx context.Context, userID, orgID, envID xid.ID) ([]schema.Role, error) {
	var roles []schema.Role

	err := r.db.NewSelect().Model(&roles).
		Join("JOIN user_roles AS ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("ur.app_id = ?", orgID).
		Where("r.environment_id = ?", envID).
		Scan(ctx)

	return roles, err
}

// ListRolesForUserInApp gets roles for a specific user across all orgs in an app with environment filter.
func (r *UserRoleRepository) ListRolesForUserInApp(ctx context.Context, userID, appID, envID xid.ID) ([]schema.Role, error) {
	var roles []schema.Role

	err := r.db.NewSelect().Model(&roles).
		Join("JOIN user_roles AS ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("r.app_id = ?", appID).
		Where("r.environment_id = ?", envID).
		Scan(ctx)

	return roles, err
}

// ListAllUserRolesInOrg lists all user-role assignments in an organization (admin view).
func (r *UserRoleRepository) ListAllUserRolesInOrg(ctx context.Context, orgID, envID xid.ID) ([]schema.UserRole, error) {
	var userRoles []schema.UserRole

	err := r.db.NewSelect().Model(&userRoles).
		Relation("User").
		Relation("Role").
		Where("ur.app_id = ?", orgID).
		Where("EXISTS (SELECT 1 FROM roles WHERE roles.id = ur.role_id AND roles.environment_id = ?)", envID).
		Scan(ctx)

	return userRoles, err
}

// ListAllUserRolesInApp lists all user-role assignments in an app across all orgs (admin view).
func (r *UserRoleRepository) ListAllUserRolesInApp(ctx context.Context, appID, envID xid.ID) ([]schema.UserRole, error) {
	var userRoles []schema.UserRole

	err := r.db.NewSelect().Model(&userRoles).
		Relation("User").
		Relation("Role").
		Join("JOIN roles AS r ON r.id = ur.role_id").
		Where("r.app_id = ?", appID).
		Where("r.environment_id = ?", envID).
		Scan(ctx)

	return userRoles, err
}

// Unassign removes a user-role assignment within an organization.
func (r *UserRoleRepository) Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.UserRole)(nil)).
		Where("user_id = ?", userID).
		Where("role_id = ?", roleID).
		Where("app_id = ?", orgID).
		Exec(ctx)

	return err
}

// ====== Unassignment Methods ======

// UnassignBatch removes multiple roles from a single user in an organization.
func (r *UserRoleRepository) UnassignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	_, err := r.db.NewDelete().Model((*schema.UserRole)(nil)).
		Where("user_id = ?", userID).
		Where("role_id IN (?)", bun.In(roleIDs)).
		Where("app_id = ?", orgID).
		Exec(ctx)

	return err
}

// UnassignBulk removes a single role from multiple users in an organization.
func (r *UserRoleRepository) UnassignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	errors := make(map[xid.ID]error)

	for _, userID := range userIDs {
		_, err := r.db.NewDelete().Model((*schema.UserRole)(nil)).
			Where("user_id = ?", userID).
			Where("role_id = ?", roleID).
			Where("app_id = ?", orgID).
			Exec(ctx)
		if err != nil {
			errors[userID] = err
		}
	}

	if len(errors) > 0 {
		return errors, nil
	}

	return nil, nil
}

// ClearUserRolesInOrg removes all roles from a user in an organization.
func (r *UserRoleRepository) ClearUserRolesInOrg(ctx context.Context, userID, orgID xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.UserRole)(nil)).
		Where("user_id = ?", userID).
		Where("app_id = ?", orgID).
		Exec(ctx)

	return err
}

// ClearUserRolesInApp removes all roles from a user in an app.
func (r *UserRoleRepository) ClearUserRolesInApp(ctx context.Context, userID, appID xid.ID) error {
	// For app-level clearing, we need to delete all roles where the role's app_id matches
	// This requires joining with the roles table
	_, err := r.db.NewDelete().Model((*schema.UserRole)(nil)).
		Where("user_id = ?", userID).
		Where("EXISTS (SELECT 1 FROM roles WHERE roles.id = user_roles.role_id AND roles.app_id = ?)", appID).
		Exec(ctx)

	return err
}

// ====== Transfer/Move Methods ======

// TransferRoles moves roles from one org to another (delete + insert in transaction).
func (r *UserRoleRepository) TransferRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete from source org
		_, err := tx.NewDelete().Model((*schema.UserRole)(nil)).
			Where("user_id = ?", userID).
			Where("role_id IN (?)", bun.In(roleIDs)).
			Where("app_id = ?", sourceOrgID).
			Exec(ctx)
		if err != nil {
			return err
		}

		// Insert to target org
		userRoles := make([]*schema.UserRole, len(roleIDs))
		createdBy := xid.New()

		for i, roleID := range roleIDs {
			userRoles[i] = &schema.UserRole{
				ID:     xid.New(),
				UserID: userID,
				RoleID: roleID,
				AppID:  targetOrgID,
			}
			userRoles[i].CreatedBy = createdBy
			userRoles[i].UpdatedBy = createdBy
		}

		_, err = tx.NewInsert().Model(&userRoles).Exec(ctx)

		return err
	})
}

// CopyRoles duplicates roles from one org to another (insert only).
func (r *UserRoleRepository) CopyRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	// Verify roles exist in source org
	var existingRoles []schema.UserRole

	err := r.db.NewSelect().Model(&existingRoles).
		Where("user_id = ?", userID).
		Where("role_id IN (?)", bun.In(roleIDs)).
		Where("app_id = ?", sourceOrgID).
		Scan(ctx)
	if err != nil {
		return err
	}

	// Build user roles for target org
	userRoles := make([]*schema.UserRole, len(existingRoles))
	createdBy := xid.New()

	for i, existing := range existingRoles {
		userRoles[i] = &schema.UserRole{
			ID:     xid.New(),
			UserID: userID,
			RoleID: existing.RoleID,
			AppID:  targetOrgID,
		}
		userRoles[i].CreatedBy = createdBy
		userRoles[i].UpdatedBy = createdBy
	}

	if len(userRoles) == 0 {
		return nil
	}

	_, err = r.db.NewInsert().Model(&userRoles).Exec(ctx)

	return err
}

// ReplaceUserRoles atomically replaces all user roles in an org with a new set.
func (r *UserRoleRepository) ReplaceUserRoles(ctx context.Context, userID, orgID xid.ID, newRoleIDs []xid.ID) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete all existing roles
		_, err := tx.NewDelete().Model((*schema.UserRole)(nil)).
			Where("user_id = ?", userID).
			Where("app_id = ?", orgID).
			Exec(ctx)
		if err != nil {
			return err
		}

		// Insert new roles if any
		if len(newRoleIDs) == 0 {
			return nil
		}

		userRoles := make([]*schema.UserRole, len(newRoleIDs))
		createdBy := xid.New()

		for i, roleID := range newRoleIDs {
			userRoles[i] = &schema.UserRole{
				ID:     xid.New(),
				UserID: userID,
				RoleID: roleID,
				AppID:  orgID,
			}
			userRoles[i].CreatedBy = createdBy
			userRoles[i].UpdatedBy = createdBy
		}

		_, err = tx.NewInsert().Model(&userRoles).Exec(ctx)

		return err
	})
}
