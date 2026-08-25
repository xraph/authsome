package passkey

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

// MemoryStore locks correctly and was never once called from two goroutines
// by this package's tests, so `go test -race` here proved nothing about it.
//
// The interesting part is what the lock does not cover. UpdateSignCount
// writes SignCount and UpdatedAt on a stored *Credential in place, while
// GetCredential and ListUserCredentials used to return that same pointer. A
// verifier reading SignCount off the value it was handed was reading a field
// another goroutine could be writing, unordered against it.
//
// SignCount is WebAuthn's clone-detection signal: an authenticator's counter
// is supposed to move strictly upward, and a verifier compares the stored
// value against the one in the assertion. A torn or stale read there is a
// check that silently stops checking. The store copies now, and these tests
// are what would catch that copy going away.

const (
	passkeyHammerWorkers = 4
	passkeyHammerBudget  = 150 * time.Millisecond
)

func newCredential(userID id.UserID, appID id.AppID, n int) *Credential {
	return &Credential{
		ID:              id.NewPasskeyID(),
		UserID:          userID,
		AppID:           appID,
		CredentialID:    []byte(fmt.Sprintf("cred-%d", n)),
		PublicKey:       []byte("public-key-material"),
		AAGUID:          []byte("aaguid"),
		Transport:       []string{"internal"},
		AttestationType: "none",
		DisplayName:     "key",
		SignCount:       1,
	}
}

// TestMemoryStore_ConcurrentSignCountUpdatesAndReads runs the real shape of a
// WebAuthn verification loop: many goroutines reading a credential while
// others advance its sign counter, with the readers also writing to what they
// were handed.
func TestMemoryStore_ConcurrentSignCountUpdatesAndReads(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID, appID := id.NewUserID(), id.NewAppID()

	creds := make([]*Credential, 6)
	for i := range creds {
		creds[i] = newCredential(userID, appID, i)
		require.NoError(t, s.CreateCredential(ctx, creds[i]))
	}

	deadline := time.Now().Add(passkeyHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64
	var counter atomic.Uint32

	for w := 0; w < passkeyHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				c := creds[(w+i)%len(creds)]
				if err := s.UpdateSignCount(ctx, c.CredentialID, counter.Add(1)); err != nil {
					t.Errorf("UpdateSignCount: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}

	for w := 0; w < passkeyHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				c := creds[(w+i)%len(creds)]
				if got, err := s.GetCredential(ctx, c.CredentialID); err == nil {
					// A verifier that scribbles on what it was handed must not
					// be able to reach the store's own record.
					got.SignCount = 0
					got.PublicKey[0] = 'X'
					got.Transport[0] = "tampered"
				}
				if _, err := s.ListUserCredentials(ctx, userID); err != nil {
					t.Errorf("ListUserCredentials: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(500),
		"the hammer did no meaningful work; it is not exercising the store concurrently")

	// The stored key material must be exactly what was put in.
	stored, err := s.ListUserCredentials(ctx, userID)
	require.NoError(t, err)
	require.Len(t, stored, len(creds))
	for _, c := range stored {
		require.Equal(t, []byte("public-key-material"), c.PublicKey,
			"a reader's write reached the stored public key")
		require.Equal(t, []string{"internal"}, c.Transport,
			"a reader's write reached the stored transport list")
	}
}

// TestMemoryStore_GetDoesNotAliasStoredCredential pins the read side, and
// specifically the byte slices. A shallow struct copy would pass every
// scalar assertion here and still share PublicKey, CredentialID and AAGUID
// with the store.
func TestMemoryStore_GetDoesNotAliasStoredCredential(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	c := newCredential(id.NewUserID(), id.NewAppID(), 1)
	require.NoError(t, s.CreateCredential(ctx, c))

	got, err := s.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	got.SignCount = 9999
	got.PublicKey[0] = 'X'
	got.AAGUID[0] = 'X'
	got.Transport[0] = "tampered"

	fresh, err := s.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	require.Equal(t, uint32(1), fresh.SignCount, "writing SignCount on a returned copy must not reach the store")
	require.Equal(t, []byte("public-key-material"), fresh.PublicKey, "PublicKey is a slice; a shallow copy would still alias it")
	require.Equal(t, []byte("aaguid"), fresh.AAGUID)
	require.Equal(t, []string{"internal"}, fresh.Transport)
}

// TestMemoryStore_CreateDoesNotAliasCallerCredential is the mirror image. The
// caller still holds the *Credential it passed in, and writing to it must not
// rewrite what the store kept.
func TestMemoryStore_CreateDoesNotAliasCallerCredential(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	c := newCredential(id.NewUserID(), id.NewAppID(), 1)
	require.NoError(t, s.CreateCredential(ctx, c))

	c.SignCount = 9999
	c.PublicKey[0] = 'X'
	c.Transport[0] = "tampered"

	got, err := s.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	require.Equal(t, uint32(1), got.SignCount, "mutating the caller's credential after Create must not affect the store")
	require.Equal(t, []byte("public-key-material"), got.PublicKey)
	require.Equal(t, []string{"internal"}, got.Transport)
}

// TestMemoryStore_SignCountAdvancesMonotonically is the property a verifier
// actually depends on. Writers only ever hand UpdateSignCount a value from a
// strictly increasing sequence, so whatever the interleaving, the stored
// counter must never be seen to move backwards by a reader that observed a
// higher one earlier.
func TestMemoryStore_SignCountAdvancesMonotonically(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	c := newCredential(id.NewUserID(), id.NewAppID(), 1)
	require.NoError(t, s.CreateCredential(ctx, c))

	const updates = 200
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := uint32(1); i <= updates; i++ {
			if err := s.UpdateSignCount(ctx, c.CredentialID, i); err != nil {
				t.Errorf("UpdateSignCount: %v", err)
				return
			}
		}
	}()

	for w := 0; w < passkeyHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var highest uint32
			for i := 0; i < updates; i++ {
				got, err := s.GetCredential(ctx, c.CredentialID)
				if err != nil {
					t.Errorf("GetCredential: %v", err)
					return
				}
				if got.SignCount < highest {
					t.Errorf("sign counter went backwards: saw %d after %d", got.SignCount, highest)
					return
				}
				highest = got.SignCount
				// Scribble on the copy; it must not affect what anyone else sees.
				got.SignCount = 0
			}
		}()
	}
	wg.Wait()

	final, err := s.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	require.Equal(t, uint32(updates), final.SignCount)
}

// TestMemoryStore_ConcurrentCreateAndDelete drives the map's own write paths
// against each other, which nothing else here covers.
func TestMemoryStore_ConcurrentCreateAndDelete(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID, appID := id.NewUserID(), id.NewAppID()

	deadline := time.Now().Add(passkeyHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < passkeyHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				c := newCredential(userID, appID, (w*100+i)%16)
				if err := s.CreateCredential(ctx, c); err != nil {
					t.Errorf("CreateCredential: %v", err)
					return
				}
				_, _ = s.GetCredential(ctx, c.CredentialID)
				if i%4 == 0 {
					_ = s.DeleteCredential(ctx, c.CredentialID)
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(500), "the hammer did no meaningful work")
}
