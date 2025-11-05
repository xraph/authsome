package repository

import (
	"context"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	rbac "github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

// PolicyRepository implements rbac.PolicyRepository using Bun
type PolicyRepository struct{ db *bun.DB }

func NewPolicyRepository(db *bun.DB) *PolicyRepository { return &PolicyRepository{db: db} }

func (r *PolicyRepository) ListAll(ctx context.Context) ([]string, error) {
	var rows []schema.Policy
	err := r.db.NewSelect().Model(&rows).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Expression)
	}
	return out, nil
}

func (r *PolicyRepository) Create(ctx context.Context, expression string) error {
	// Populate required auditable fields
	p := &schema.Policy{Expression: expression}
	p.ID = xid.New()
	p.AuditableModel.CreatedBy = xid.New()
	p.AuditableModel.UpdatedBy = p.AuditableModel.CreatedBy
	_, err := r.db.NewInsert().Model(p).Exec(ctx)
	return err
}

// Update modifies an existing policy's expression by ID
func (r *PolicyRepository) Update(ctx context.Context, id xid.ID, expression string) error {
	p := &schema.Policy{ID: id, Expression: expression}
	_, err := r.db.NewUpdate().Model(p).Column("expression").WherePK().Exec(ctx)
	return err
}

// List returns full policy rows for management
func (r *PolicyRepository) List(ctx context.Context) ([]schema.Policy, error) {
	var rows []schema.Policy
	err := r.db.NewSelect().Model(&rows).Scan(ctx)
	return rows, err
}

// Delete removes a policy by ID
func (r *PolicyRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Policy)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

var _ rbac.PolicyRepository = (*PolicyRepository)(nil)
