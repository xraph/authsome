package lockout_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/lockout"
)

func TestMemoryTracker_RecordAndLock(t *testing.T) {
	tracker := lockout.NewMemoryTracker(
		lockout.WithMaxAttempts(3),
		lockout.WithLockoutDuration(100*time.Millisecond),
	)
	ctx := context.Background()
	key := "user@example.com"

	// Not locked initially
	locked, _, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked)

	// Record failures
	for i := 0; i < 3; i++ {
		attempts, recordErr := tracker.RecordFailure(ctx, key)
		require.NoError(t, recordErr)
		assert.Equal(t, i+1, attempts)
	}

	// Should now be locked
	locked, until, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.True(t, locked)
	assert.True(t, until.After(time.Now()))
}

func TestMemoryTracker_LockoutExpires(t *testing.T) {
	tracker := lockout.NewMemoryTracker(
		lockout.WithMaxAttempts(2),
		lockout.WithLockoutDuration(50*time.Millisecond),
	)
	ctx := context.Background()
	key := "user2@example.com"

	_, _ = tracker.RecordFailure(ctx, key)
	_, _ = tracker.RecordFailure(ctx, key)

	locked, _, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.True(t, locked)

	// Wait for lockout to expire
	time.Sleep(60 * time.Millisecond)

	locked, _, err = tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestMemoryTracker_Reset(t *testing.T) {
	tracker := lockout.NewMemoryTracker(
		lockout.WithMaxAttempts(3),
		lockout.WithLockoutDuration(10*time.Second),
	)
	ctx := context.Background()
	key := "reset@example.com"

	// Record some failures
	_, _ = tracker.RecordFailure(ctx, key)
	_, _ = tracker.RecordFailure(ctx, key)

	// Reset (simulate successful login)
	err := tracker.Reset(ctx, key)
	require.NoError(t, err)

	// Should not be locked and counter is reset
	locked, _, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked)

	// Should take another 3 failures to lock
	_, _ = tracker.RecordFailure(ctx, key)
	_, _ = tracker.RecordFailure(ctx, key)

	locked, _, err = tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked)

	_, _ = tracker.RecordFailure(ctx, key)

	locked, _, err = tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestMemoryTracker_ResetAfterInactivity(t *testing.T) {
	tracker := lockout.NewMemoryTracker(
		lockout.WithMaxAttempts(3),
		lockout.WithLockoutDuration(10*time.Second),
		lockout.WithResetAfter(50*time.Millisecond),
	)
	ctx := context.Background()
	key := "inactive@example.com"

	// Record 2 failures
	_, _ = tracker.RecordFailure(ctx, key)
	_, _ = tracker.RecordFailure(ctx, key)

	// Wait for reset period
	time.Sleep(60 * time.Millisecond)

	// This should be attempt #1 (counter reset due to inactivity)
	attempts, err := tracker.RecordFailure(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, 1, attempts)

	// Should not be locked
	locked, _, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestMemoryTracker_DifferentKeys(t *testing.T) {
	tracker := lockout.NewMemoryTracker(lockout.WithMaxAttempts(2))
	ctx := context.Background()

	_, _ = tracker.RecordFailure(ctx, "key-a")
	_, _ = tracker.RecordFailure(ctx, "key-a")

	locked, _, _ := tracker.IsLocked(ctx, "key-a")
	assert.True(t, locked)

	locked, _, _ = tracker.IsLocked(ctx, "key-b")
	assert.False(t, locked)
}

func TestNoopTracker(t *testing.T) {
	tracker := lockout.NewNoopTracker()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		attempts, err := tracker.RecordFailure(ctx, "key")
		require.NoError(t, err)
		assert.Equal(t, 0, attempts)
	}

	locked, _, err := tracker.IsLocked(ctx, "key")
	require.NoError(t, err)
	assert.False(t, locked)

	err = tracker.Reset(ctx, "key")
	require.NoError(t, err)
}
