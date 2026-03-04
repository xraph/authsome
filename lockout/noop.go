package lockout

import (
	"context"
	"time"
)

// NoopTracker is a lockout tracker that never locks out.
type NoopTracker struct{}

// NewNoopTracker creates a no-op lockout tracker.
func NewNoopTracker() *NoopTracker { return &NoopTracker{} }

var _ Tracker = (*NoopTracker)(nil)

// RecordFailure always returns 0 attempts.
func (*NoopTracker) RecordFailure(context.Context, string) (int, error) { return 0, nil }

// IsLocked always returns false.
func (*NoopTracker) IsLocked(context.Context, string) (bool, time.Time, error) {
	return false, time.Time{}, nil
}

// Reset is a no-op.
func (*NoopTracker) Reset(context.Context, string) error { return nil }
