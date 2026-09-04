package retention

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// TestRegisterProviderConcurrentWithDeliveryDoesNotRace is the regression
// test for the provider-map race: OnInit used to hand the delivery worker
// the very map RegisterProvider keeps writing to, so a RegisterProvider call
// after OnInit raced the worker goroutine's lookups on the same map with no
// synchronisation on either side.
//
// This drives both at once, under -race, against the real Plugin (not a
// hand-built worker) so it exercises the actual OnInit wiring. It proves the
// fix by construction rather than by timing: RegisterProvider and the
// worker's lookups now go through the same *providerRegistry, which
// publishes each new provider map with a single atomic pointer store and
// hands every reader an immutable map to range over. Neither side ever
// mutates a map another goroutine can see, so there is nothing for -race to
// catch regardless of how the two goroutines interleave. See
// TestProviderRegisteredAfterOnInitReachesTheWorker for the companion
// correctness test -- this one proves "does not race", not "does not
// dead-letter".
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

// userStubEngine is stubEngine plus a real GetUser, so loadContact (called by
// the delivery worker for every job) can resolve the one user this test
// cares about instead of failing with "user not found".
type userStubEngine struct {
	stubEngine
	u *user.User
}

func (e userStubEngine) GetUser(_ context.Context, uid id.UserID) (*user.User, error) {
	if e.u != nil && e.u.ID == uid {
		return e.u, nil
	}
	return nil, nil
}

// TestProviderRegisteredAfterOnInitReachesTheWorker is the correctness
// regression test for the provider-map bug (as distinct from
// TestRegisterProviderConcurrentWithDeliveryDoesNotRace above, which only
// proves the two sides no longer race).
//
// Before the fix, OnInit handed the worker a private snapshot of
// p.providers and RegisterProvider kept writing to the live map:
// enqueueFor (reading the live map) queued a job for a provider registered
// after OnInit, and the worker (reading the stale snapshot) could never
// find that provider, so deliverOne dead-lettered the job on its very first
// attempt -- permanently, since nothing here re-enqueues a dead row. The
// two sides simply disagreed about which providers existed.
//
// This drives the real hook -> real worker path with a provider registered
// only after OnInit has already built and started the worker, and asserts
// the job is actually delivered rather than dead-lettered.
func TestProviderRegisteredAfterOnInitReachesTheWorker(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID, userID := id.NewAppID(), id.NewUserID()
	u := &user.User{ID: userID, AppID: appID, Email: "late@example.com"}

	p := New()
	p.SetStore(s)
	require.NoError(t, p.OnInit(ctx, userStubEngine{u: u}))
	t.Cleanup(func() {
		require.NoError(t, p.OnShutdown(ctx))
	})

	// Registered after OnInit has already built the worker's dependencies --
	// exactly the sequence RegisterProvider's doc comment used to wave off
	// as "changes p.providers itself but not a worker already running
	// against the earlier snapshot".
	late := &fakeProvider{caps: CapContacts | CapActivities}
	p.RegisterProvider(late)

	require.NoError(t, p.OnAfterSignIn(ctx, u, nil))

	p.worker.runOnce(ctx)

	dead, err := s.ListDead(ctx, appID, 10)
	require.NoError(t, err)
	assert.Empty(t, dead,
		"a provider registered after OnInit must be visible to the worker, not silently dead-lettered")

	late.mu.Lock()
	activityCount := len(late.activity)
	late.mu.Unlock()
	assert.Equal(t, 1, activityCount,
		"the login activity enqueued for the late provider must actually be delivered")
}
