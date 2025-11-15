package repository

import (
	"context"

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
