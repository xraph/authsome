package repository

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// DeviceRepository implements core device repository using Bun
type DeviceRepository struct {
	db *bun.DB
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *bun.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// CreateDevice creates a new device
func (r *DeviceRepository) CreateDevice(ctx context.Context, d *schema.Device) error {
	_, err := r.db.NewInsert().Model(d).Exec(ctx)
	return err
}

// FindDeviceByID finds a device by ID
func (r *DeviceRepository) FindDeviceByID(ctx context.Context, id xid.ID) (*schema.Device, error) {
	device := &schema.Device{}
	err := r.db.NewSelect().
		Model(device).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// FindDeviceByFingerprint finds a device by user ID and fingerprint
func (r *DeviceRepository) FindDeviceByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) (*schema.Device, error) {
	device := &schema.Device{}
	err := r.db.NewSelect().
		Model(device).
		Where("user_id = ?", userID).
		Where("fingerprint = ?", fingerprint).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// ListDevices lists devices with filtering and pagination
func (r *DeviceRepository) ListDevices(ctx context.Context, filter *device.ListDevicesFilter) (*pagination.PageResponse[*schema.Device], error) {
	var devices []*schema.Device

	// Build query with filters
	query := r.db.NewSelect().Model(&devices).Where("deleted_at IS NULL")
	query = query.Where("user_id = ?", filter.UserID)

	// Count query with same filters
	countQuery := r.db.NewSelect().Model((*schema.Device)(nil)).Where("deleted_at IS NULL")
	countQuery = countQuery.Where("user_id = ?", filter.UserID)

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination and ordering
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())
	query = query.Order("last_active DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(devices, int64(total), &filter.PaginationParams), nil
}

// UpdateDevice updates a device
func (r *DeviceRepository) UpdateDevice(ctx context.Context, d *schema.Device) error {
	_, err := r.db.NewUpdate().
		Model(d).
		Where("id = ?", d.ID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// DeleteDevice soft deletes a device by ID
func (r *DeviceRepository) DeleteDevice(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Device)(nil)).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// DeleteDeviceByFingerprint soft deletes a device by user ID and fingerprint
func (r *DeviceRepository) DeleteDeviceByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) error {
	_, err := r.db.NewDelete().
		Model((*schema.Device)(nil)).
		Where("user_id = ?", userID).
		Where("fingerprint = ?", fingerprint).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// CountDevices counts devices for a user
func (r *DeviceRepository) CountDevices(ctx context.Context, userID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Device)(nil)).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Count(ctx)
	return count, err
}
