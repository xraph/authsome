package authsome

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

func newEngineForCeremonyTest(t *testing.T) *Engine {
	t.Helper()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := NewEngine(
		WithStore(memory.New()),
		WithWarden(w),
		WithDisableMigrate(),
	)
	require.NoError(t, err)
	return eng
}

// TestCeremonyStore_EagerlyAllocatedAndStable pins the fix for the ceremony
// store data race: when no store is configured, NewEngine allocates one up
// front, and the accessor returns that same instance every time without
// mutating engine state. Previously the store was lazily created on the first
// request goroutine, racing concurrent MFA-gated logins and dropping tickets.
func TestCeremonyStore_EagerlyAllocatedAndStable(t *testing.T) {
	eng := newEngineForCeremonyTest(t)

	first := eng.ceremonyStoreOrFallback()
	require.NotNil(t, first, "ceremony store must be allocated at construction, not lazily")

	second := eng.ceremonyStoreOrFallback()
	require.Same(t, first, second, "accessor must return the same store, never re-allocate")
}

// TestCeremonyStore_ConcurrentAccessNoRace exercises the accessor from many
// goroutines. With the lazy nil-check-and-assign removed, `go test -race`
// reports no data race on the engine's ceremonyStore field.
func TestCeremonyStore_ConcurrentAccessNoRace(t *testing.T) {
	eng := newEngineForCeremonyTest(t)

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s := eng.ceremonyStoreOrFallback(); s == nil {
				t.Error("ceremony store must never be nil")
			}
		}()
	}
	wg.Wait()
}
