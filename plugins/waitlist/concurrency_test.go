package waitlist

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

// MemoryStore locks every method, and nothing in this package called it from
// a second goroutine, so `go test -race` here proved nothing.
//
// The store mutates entries in place: UpdateEntryStatus writes Status, Note
// and UpdatedAt on a stored *WaitlistEntry under the write lock, while
// GetEntry, GetEntryByEmail and ListEntries used to return that same pointer.
// Reading Status off what you were handed was therefore a read of a field
// another goroutine could be writing, and a caller could also assign to it and
// approve itself off the waitlist without going through the store at all.

const (
	waitlistHammerWorkers = 4
	waitlistHammerBudget  = 150 * time.Millisecond
)

func newEntry(appID id.AppID, email string) *WaitlistEntry {
	return &WaitlistEntry{
		AppID: appID, Email: email, Name: "Test User",
		Status: StatusPending, IPAddress: "203.0.113.1",
	}
}

// TestMemoryStore_ConcurrentStatusUpdatesAndReads runs approvals against
// reads, with readers writing to what they receive.
func TestMemoryStore_ConcurrentStatusUpdatesAndReads(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	entries := make([]*WaitlistEntry, 6)
	for i := range entries {
		entries[i] = newEntry(appID, fmt.Sprintf("user%d@example.com", i))
		require.NoError(t, s.CreateEntry(ctx, entries[i]))
	}

	deadline := time.Now().Add(waitlistHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < waitlistHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				e := entries[(w+i)%len(entries)]
				if err := s.UpdateEntryStatus(ctx, e.ID, StatusRejected, "not this time"); err != nil {
					t.Errorf("UpdateEntryStatus: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}

	for w := 0; w < waitlistHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				e := entries[(w+i)%len(entries)]
				if got, err := s.GetEntry(ctx, e.ID); err == nil {
					// Approve ourselves, on our own copy. This must not reach
					// the store.
					got.Status = StatusApproved
					got.Note = "self-approved"
				}
				if got, err := s.GetEntryByEmail(ctx, appID, e.Email); err == nil {
					got.Status = StatusApproved
				}
				if _, err := s.ListEntries(ctx, &WaitlistQuery{AppID: appID, Limit: 10}); err != nil {
					t.Errorf("ListEntries: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(500),
		"the hammer did no meaningful work; it is not exercising the store concurrently")

	for i, e := range entries {
		got, err := s.GetEntry(ctx, e.ID)
		require.NoError(t, err)
		require.Equal(t, StatusRejected, got.Status,
			"entry %d was approved by a reader writing through the value it was handed", i)
		require.Equal(t, "not this time", got.Note)
	}
}

// TestMemoryStore_GetDoesNotAliasStoredEntry pins the read side on its own.
func TestMemoryStore_GetDoesNotAliasStoredEntry(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	e := newEntry(appID, "user@example.com")
	require.NoError(t, s.CreateEntry(ctx, e))

	got, err := s.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	got.Status = StatusApproved
	got.Note = "self-approved"

	fresh, err := s.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, StatusPending, fresh.Status,
		"approving a returned copy must not approve the stored entry")
	require.Empty(t, fresh.Note)
}

// TestMemoryStore_GetByEmailDoesNotAliasStoredEntry covers the second lookup
// path, which is the one the public sign-up flow uses.
func TestMemoryStore_GetByEmailDoesNotAliasStoredEntry(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	require.NoError(t, s.CreateEntry(ctx, newEntry(appID, "user@example.com")))

	got, err := s.GetEntryByEmail(ctx, appID, "user@example.com")
	require.NoError(t, err)
	got.Status = StatusApproved

	fresh, err := s.GetEntryByEmail(ctx, appID, "user@example.com")
	require.NoError(t, err)
	require.Equal(t, StatusPending, fresh.Status,
		"approving a copy from GetEntryByEmail must not approve the stored entry")
}

// TestMemoryStore_CreateDoesNotAliasCallerEntry is the mirror image.
func TestMemoryStore_CreateDoesNotAliasCallerEntry(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	e := newEntry(appID, "user@example.com")
	require.NoError(t, s.CreateEntry(ctx, e))

	e.Status = StatusApproved
	e.Note = "self-approved"

	got, err := s.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, StatusPending, got.Status,
		"mutating the caller's entry after CreateEntry must not affect the store")
	require.Empty(t, got.Note)
}

// TestMemoryStore_ListDoesNotAliasStoredEntries covers the many-pointer path.
func TestMemoryStore_ListDoesNotAliasStoredEntries(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	for i := 0; i < 3; i++ {
		require.NoError(t, s.CreateEntry(ctx, newEntry(appID, fmt.Sprintf("user%d@example.com", i))))
	}

	first, err := s.ListEntries(ctx, &WaitlistQuery{AppID: appID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, first.Entries, 3)
	for _, e := range first.Entries {
		e.Status = StatusApproved
	}

	second, err := s.ListEntries(ctx, &WaitlistQuery{AppID: appID, Limit: 10})
	require.NoError(t, err)
	for _, e := range second.Entries {
		require.Equal(t, StatusPending, e.Status, "mutating a listed entry must not affect the store")
	}
}

// TestMemoryStore_ConcurrentCreateAndCount drives creation against the
// counting path, which scans the whole slice under the read lock while
// creators append to it under the write lock.
func TestMemoryStore_ConcurrentCreateAndCount(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	const perWorker = 40

	var wg sync.WaitGroup
	for w := 0; w < waitlistHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				if err := s.CreateEntry(ctx, newEntry(appID, fmt.Sprintf("w%d-u%d@example.com", w, i))); err != nil {
					t.Errorf("CreateEntry: %v", err)
					return
				}
			}
		}(w)
	}
	for w := 0; w < waitlistHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				if _, _, _, err := s.CountByStatus(ctx, appID); err != nil {
					t.Errorf("CountByStatus: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	pending, _, _, err := s.CountByStatus(ctx, appID)
	require.NoError(t, err)
	require.Equal(t, waitlistHammerWorkers*perWorker, pending,
		"entries were lost: every create must land under the write lock")
}
