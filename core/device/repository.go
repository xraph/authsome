package device

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines device persistence operations
// Following Interface Segregation Principle (ISP) - works with schema types.
type Repository interface {
	// Create/Read operations
	CreateDevice(ctx context.Context, d *schema.Device) error
	FindDeviceByID(ctx context.Context, id xid.ID) (*schema.Device, error)
	FindDeviceByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) (*schema.Device, error)

	// List with pagination
	ListDevices(ctx context.Context, filter *ListDevicesFilter) (*pagination.PageResponse[*schema.Device], error)

	// Update operations
	UpdateDevice(ctx context.Context, d *schema.Device) error
	DeleteDevice(ctx context.Context, id xid.ID) error
	DeleteDeviceByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) error

	// Count operations
	CountDevices(ctx context.Context, userID xid.ID) (int, error)
}
