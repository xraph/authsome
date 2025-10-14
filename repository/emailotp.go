package repository

import (
    "context"
    "time"

    "github.com/rs/xid"
    "github.com/uptrace/bun"
    "github.com/xraph/authsome/schema"
)

// EmailOTPRepository provides persistence for Email OTP entities
type EmailOTPRepository struct {
    db *bun.DB
}

func NewEmailOTPRepository(db *bun.DB) *EmailOTPRepository { return &EmailOTPRepository{db: db} }

// Create stores a new email OTP record
func (r *EmailOTPRepository) Create(ctx context.Context, email, otp string, expiresAt time.Time) error {
    rec := &schema.EmailOTP{ID: xid.New(), Email: email, OTP: otp, ExpiresAt: expiresAt, Attempts: 0}
    rec.AuditableModel.CreatedBy = rec.ID
    rec.AuditableModel.UpdatedBy = rec.ID
    _, err := r.db.NewInsert().Model(rec).Exec(ctx)
    return err
}

// FindByEmail returns the latest active OTP record for an email
func (r *EmailOTPRepository) FindByEmail(ctx context.Context, email string, now time.Time) (*schema.EmailOTP, error) {
    rec := new(schema.EmailOTP)
    err := r.db.NewSelect().Model(rec).
        Where("email = ?", email).
        Where("expires_at > ?", now).
        OrderExpr("expires_at DESC").
        Scan(ctx)
    if err != nil { return nil, err }
    return rec, nil
}

// IncrementAttempts increments attempts count
func (r *EmailOTPRepository) IncrementAttempts(ctx context.Context, rec *schema.EmailOTP) error {
    rec.Attempts++
    _, err := r.db.NewUpdate().Model(rec).WherePK().Exec(ctx)
    return err
}

// Consume marks OTP as consumed by expiring now
func (r *EmailOTPRepository) Consume(ctx context.Context, rec *schema.EmailOTP, now time.Time) error {
    rec.ExpiresAt = now
    _, err := r.db.NewUpdate().Model(rec).WherePK().Exec(ctx)
    return err
}