package device

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for device tracking operations.
type Store interface {
	CreateDevice(ctx context.Context, d *Device) error
	GetDevice(ctx context.Context, deviceID id.DeviceID) (*Device, error)
	GetDeviceByFingerprint(ctx context.Context, userID id.UserID, fingerprint string) (*Device, error)
	UpdateDevice(ctx context.Context, d *Device) error
	DeleteDevice(ctx context.Context, deviceID id.DeviceID) error
	ListUserDevices(ctx context.Context, userID id.UserID) ([]*Device, error)
	// ListDevices returns the most recent devices across all users, up to limit.
	ListDevices(ctx context.Context, limit int) ([]*Device, error)
}
