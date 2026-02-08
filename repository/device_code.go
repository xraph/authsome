package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// DeviceCodeRepository handles device code persistence
type DeviceCodeRepository struct {
	db *bun.DB
}

// NewDeviceCodeRepository creates a new device code repository
func NewDeviceCodeRepository(db *bun.DB) *DeviceCodeRepository {
	return &DeviceCodeRepository{db: db}
}

// Create stores a new device code
func (r *DeviceCodeRepository) Create(ctx context.Context, code *schema.DeviceCode) error {
	_, err := r.db.NewInsert().Model(code).Exec(ctx)
	return err
}

// FindByDeviceCode retrieves a device code by its device_code value
func (r *DeviceCodeRepository) FindByDeviceCode(ctx context.Context, deviceCode string) (*schema.DeviceCode, error) {
	dc := &schema.DeviceCode{}
	err := r.db.NewSelect().
		Model(dc).
		Where("device_code = ?", deviceCode).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dc, nil
}

// FindByUserCode retrieves a device code by its user_code value
func (r *DeviceCodeRepository) FindByUserCode(ctx context.Context, userCode string) (*schema.DeviceCode, error) {
	dc := &schema.DeviceCode{}
	err := r.db.NewSelect().
		Model(dc).
		Where("user_code = ?", userCode).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dc, nil
}

// FindByDeviceCodeWithContext retrieves a device code with context filtering
func (r *DeviceCodeRepository) FindByDeviceCodeWithContext(ctx context.Context, deviceCode string, appID, envID xid.ID, orgID *xid.ID) (*schema.DeviceCode, error) {
	dc := &schema.DeviceCode{}
	query := r.db.NewSelect().Model(dc).
		Where("device_code = ?", deviceCode).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if orgID != nil && !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dc, nil
}

// UpdateStatus updates the status of a device code
func (r *DeviceCodeRepository) UpdateStatus(ctx context.Context, deviceCode, status string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.DeviceCode)(nil)).
		Set("status = ?", status).
		Set("updated_at = ?", now).
		Where("device_code = ?", deviceCode).
		Exec(ctx)
	return err
}

// AuthorizeDevice marks a device code as authorized with user and session info
func (r *DeviceCodeRepository) AuthorizeDevice(ctx context.Context, userCode string, userID, sessionID xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.DeviceCode)(nil)).
		Set("status = ?", schema.DeviceCodeStatusAuthorized).
		Set("user_id = ?", userID).
		Set("session_id = ?", sessionID).
		Set("updated_at = ?", now).
		Where("user_code = ?", userCode).
		Exec(ctx)
	return err
}

// DenyDevice marks a device code as denied
func (r *DeviceCodeRepository) DenyDevice(ctx context.Context, userCode string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.DeviceCode)(nil)).
		Set("status = ?", schema.DeviceCodeStatusDenied).
		Set("updated_at = ?", now).
		Where("user_code = ?", userCode).
		Exec(ctx)
	return err
}

// MarkAsConsumed marks a device code as consumed (after token exchange)
func (r *DeviceCodeRepository) MarkAsConsumed(ctx context.Context, deviceCode string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.DeviceCode)(nil)).
		Set("status = ?", schema.DeviceCodeStatusConsumed).
		Set("updated_at = ?", now).
		Where("device_code = ?", deviceCode).
		Exec(ctx)
	return err
}

// UpdatePollInfo updates polling metadata
func (r *DeviceCodeRepository) UpdatePollInfo(ctx context.Context, deviceCode string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.DeviceCode)(nil)).
		Set("poll_count = poll_count + 1").
		Set("last_polled_at = ?", now).
		Set("updated_at = ?", now).
		Where("device_code = ?", deviceCode).
		Exec(ctx)
	return err
}

// DeleteExpired removes expired device codes
func (r *DeviceCodeRepository) DeleteExpired(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.DeviceCode)(nil)).
		Where("expires_at < ?", time.Now()).
		Where("status = ?", schema.DeviceCodeStatusPending).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// DeleteOldConsumedCodes removes consumed device codes older than the specified duration
func (r *DeviceCodeRepository) DeleteOldConsumedCodes(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.NewDelete().
		Model((*schema.DeviceCode)(nil)).
		Where("status = ?", schema.DeviceCodeStatusConsumed).
		Where("updated_at < ?", cutoff).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// FindByClientID retrieves device codes for a specific client
func (r *DeviceCodeRepository) FindByClientID(ctx context.Context, clientID string, limit int) ([]*schema.DeviceCode, error) {
	var codes []*schema.DeviceCode
	query := r.db.NewSelect().
		Model(&codes).
		Where("client_id = ?", clientID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Scan(ctx)
	return codes, err
}

// FindByUserID retrieves device codes for a specific user
func (r *DeviceCodeRepository) FindByUserID(ctx context.Context, userID xid.ID, limit int) ([]*schema.DeviceCode, error) {
	var codes []*schema.DeviceCode
	query := r.db.NewSelect().
		Model(&codes).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Scan(ctx)
	return codes, err
}

// DeleteBySession removes device codes associated with a session
func (r *DeviceCodeRepository) DeleteBySession(ctx context.Context, sessionID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.DeviceCode)(nil)).
		Where("session_id = ?", sessionID).
		Exec(ctx)
	return err
}

// CountPending counts pending device codes for a client
func (r *DeviceCodeRepository) CountPending(ctx context.Context, clientID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.DeviceCode)(nil)).
		Where("client_id = ?", clientID).
		Where("status = ?", schema.DeviceCodeStatusPending).
		Where("expires_at > ?", time.Now()).
		Count(ctx)
	return count, err
}

// ListByAppAndEnv returns device codes for an app with pagination
func (r *DeviceCodeRepository) ListByAppAndEnv(ctx context.Context, appID, envID xid.ID, page, pageSize int) ([]*schema.DeviceCode, error) {
	var codes []*schema.DeviceCode
	offset := (page - 1) * pageSize

	err := r.db.NewSelect().Model(&codes).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx)

	return codes, err
}

// CountByAppAndEnv returns total count of device codes for an app
func (r *DeviceCodeRepository) CountByAppAndEnv(ctx context.Context, appID, envID xid.ID) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.DeviceCode)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Count(ctx)
	return int64(count), err
}

// ListByAppEnvAndStatus returns device codes filtered by status
func (r *DeviceCodeRepository) ListByAppEnvAndStatus(ctx context.Context, appID, envID xid.ID, status string, page, pageSize int) ([]*schema.DeviceCode, error) {
	var codes []*schema.DeviceCode
	offset := (page - 1) * pageSize

	err := r.db.NewSelect().Model(&codes).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx)

	return codes, err
}

// CountByAppEnvAndStatus returns count of device codes by status
func (r *DeviceCodeRepository) CountByAppEnvAndStatus(ctx context.Context, appID, envID xid.ID, status string) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.DeviceCode)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("status = ?", status).
		Count(ctx)
	return int64(count), err
}

// Update updates a device code
func (r *DeviceCodeRepository) Update(ctx context.Context, code *schema.DeviceCode) error {
	_, err := r.db.NewUpdate().Model(code).WherePK().Exec(ctx)
	return err
}
