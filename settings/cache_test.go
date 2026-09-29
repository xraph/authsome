package settings

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"
)

// countingStore counts the resolve calls that reach the store.
type countingStore struct {
	*memStore
	resolves atomic.Int64
	batches  atomic.Int64
}

func (s *countingStore) ResolveSettings(ctx context.Context, key string, opts ResolveOpts) ([]*Setting, error) {
	s.resolves.Add(1)
	return s.memStore.ResolveSettings(ctx, key, opts)
}

func (s *countingStore) BatchResolve(ctx context.Context, keys []string, opts ResolveOpts) (map[string][]*Setting, error) {
	s.batches.Add(1)
	return s.memStore.BatchResolve(ctx, keys, opts)
}

func newCachedManager(t *testing.T, ttl time.Duration) (*Manager, *countingStore) {
	t.Helper()
	st := &countingStore{memStore: newMemStore()}
	m := NewManager(st, log.NewNoopLogger(), WithCacheTTL(ttl))
	require.NoError(t, m.Register(Definition{
		Key: "test.flag", Namespace: "test", Type: TypeBool, Default: json.RawMessage("false"),
		Scopes: []Scope{ScopeGlobal, ScopeApp},
	}))
	return m, st
}

func TestCache_ServesRepeatReadsFromMemory(t *testing.T) {
	m, st := newCachedManager(t, time.Minute)
	ctx := context.Background()
	opts := ResolveOpts{AppID: "app-a"}
	for range 5 {
		_, err := m.Resolve(ctx, "test.flag", opts)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, st.resolves.Load(), "the store answers once per window")
}

func TestCache_WriteDropsTheKey(t *testing.T) {
	m, _ := newCachedManager(t, time.Minute)
	ctx := context.Background()
	opts := ResolveOpts{AppID: "app-a"}
	v, err := m.Resolve(ctx, "test.flag", opts)
	require.NoError(t, err)
	assert.JSONEq(t, "false", string(v))

	require.NoError(t, m.Set(ctx, "test.flag", json.RawMessage("true"), ScopeApp, "app-a", "app-a", "", "tester"))
	v, err = m.Resolve(ctx, "test.flag", opts)
	require.NoError(t, err)
	assert.JSONEq(t, "true", string(v), "a write is visible at once")

	require.NoError(t, m.Delete(ctx, "test.flag", ScopeApp, "app-a"))
	v, err = m.Resolve(ctx, "test.flag", opts)
	require.NoError(t, err)
	assert.JSONEq(t, "false", string(v), "a delete is visible at once")
}

func TestCache_ScopesDoNotBleed(t *testing.T) {
	m, _ := newCachedManager(t, time.Minute)
	ctx := context.Background()
	require.NoError(t, m.Set(ctx, "test.flag", json.RawMessage("true"), ScopeApp, "app-a", "app-a", "", "tester"))
	a, err := m.Resolve(ctx, "test.flag", ResolveOpts{AppID: "app-a"})
	require.NoError(t, err)
	b, err := m.Resolve(ctx, "test.flag", ResolveOpts{AppID: "app-b"})
	require.NoError(t, err)
	assert.JSONEq(t, "true", string(a))
	assert.JSONEq(t, "false", string(b), "app B never sees app A's cached rows")
}

func TestCache_ExpiresAndCanBeDisabled(t *testing.T) {
	m, st := newCachedManager(t, 20*time.Millisecond)
	ctx := context.Background()
	opts := ResolveOpts{AppID: "app-a"}
	_, err := m.Resolve(ctx, "test.flag", opts)
	require.NoError(t, err)
	time.Sleep(40 * time.Millisecond)
	_, err = m.Resolve(ctx, "test.flag", opts)
	require.NoError(t, err)
	assert.EqualValues(t, 2, st.resolves.Load(), "an expired entry is fetched again")

	off, offStore := newCachedManager(t, 0)
	for range 3 {
		_, err = off.Resolve(ctx, "test.flag", opts)
		require.NoError(t, err)
	}
	assert.EqualValues(t, 3, offStore.resolves.Load(), "a zero TTL disables the cache")
}

func TestCache_BatchFetchesOnlyMissingKeys(t *testing.T) {
	m, st := newCachedManager(t, time.Minute)
	require.NoError(t, m.Register(Definition{
		Key: "test.other", Namespace: "test", Type: TypeBool, Default: json.RawMessage("true"),
		Scopes: []Scope{ScopeGlobal, ScopeApp},
	}))
	ctx := context.Background()
	opts := ResolveOpts{AppID: "app-a"}
	_, err := m.Resolve(ctx, "test.flag", opts)
	require.NoError(t, err)

	out, err := m.BatchResolve(ctx, []string{"test.flag", "test.other"}, opts)
	require.NoError(t, err)
	assert.JSONEq(t, "false", string(out["test.flag"]))
	assert.JSONEq(t, "true", string(out["test.other"]))
	assert.EqualValues(t, 1, st.batches.Load())

	_, err = m.BatchResolve(ctx, []string{"test.flag", "test.other"}, opts)
	require.NoError(t, err)
	assert.EqualValues(t, 1, st.batches.Load(), "every key was cached, so the store was not asked")
}
