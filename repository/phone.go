package repository

import (
    "context"
    "time"

    "github.com/rs/xid"
    "github.com/uptrace/bun"
    "github.com/xraph/authsome/schema"
)

// PhoneRepository provides persistence for phone verification codes
type PhoneRepository struct {
    db *bun.DB
}

func NewPhoneRepository(db *bun.DB) *PhoneRepository { return &PhoneRepository{db: db} }

// Create stores a new phone verification record
func (r *PhoneRepository) Create(ctx context.Context, phone, code string, expiresAt time.Time) error {
    rec := &schema.PhoneVerification{ID: xid.New(), Phone: phone, Code: code, ExpiresAt: expiresAt, Attempts: 0}
    rec.AuditableModel.CreatedBy = rec.ID
    rec.AuditableModel.UpdatedBy = rec.ID
    _, err := r.db.NewInsert().Model(rec).Exec(ctx)
    return err
}

// FindByPhone returns the latest active verification for a phone
func (r *PhoneRepository) FindByPhone(ctx context.Context, phone string, now time.Time) (*schema.PhoneVerification, error) {
    rec := new(schema.PhoneVerification)
    err := r.db.NewSelect().Model(rec).
        Where("phone = ?", phone).
        Where("expires_at > ?", now).
        OrderExpr("expires_at DESC").
        Scan(ctx)
    if err != nil { return nil, err }
    return rec, nil
}

// IncrementAttempts increments attempts count
func (r *PhoneRepository) IncrementAttempts(ctx context.Context, rec *schema.PhoneVerification) error {
    rec.Attempts++
    _, err := r.db.NewUpdate().Model(rec).WherePK().Exec(ctx)
    return err
}

// Consume marks code as consumed by expiring now
func (r *PhoneRepository) Consume(ctx context.Context, rec *schema.PhoneVerification, now time.Time) error {
    rec.ExpiresAt = now
    _, err := r.db.NewUpdate().Model(rec).WherePK().Exec(ctx)
    return err
}