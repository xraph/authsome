package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/store/memory"
)

func TestKVLimiter_Allow(t *testing.T) {
	limiter := ratelimit.NewKVLimiter(memory.New())
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, "k", 3, time.Minute)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d fits", i+1)
	}
	allowed, err := limiter.Allow(ctx, "k", 3, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed, "the fourth request is refused")

	allowed, err = limiter.Allow(ctx, "other", 3, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed, "keys are independent")
}

func TestKVLimiter_WindowResets(t *testing.T) {
	limiter := ratelimit.NewKVLimiter(memory.New())
	ctx := context.Background()
	window := 40 * time.Millisecond

	allowed, err := limiter.Allow(ctx, "k", 1, window)
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = limiter.Allow(ctx, "k", 1, window)
	require.NoError(t, err)
	assert.False(t, allowed)

	time.Sleep(2 * window)
	allowed, err = limiter.Allow(ctx, "k", 1, window)
	require.NoError(t, err)
	assert.True(t, allowed, "a new window starts clean")
}

func TestKVLimiter_SharedAcrossInstances(t *testing.T) {
	kv := memory.New()
	a := ratelimit.NewKVLimiter(kv)
	b := ratelimit.NewKVLimiter(kv)
	ctx := context.Background()

	allowed, err := a.Allow(ctx, "k", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = b.Allow(ctx, "k", 1, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed, "a second replica sees the first replica's count")
}
