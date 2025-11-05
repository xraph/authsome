package security

import "context"

// Repository defines persistence for security events
type Repository interface {
	Create(ctx context.Context, e *SecurityEvent) error
}
