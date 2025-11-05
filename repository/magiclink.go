package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// MagicLinkRepository provides persistence for Magic Links
type MagicLinkRepository struct {
	db *bun.DB
}

func NewMagicLinkRepository(db *bun.DB) *MagicLinkRepository { return &MagicLinkRepository{db: db} }

// Create stores a new magic link record
func (r *MagicLinkRepository) Create(ctx context.Context, email, token string, expiresAt time.Time) error {
	rec := &schema.MagicLink{ID: xid.New(), Email: email, Token: token, ExpiresAt: expiresAt}
	rec.AuditableModel.CreatedBy = rec.ID
	rec.AuditableModel.UpdatedBy = rec.ID
	_, err := r.db.NewInsert().Model(rec).Exec(ctx)
	return err
}

// FindByToken returns an active magic link by token
func (r *MagicLinkRepository) FindByToken(ctx context.Context, token string, now time.Time) (*schema.MagicLink, error) {
	rec := new(schema.MagicLink)
	err := r.db.NewSelect().Model(rec).
		Where("token = ?", token).
		Where("expires_at > ?", now).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// Consume marks link as expired by setting expiresAt to now
func (r *MagicLinkRepository) Consume(ctx context.Context, rec *schema.MagicLink, now time.Time) error {
	rec.ExpiresAt = now
	_, err := r.db.NewUpdate().Model(rec).WherePK().Exec(ctx)
	return err
}
