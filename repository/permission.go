package repository

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// PermissionRepository provides basic CRUD for permissions
type PermissionRepository struct{ db *bun.DB }

func NewPermissionRepository(db *bun.DB) *PermissionRepository { return &PermissionRepository{db: db} }

func (r *PermissionRepository) Create(ctx context.Context, perm *schema.Permission) error {
	_, err := r.db.NewInsert().Model(perm).Exec(ctx)
	return err
}

func (r *PermissionRepository) ListByOrg(ctx context.Context, orgID *string) ([]schema.Permission, error) {
	var rows []schema.Permission
	q := r.db.NewSelect().Model(&rows)
	if orgID != nil {
		q = q.Where("app_id = ?", *orgID)
	}
	err := q.Scan(ctx)
	return rows, err
}
