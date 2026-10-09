package ceremony_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/store/memory"
)

// Both stores must behave the same: the memory one is for a single
// process, the KV one for every replica sharing a database.
func stores() map[string]func() ceremony.Store {
	return map[string]func() ceremony.Store{
		"memory": func() ceremony.Store { return ceremony.NewMemory() },
		"kv":     func() ceremony.Store { return ceremony.NewKV(memory.New()) },
	}
}

func TestStore_Contract(t *testing.T) {
	for name, mk := range stores() {
		t.Run(name, func(t *testing.T) {
			s := mk()
			ctx := context.Background()

			_, err := s.Get(ctx, "missing")
			assert.ErrorIs(t, err, ceremony.ErrNotFound)

			require.NoError(t, s.Set(ctx, "k", []byte("v"), time.Minute))
			got, err := s.Get(ctx, "k")
			require.NoError(t, err)
			assert.Equal(t, []byte("v"), got)

			set, err := s.SetNX(ctx, "k", []byte("w"), time.Minute)
			require.NoError(t, err)
			assert.False(t, set, "a live key is not replaced")
			set, err = s.SetNX(ctx, "fresh", []byte("w"), time.Minute)
			require.NoError(t, err)
			assert.True(t, set)

			for want := int64(1); want <= 3; want++ {
				n, incErr := s.Increment(ctx, "n", time.Minute)
				require.NoError(t, incErr)
				assert.Equal(t, want, n)
			}

			require.NoError(t, s.Delete(ctx, "k"))
			_, err = s.Get(ctx, "k")
			assert.ErrorIs(t, err, ceremony.ErrNotFound)
			require.NoError(t, s.Delete(ctx, "k"), "delete is idempotent")
		})
	}
}

func TestStore_Expiry(t *testing.T) {
	for name, mk := range stores() {
		t.Run(name, func(t *testing.T) {
			s := mk()
			ctx := context.Background()
			require.NoError(t, s.Set(ctx, "k", []byte("v"), 20*time.Millisecond))
			_, err := s.Increment(ctx, "n", 20*time.Millisecond)
			require.NoError(t, err)
			time.Sleep(40 * time.Millisecond)

			_, err = s.Get(ctx, "k")
			assert.ErrorIs(t, err, ceremony.ErrNotFound)
			set, err := s.SetNX(ctx, "k", []byte("w"), time.Minute)
			require.NoError(t, err)
			assert.True(t, set, "an expired key can be claimed")
			n, err := s.Increment(ctx, "n", time.Minute)
			require.NoError(t, err)
			assert.Equal(t, int64(1), n, "an expired counter restarts")
		})
	}
}

func TestMemoryStore_IsBounded(t *testing.T) {
	s := ceremony.NewMemoryWithLimit(10)
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		require.NoError(t, s.Set(ctx, string(rune('a'+i%26))+string(rune('a'+i/26)), []byte("v"), time.Minute))
	}
	assert.LessOrEqual(t, s.Len(), 10, "the store never grows past its bound")
}
