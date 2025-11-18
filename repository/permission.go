package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// PermissionRepository provides basic CRUD for permissions
type PermissionRepository struct{ db *bun.DB }

func NewPermissionRepository(db *bun.DB) *PermissionRepository { return &PermissionRepository{db: db} }

func (r *PermissionRepository) Create(ctx context.Context, perm *schema.Permission) error {
	if perm.ID.IsNil() {
		perm.ID = xid.New()
	}
	if perm.AuditableModel.CreatedBy.IsNil() {
		perm.AuditableModel.CreatedBy = xid.New()
	}
	if perm.AuditableModel.UpdatedBy.IsNil() {
		perm.AuditableModel.UpdatedBy = perm.AuditableModel.CreatedBy
	}
	_, err := r.db.NewInsert().Model(perm).Exec(ctx)
	return err
}

func (r *PermissionRepository) Update(ctx context.Context, perm *schema.Permission) error {
	now := time.Now()
	perm.UpdatedAt = now
	_, err := r.db.NewUpdate().
		Model(perm).
		WherePK().
		Exec(ctx)
	return err
}

func (r *PermissionRepository) Delete(ctx context.Context, permissionID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Permission)(nil)).
		Where("id = ?", permissionID).
		Exec(ctx)
	return err
}

func (r *PermissionRepository) FindByID(ctx context.Context, permissionID xid.ID) (*schema.Permission, error) {
	var perm schema.Permission
	err := r.db.NewSelect().
		Model(&perm).
		Where("id = ?", permissionID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *PermissionRepository) FindByName(ctx context.Context, name string, appID xid.ID, orgID *xid.ID) (*schema.Permission, error) {
	var perm schema.Permission
	q := r.db.NewSelect().
		Model(&perm).
		Where("name = ?", name).
		Where("app_id = ?", appID)
	
	if orgID != nil {
		q = q.Where("organization_id = ?", *orgID)
	} else {
		q = q.Where("organization_id IS NULL")
	}
	
	err := q.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *PermissionRepository) ListByApp(ctx context.Context, appID xid.ID) ([]*schema.Permission, error) {
	var perms []*schema.Permission
	err := r.db.NewSelect().
		Model(&perms).
		Where("app_id = ?", appID).
		Where("organization_id IS NULL"). // Only app-level permissions
		Order("category ASC, name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *PermissionRepository) ListByOrg(ctx context.Context, orgID xid.ID) ([]*schema.Permission, error) {
	var perms []*schema.Permission
	err := r.db.NewSelect().
		Model(&perms).
		Where("organization_id = ?", orgID).
		Order("category ASC, name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *PermissionRepository) ListByCategory(ctx context.Context, category string, appID xid.ID) ([]*schema.Permission, error) {
	var perms []*schema.Permission
	err := r.db.NewSelect().
		Model(&perms).
		Where("app_id = ?", appID).
		Where("category = ?", category).
		Where("organization_id IS NULL"). // Only app-level permissions
		Order("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *PermissionRepository) CreateCustomPermission(ctx context.Context, name, description, category string, orgID xid.ID) (*schema.Permission, error) {
	now := time.Now()
	perm := &schema.Permission{
		ID:             xid.New(),
		OrganizationID: &orgID,
		Name:           name,
		Description:    description,
		IsCustom:       true,
		Category:       category,
	}
	
	perm.CreatedAt = now
	perm.UpdatedAt = now
	perm.CreatedBy = orgID // Use org ID as creator
	perm.UpdatedBy = orgID
	perm.Version = 1
	
	_, err := r.db.NewInsert().Model(perm).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return perm, nil
}
