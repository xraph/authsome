package repository

import (
    "context"
    "github.com/rs/xid"
    "github.com/uptrace/bun"
    "github.com/xraph/authsome/schema"
)

// UserRoleRepository manages user-role assignments
type UserRoleRepository struct{ db *bun.DB }

func NewUserRoleRepository(db *bun.DB) *UserRoleRepository { return &UserRoleRepository{db: db} }

// Assign links a user to a role within an organization
func (r *UserRoleRepository) Assign(ctx context.Context, userID, roleID, orgID xid.ID) error {
    ur := &schema.UserRole{UserID: userID, RoleID: roleID, OrganizationID: orgID}
    // Populate required auditable fields
    ur.ID = xid.New()
    ur.AuditableModel.CreatedBy = xid.New()
    ur.AuditableModel.UpdatedBy = ur.AuditableModel.CreatedBy
    _, err := r.db.NewInsert().Model(ur).Exec(ctx)
    return err
}

// ListRolesForUser returns roles assigned to a user, optionally filtered by org
func (r *UserRoleRepository) ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error) {
    var roles []schema.Role
    q := r.db.NewSelect().Model(&roles).
        Join("JOIN user_roles AS ur ON ur.role_id = r.id").
        Where("ur.user_id = ?", userID)
    if orgID != nil && !orgID.IsNil() {
        q = q.Where("ur.organization_id = ?", *orgID)
    }
    err := q.Scan(ctx)
    return roles, err
}

// Unassign removes a user-role assignment within an organization
func (r *UserRoleRepository) Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.UserRole)(nil)).
        Where("user_id = ?", userID).
        Where("role_id = ?", roleID).
        Where("organization_id = ?", orgID).
        Exec(ctx)
    return err
}