package repository

import (
    "context"
    "database/sql"
    "time"

    "github.com/rs/xid"
    "github.com/uptrace/bun"
    "github.com/xraph/authsome/schema"
)

// TwoFARepository provides persistence for 2FA entities
type TwoFARepository struct {
    db *bun.DB
}

func NewTwoFARepository(db *bun.DB) *TwoFARepository { return &TwoFARepository{db: db} }

// UpsertSecret sets or updates a user's 2FA secret
func (r *TwoFARepository) UpsertSecret(ctx context.Context, userID xid.ID, method, secret string, enabled bool) error {
    // Try fetch existing
    existing := new(schema.TwoFASecret)
    err := r.db.NewSelect().Model(existing).Where("user_id = ?", userID).Scan(ctx)
    if err != nil && err != sql.ErrNoRows {
        return err
    }
    if err == sql.ErrNoRows {
        m := &schema.TwoFASecret{ID: xid.New(), UserID: userID, Method: method, Secret: secret, Enabled: enabled}
        m.AuditableModel.CreatedBy = userID
        m.AuditableModel.UpdatedBy = userID
        _, err = r.db.NewInsert().Model(m).Exec(ctx)
        return err
    }
    existing.Method = method
    existing.Secret = secret
    existing.Enabled = enabled
    _, err = r.db.NewUpdate().Model(existing).WherePK().Exec(ctx)
    return err
}

// GetSecret returns a user's 2FA secret
func (r *TwoFARepository) GetSecret(ctx context.Context, userID xid.ID) (*schema.TwoFASecret, error) {
    m := new(schema.TwoFASecret)
    err := r.db.NewSelect().Model(m).Where("user_id = ?", userID).Scan(ctx)
    if err == sql.ErrNoRows { return nil, nil }
    if err != nil { return nil, err }
    return m, nil
}

// DisableSecret disables 2FA for a user
func (r *TwoFARepository) DisableSecret(ctx context.Context, userID xid.ID) error {
    _, err := r.db.NewUpdate().Model((*schema.TwoFASecret)(nil)).
        Set("enabled = ?", false).
        Where("user_id = ?", userID).
        Exec(ctx)
    return err
}

// CreateBackupCodes stores hashed backup codes
func (r *TwoFARepository) CreateBackupCodes(ctx context.Context, userID xid.ID, hashes []string) error {
    if len(hashes) == 0 { return nil }
    for _, h := range hashes {
        bc := &schema.BackupCode{ID: xid.New(), UserID: userID, CodeHash: h}
        bc.AuditableModel.CreatedBy = userID
        bc.AuditableModel.UpdatedBy = userID
        if _, err := r.db.NewInsert().Model(bc).Exec(ctx); err != nil { return err }
    }
    return nil
}

// VerifyAndUseBackupCode verifies a backup code hash and marks it used
func (r *TwoFARepository) VerifyAndUseBackupCode(ctx context.Context, userID xid.ID, hash string) (bool, error) {
    bc := new(schema.BackupCode)
    err := r.db.NewSelect().Model(bc).
        Where("user_id = ?", userID).
        Where("code_hash = ?", hash).
        Where("used_at IS NULL").
        Scan(ctx)
    if err == sql.ErrNoRows { return false, nil }
    if err != nil { return false, err }
    now := time.Now()
    bc.UsedAt = &now
    if _, err := r.db.NewUpdate().Model(bc).WherePK().Exec(ctx); err != nil { return false, err }
    return true, nil
}

// Trusted devices
func (r *TwoFARepository) MarkTrustedDevice(ctx context.Context, userID xid.ID, deviceID string, expiresAt time.Time) error {
    td := &schema.TrustedDevice{ID: xid.New(), UserID: userID, DeviceID: deviceID, ExpiresAt: expiresAt}
    td.AuditableModel.CreatedBy = userID
    td.AuditableModel.UpdatedBy = userID
    _, err := r.db.NewInsert().Model(td).Exec(ctx)
    return err
}

func (r *TwoFARepository) IsTrustedDevice(ctx context.Context, userID xid.ID, deviceID string, now time.Time) (bool, error) {
    td := new(schema.TrustedDevice)
    err := r.db.NewSelect().Model(td).
        Where("user_id = ?", userID).
        Where("device_id = ?", deviceID).
        Where("expires_at > ?", now).
        Scan(ctx)
    if err == sql.ErrNoRows { return false, nil }
    if err != nil { return false, err }
    return true, nil
}

// OTP codes
func (r *TwoFARepository) CreateOTPCode(ctx context.Context, userID xid.ID, codeHash string, expiresAt time.Time) error {
    oc := &schema.OTPCode{ID: xid.New(), UserID: userID, CodeHash: codeHash, ExpiresAt: expiresAt, Attempts: 0}
    oc.AuditableModel.CreatedBy = userID
    oc.AuditableModel.UpdatedBy = userID
    _, err := r.db.NewInsert().Model(oc).Exec(ctx)
    return err
}

func (r *TwoFARepository) VerifyOTPCode(ctx context.Context, userID xid.ID, codeHash string, now time.Time, maxAttempts int) (bool, error) {
    oc := new(schema.OTPCode)
    err := r.db.NewSelect().Model(oc).
        Where("user_id = ?", userID).
        Where("expires_at > ?", now).
        Scan(ctx)
    if err == sql.ErrNoRows { return false, nil }
    if err != nil { return false, err }
    // simple attempts cap
    if oc.Attempts >= maxAttempts { return false, nil }
    // verify hash match
    if oc.CodeHash != codeHash {
        oc.Attempts++
        _, _ = r.db.NewUpdate().Model(oc).WherePK().Exec(ctx)
        return false, nil
    }
    // consume OTP by expiring now
    oc.ExpiresAt = now
    _, err = r.db.NewUpdate().Model(oc).WherePK().Exec(ctx)
    if err != nil { return false, err }
    return true, nil
}