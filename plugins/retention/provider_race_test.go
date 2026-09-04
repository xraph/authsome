package retention

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRegisterProviderConcurrentWithDeliveryDoesNotRace is the regression
// test for the provider-map race: OnInit used to hand the delivery worker
// the very map RegisterProvider keeps writing to, so a RegisterProvider call
// after OnInit raced the worker goroutine's lookups on the same map with no
// synchronisation on either side.
//
// This drives both at once, under -race, against the real Plugin (not a
// hand-built worker) so it exercises the actual OnInit wiring. It proves the
// fix by construction rather than by timing: OnInit now hands the worker an
// immutable snapshot distinct from p.providers, so RegisterProvider and the
// worker's lookups never touch the same map again, and there is nothing for
// -race to catch regardless of how the two goroutines interleave.
func TestRegisterProviderConcurrentWithDeliveryDoesNotRace(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	p := New()
	p.SetStore(s)
	require.NoError(t, p.OnInit(ctx, &stubEngine{}))
	t.Cleanup(func() {
		require.NoError(t, p.OnShutdown(ctx))
	})

	// Keep the worker's claim loop busy across the whole run, so its
	// deliverOne calls keep reaching the Providers map lookup (the racy
	// access) rather than finding nothing to claim after the first pass.
	for i := 0; i < 200; i++ {
		enqueued(t, s, KindContactUpsert, fmt.Sprintf("race-%d", i))
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			p.RegisterProvider(&fakeProvider{caps: CapContacts})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			p.worker.runOnce(ctx)
		}
	}()

	wg.Wait()
}
