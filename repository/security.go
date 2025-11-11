package repository

import (
	"context"

	"github.com/uptrace/bun"
	core "github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/schema"
)

// SecurityRepository implements core security repository using Bun
type SecurityRepository struct{ db *bun.DB }

func NewSecurityRepository(db *bun.DB) *SecurityRepository { return &SecurityRepository{db: db} }

func (r *SecurityRepository) toSchema(e *core.SecurityEvent) *schema.SecurityEvent {
	return &schema.SecurityEvent{
		ID:        e.ID,
		UserID:    e.UserID,
		Type:      e.Type,
		IPAddress: e.IPAddress,
		UserAgent: e.UserAgent,
		Geo:       e.Geo,
	}
}

func (r *SecurityRepository) fromSchema(se *schema.SecurityEvent) *core.SecurityEvent {
	if se == nil {
		return nil
	}
	return &core.SecurityEvent{
		ID:        se.ID,
		UserID:    se.UserID,
		Type:      se.Type,
		IPAddress: se.IPAddress,
		UserAgent: se.UserAgent,
		Geo:       se.Geo,
		CreatedAt: se.CreatedAt,
		UpdatedAt: se.UpdatedAt,
	}
}

func (r *SecurityRepository) Create(ctx context.Context, e *core.SecurityEvent) error {
	se := r.toSchema(e)
	_, err := r.db.NewInsert().Model(se).Exec(ctx)
	return err
}
