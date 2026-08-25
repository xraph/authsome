package consent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// MemoryStore guards everything with an RWMutex and nothing in this package
// ever called it from a second goroutine, so `go test -race` here watched no
// concurrent access and passed on that basis alone.
//
// The lock was not the whole story. RevokeConsent mutates a stored *Consent
// in place (Granted, RevokedAt, UpdatedAt) while holding the write lock, and
// GetConsent and ListConsents used to hand that same pointer back. A caller
// reading Granted off the value it received was reading a field another
// goroutine's RevokeConsent could be writing, with nothing ordering the two.
// The store copies on the way out now, and these tests are what would notice
// if that copy were ever dropped.

const (
	consentHammerWorkers = 4
	consentHammerBudget  = 150 * time.Millisecond
)

func newConsent(userID id.UserID, appID id.AppID, purpose string) *Consent {
	return &Consent{
		UserID: userID, AppID: appID, Purpose: purpose,
		Granted: true, Version: "v1", GrantedAt: time.Now(),
	}
}

// TestMemoryStore_ConcurrentGrantRevokeAndRead drives grant, revoke and read
// against overlapping keys from several goroutines at once. Readers write to
// what they are handed, which is the shape that turns an uncloned read into a
// data race rather than merely a stale one.
func TestMemoryStore_ConcurrentGrantRevokeAndRead(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	appID := id.NewAppID()
	users := make([]id.UserID, 6)
	for i := range users {
		users[i] = id.NewUserID()
	}
	purposes := []string{"marketing", "analytics"}

	deadline := time.Now().Add(consentHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < consentHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				u := users[(w+i)%len(users)]
				p := purposes[i%len(purposes)]

				if err := s.GrantConsent(ctx, newConsent(u, appID, p)); err != nil {
					t.Errorf("GrantConsent: %v", err)
					return
				}
				if got, err := s.GetConsent(ctx, u, appID, p); err == nil {
					// Write through what we were handed. Against the store's
					// own records this would be a race and a corruption.
					got.Granted = false
					got.Purpose = "tampered"
				}
				if i%3 == 0 {
					_ = s.RevokeConsent(ctx, u, appID, p)
				}
				if _, _, err := s.ListConsents(ctx, &Query{AppID: appID, Limit: 10}); err != nil {
					t.Errorf("ListConsents: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(200),
		"the hammer did no meaningful work; it is not exercising the store concurrently")

	// Every purpose string must still be one this test actually granted. If a
	// reader's write reached the store, "tampered" would be in there.
	all, _, err := s.ListConsents(ctx, &Query{AppID: appID, Limit: 1000})
	require.NoError(t, err)
	for _, c := range all {
		require.Contains(t, purposes, c.Purpose, "a reader's write reached the store's own record")
	}
}

// TestMemoryStore_GetDoesNotAliasStoredConsent pins the read side on its own,
// without concurrency, so a failure points straight at the missing copy.
func TestMemoryStore_GetDoesNotAliasStoredConsent(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID, appID := id.NewUserID(), id.NewAppID()

	require.NoError(t, s.GrantConsent(ctx, newConsent(userID, appID, "marketing")))

	got, err := s.GetConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	got.Granted = false
	got.Version = "tampered"

	fresh, err := s.GetConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	require.True(t, fresh.Granted, "revoking consent on a returned copy must not revoke it in the store")
	require.Equal(t, "v1", fresh.Version)
}

// TestMemoryStore_GrantDoesNotAliasCallerConsent is the mirror image: the
// caller keeps its own *Consent after GrantConsent returns, and writing to it
// must not reach into the store.
func TestMemoryStore_GrantDoesNotAliasCallerConsent(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID, appID := id.NewUserID(), id.NewAppID()

	c := newConsent(userID, appID, "marketing")
	require.NoError(t, s.GrantConsent(ctx, c))

	c.Granted = false
	c.Version = "tampered"

	got, err := s.GetConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	require.True(t, got.Granted, "mutating the caller's consent after GrantConsent must not affect the store")
	require.Equal(t, "v1", got.Version)
}

// TestMemoryStore_ListDoesNotAliasStoredConsents covers the same property for
// the list path, which returns many pointers rather than one and is the
// easier of the two to forget when adding a copy.
func TestMemoryStore_ListDoesNotAliasStoredConsents(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	for i := 0; i < 3; i++ {
		require.NoError(t, s.GrantConsent(ctx, newConsent(id.NewUserID(), appID, fmt.Sprintf("purpose-%d", i))))
	}

	first, _, err := s.ListConsents(ctx, &Query{AppID: appID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, first, 3)
	for _, c := range first {
		c.Granted = false
		c.Version = "tampered"
	}

	second, _, err := s.ListConsents(ctx, &Query{AppID: appID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, second, 3)
	for _, c := range second {
		require.True(t, c.Granted, "mutating a listed consent must not affect the store")
		require.Equal(t, "v1", c.Version)
	}
}

// TestMemoryStore_ConcurrentRevokeIsVisibleAndStable checks the state that
// matters after the dust settles. Many goroutines revoke the same consent;
// once any of them has returned, every later read must agree it is revoked.
func TestMemoryStore_ConcurrentRevokeIsVisibleAndStable(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID, appID := id.NewUserID(), id.NewAppID()
	require.NoError(t, s.GrantConsent(ctx, newConsent(userID, appID, "marketing")))

	var wg sync.WaitGroup
	for w := 0; w < consentHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = s.RevokeConsent(ctx, userID, appID, "marketing")
			}
		}()
	}
	// Readers running throughout, each writing to what it is handed.
	for w := 0; w < consentHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if got, err := s.GetConsent(ctx, userID, appID, "marketing"); err == nil {
					got.Granted = true
				}
			}
		}()
	}
	wg.Wait()

	got, err := s.GetConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	require.False(t, got.Granted, "the consent was revoked; no reader may have put it back")
	require.NotNil(t, got.RevokedAt)
}
