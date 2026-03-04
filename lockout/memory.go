package lockout

import (
	"context"
	"sync"
	"time"
)

// MemoryTracker is an in-memory account lockout tracker.
type MemoryTracker struct {
	mu              sync.Mutex
	records         map[string]*record
	maxAttempts     int
	lockoutDuration time.Duration
	resetAfter      time.Duration
}

type record struct {
	attempts    int
	lastFailure time.Time
	lockedUntil time.Time
}

// MemoryOption configures the memory tracker.
type MemoryOption func(*MemoryTracker)

// WithMaxAttempts sets the max failed attempts before lockout.
func WithMaxAttempts(n int) MemoryOption {
	return func(t *MemoryTracker) { t.maxAttempts = n }
}

// WithLockoutDuration sets how long the lockout lasts.
func WithLockoutDuration(d time.Duration) MemoryOption {
	return func(t *MemoryTracker) { t.lockoutDuration = d }
}

// WithResetAfter sets the inactivity period after which the failure count resets.
func WithResetAfter(d time.Duration) MemoryOption {
	return func(t *MemoryTracker) { t.resetAfter = d }
}

// NewMemoryTracker creates an in-memory lockout tracker.
func NewMemoryTracker(opts ...MemoryOption) *MemoryTracker {
	t := &MemoryTracker{
		records:         make(map[string]*record),
		maxAttempts:     5,
		lockoutDuration: 15 * time.Minute,
		resetAfter:      1 * time.Hour,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

var _ Tracker = (*MemoryTracker)(nil)

// RecordFailure records a failed attempt.
func (t *MemoryTracker) RecordFailure(_ context.Context, key string) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	r := t.getOrCreate(key)

	// Reset if the last failure was long ago
	if !r.lastFailure.IsZero() && now.Sub(r.lastFailure) > t.resetAfter {
		r.attempts = 0
	}

	r.attempts++
	r.lastFailure = now

	// Lock if threshold exceeded
	if r.attempts >= t.maxAttempts {
		r.lockedUntil = now.Add(t.lockoutDuration)
	}

	return r.attempts, nil
}

// IsLocked checks if the key is locked out.
func (t *MemoryTracker) IsLocked(_ context.Context, key string) (bool, time.Time, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	r, ok := t.records[key]
	if !ok {
		return false, time.Time{}, nil
	}

	now := time.Now()
	if !r.lockedUntil.IsZero() && now.Before(r.lockedUntil) {
		return true, r.lockedUntil, nil
	}

	// Lockout expired — clear it
	if !r.lockedUntil.IsZero() {
		r.lockedUntil = time.Time{}
		r.attempts = 0
	}

	return false, time.Time{}, nil
}

// Reset clears the failure count for a key.
func (t *MemoryTracker) Reset(_ context.Context, key string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, key)
	return nil
}

func (t *MemoryTracker) getOrCreate(key string) *record {
	r, ok := t.records[key]
	if !ok {
		r = &record{}
		t.records[key] = r
	}
	return r
}
