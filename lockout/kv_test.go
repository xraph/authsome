package lockout_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/store/memory"
)

func TestKVTracker_RecordAndLock(t *testing.T) {
	tracker := lockout.NewKVTracker(memory.New(), lockout.WithMaxAttempts(3), lockout.WithLockoutDuration(time.Minute))
	ctx := context.Background()
	key := "app:user@example.com:10.0.0.0/24"

	locked, _, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked)

	for i := 0; i < 3; i++ {
		attempts, recordErr := tracker.RecordFailure(ctx, key)
		require.NoError(t, recordErr)
		assert.Equal(t, i+1, attempts)
	}
	locked, until, err := tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.True(t, locked)
	assert.True(t, until.After(time.Now()))

	require.NoError(t, tracker.Reset(ctx, key))
	locked, _, err = tracker.IsLocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, locked, "reset clears the lock")
}

func TestKVTracker_LockoutExpires(t *testing.T) {
	tracker := lockout.NewKVTracker(memory.New(), lockout.WithMaxAttempts(1), lockout.WithLockoutDuration(30*time.Millisecond))
	ctx := context.Background()
	_, err := tracker.RecordFailure(ctx, "k:x")
	require.NoError(t, err)
	locked, _, err := tracker.IsLocked(ctx, "k:x")
	require.NoError(t, err)
	assert.True(t, locked)

	time.Sleep(60 * time.Millisecond)
	locked, _, err = tracker.IsLocked(ctx, "k:x")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestKVTracker_ResetPrefixClearsEveryNetwork(t *testing.T) {
	kv := memory.New()
	tracker := lockout.NewKVTracker(kv, lockout.WithMaxAttempts(1), lockout.WithLockoutDuration(time.Minute))
	ctx := context.Background()
	prefix := "app:user@example.com:"
	for _, net := range []string{"10.0.0.0/24", "192.168.1.0/24"} {
		_, err := tracker.RecordFailure(ctx, prefix+net)
		require.NoError(t, err)
	}
	_, err := tracker.RecordFailure(ctx, "app:other@example.com:10.0.0.0/24")
	require.NoError(t, err)

	require.NoError(t, tracker.ResetPrefix(ctx, prefix))

	for _, net := range []string{"10.0.0.0/24", "192.168.1.0/24"} {
		locked, _, lockErr := tracker.IsLocked(ctx, prefix+net)
		require.NoError(t, lockErr)
		assert.False(t, locked, "%s is unlocked", net)
	}
	locked, _, err := tracker.IsLocked(ctx, "app:other@example.com:10.0.0.0/24")
	require.NoError(t, err)
	assert.True(t, locked, "another account's lock is untouched")

	// A second replica sharing the store sees the same state.
	other := lockout.NewKVTracker(kv, lockout.WithMaxAttempts(1))
	locked, _, err = other.IsLocked(ctx, "app:other@example.com:10.0.0.0/24")
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestMemoryTracker_ResetPrefix(t *testing.T) {
	tracker := lockout.NewMemoryTracker(lockout.WithMaxAttempts(1))
	ctx := context.Background()
	_, err := tracker.RecordFailure(ctx, "app:u:a")
	require.NoError(t, err)
	_, err = tracker.RecordFailure(ctx, "app:u:b")
	require.NoError(t, err)
	require.NoError(t, tracker.ResetPrefix(ctx, "app:u:"))
	locked, _, err := tracker.IsLocked(ctx, "app:u:a")
	require.NoError(t, err)
	assert.False(t, locked)
}
