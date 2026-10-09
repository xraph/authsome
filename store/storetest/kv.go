package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// The KV contract every backend must honour: it is what rate limiting,
// lockout and ceremony state stand on, so a backend that drifts here makes
// those controls differ between deployments.

func kvKey(prefix string) string { return prefix + ":" + id.NewSessionID().String() }

func testKVRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	key := kvKey("rt")

	_, err := s.KVGet(ctx, key)
	assert.ErrorIs(t, err, store.ErrNotFound, "an absent key is not found")

	require.NoError(t, s.KVSet(ctx, key, []byte("one"), time.Minute))
	got, err := s.KVGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("one"), got)

	require.NoError(t, s.KVSet(ctx, key, []byte("two"), time.Minute))
	got, err = s.KVGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("two"), got, "set replaces")

	require.NoError(t, s.KVSet(ctx, key, []byte("stale"), -time.Second))
	_, err = s.KVGet(ctx, key)
	assert.ErrorIs(t, err, store.ErrNotFound, "an expired key reads as absent")

	require.NoError(t, s.KVDelete(ctx, key))
	require.NoError(t, s.KVDelete(ctx, key), "deleting an absent key is fine")
}

func testKVSetNXHonoursExpiry(t *testing.T, s store.Store) {
	ctx := context.Background()
	key := kvKey("nx")

	set, err := s.KVSetNX(ctx, key, []byte("first"), time.Minute)
	require.NoError(t, err)
	assert.True(t, set, "an absent key is set")

	set, err = s.KVSetNX(ctx, key, []byte("second"), time.Minute)
	require.NoError(t, err)
	assert.False(t, set, "a live key is not replaced")
	got, err := s.KVGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("first"), got)

	require.NoError(t, s.KVSet(ctx, key, []byte("old"), -time.Second))
	set, err = s.KVSetNX(ctx, key, []byte("third"), time.Minute)
	require.NoError(t, err)
	assert.True(t, set, "an expired key counts as absent")
	got, err = s.KVGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("third"), got)
}

func testKVIncrementWindow(t *testing.T, s store.Store) {
	ctx := context.Background()
	key := kvKey("inc")

	n0, err := s.KVCounter(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n0, "an absent counter reads as zero")

	for want := int64(1); want <= 3; want++ {
		n, incErr := s.KVIncrement(ctx, key, time.Minute)
		require.NoError(t, incErr)
		assert.Equal(t, want, n)
	}
	read, err := s.KVCounter(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(3), read, "the counter reads back without changing")

	// The window is fixed from the first increment: a later increment does
	// not extend it, so an expired counter starts over at one.
	require.NoError(t, s.KVSet(ctx, key, nil, -time.Second))
	n, err := s.KVIncrement(ctx, key, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "an expired counter restarts")
	n, err = s.KVIncrement(ctx, key, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func testKVDeleteExpired(t *testing.T, s store.Store) {
	ctx := context.Background()
	live := kvKey("live")
	dead := kvKey("dead")
	require.NoError(t, s.KVSet(ctx, live, []byte("l"), time.Hour))
	require.NoError(t, s.KVSet(ctx, dead, []byte("d"), -time.Second))

	n, err := s.KVDeleteExpired(ctx, time.Now())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(1), "at least the dead key goes")

	_, err = s.KVGet(ctx, live)
	assert.NoError(t, err, "the live key survives")
	_, err = s.KVGet(ctx, dead)
	assert.ErrorIs(t, err, store.ErrNotFound)
}
