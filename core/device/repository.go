package device

import (
	"context"
	"github.com/rs/xid"
)

// Repository defines device persistence operations
type Repository interface {
	Create(ctx context.Context, d *Device) error
	Update(ctx context.Context, d *Device) error
	FindByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) (*Device, error)
	ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*Device, error)
	DeleteByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) error
}
