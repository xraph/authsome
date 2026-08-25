package mfa

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

// MemoryStore locks every method and was never called concurrently by this
// package's tests, so `go test -race` here observed nothing.
//
// Underneath the lock, the store mutates records in place. ConsumeRecoveryCode
// writes Used and UsedAt on a stored *RecoveryCode, and GetRecoveryCodes used
// to return those same pointers. Reading Used off what you were handed was a
// read of a field another goroutine could be writing, unordered against it,
// and Used is the field that makes a recovery code single-use.
//
// One thing these tests deliberately do not claim. Copying fixes the data
// race, not the check-then-consume window above it: a caller that lists codes,
// finds one unused, and only then consumes it still races another caller doing
// the same, because no lock spans both steps. That is a property of the Store
// interface, not of this implementation, so it is documented rather than
// asserted. What is asserted is that consumption itself never double-counts.

const (
	mfaHammerWorkers = 4
	mfaHammerBudget  = 150 * time.Millisecond
)

func newRecoveryCode(userID id.UserID, n int) *RecoveryCode {
	return &RecoveryCode{
		ID:        id.NewRecoveryCodeID(),
		UserID:    userID,
		CodeHash:  fmt.Sprintf("hash-%d", n),
		CreatedAt: time.Now(),
	}
}

func newEnrollment(userID id.UserID, method string) *Enrollment {
	return &Enrollment{
		ID: id.NewMFAID(), UserID: userID, Method: method,
		Secret: "SECRET", CreatedAt: time.Now(),
	}
}

// TestMemoryStore_ConcurrentConsumeAndRead runs consumption against reads,
// with the readers writing to what they are handed.
func TestMemoryStore_ConcurrentConsumeAndRead(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID := id.NewUserID()

	codes := make([]*RecoveryCode, 16)
	for i := range codes {
		codes[i] = newRecoveryCode(userID, i)
	}
	require.NoError(t, s.CreateRecoveryCodes(ctx, codes))

	deadline := time.Now().Add(mfaHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < mfaHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				_ = s.ConsumeRecoveryCode(ctx, codes[(w+i)%len(codes)].ID)
				ops.Add(1)
			}
		}(w)
	}

	for w := 0; w < mfaHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				got, err := s.GetRecoveryCodes(ctx, userID)
				if err != nil {
					t.Errorf("GetRecoveryCodes: %v", err)
					return
				}
				for _, c := range got {
					// Un-use it, on our copy. This must not put a spent code
					// back into circulation for anyone else.
					c.Used = false
					c.UsedAt = nil
					c.CodeHash = "tampered"
				}
				ops.Add(1)
			}
		}()
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(500),
		"the hammer did no meaningful work; it is not exercising the store concurrently")

	// Every code was consumed many times over, so all of them must read as
	// used, and none may carry a reader's scribble.
	final, err := s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	require.Len(t, final, len(codes))
	for _, c := range final {
		require.True(t, c.Used, "a consumed recovery code was returned to circulation by a reader's write")
		require.NotEqual(t, "tampered", c.CodeHash, "a reader's write reached the stored code hash")
	}
}

// TestMemoryStore_GetRecoveryCodesDoesNotAliasStored pins the read side on
// its own. Marking a code unused on the returned value must not make it
// spendable again in the store.
func TestMemoryStore_GetRecoveryCodesDoesNotAliasStored(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID := id.NewUserID()

	code := newRecoveryCode(userID, 1)
	require.NoError(t, s.CreateRecoveryCodes(ctx, []*RecoveryCode{code}))
	require.NoError(t, s.ConsumeRecoveryCode(ctx, code.ID))

	got, err := s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.True(t, got[0].Used)
	got[0].Used = false
	got[0].UsedAt = nil

	fresh, err := s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	require.Len(t, fresh, 1)
	require.True(t, fresh[0].Used, "un-using a returned copy must not make the code spendable again")
	require.NotNil(t, fresh[0].UsedAt)
}

// TestMemoryStore_CreateRecoveryCodesDoesNotAliasCaller is the mirror image.
// The caller still holds the slice it handed over, and writing to those
// records must not reach the store.
func TestMemoryStore_CreateRecoveryCodesDoesNotAliasCaller(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID := id.NewUserID()

	code := newRecoveryCode(userID, 1)
	require.NoError(t, s.CreateRecoveryCodes(ctx, []*RecoveryCode{code}))

	code.Used = true
	code.CodeHash = "tampered"

	got, err := s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.False(t, got[0].Used, "mutating the caller's code after Create must not affect the store")
	require.Equal(t, "hash-1", got[0].CodeHash)
}

// TestMemoryStore_ConcurrentEnrollmentReadsAndWrites covers the other map,
// including the in-place-free update path, with readers scribbling on what
// they receive. Secret is the field that matters: it is the TOTP seed.
func TestMemoryStore_ConcurrentEnrollmentReadsAndWrites(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	users := make([]id.UserID, 6)
	enrollments := make([]*Enrollment, len(users))
	for i := range users {
		users[i] = id.NewUserID()
		enrollments[i] = newEnrollment(users[i], "totp")
		require.NoError(t, s.CreateEnrollment(ctx, enrollments[i]))
	}

	deadline := time.Now().Add(mfaHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < mfaHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				idx := (w + i) % len(users)
				if got, err := s.GetEnrollment(ctx, users[idx], "totp"); err == nil {
					got.Secret = "tampered"
					got.Verified = true
				}
				if _, err := s.ListEnrollments(ctx, users[idx]); err != nil {
					t.Errorf("ListEnrollments: %v", err)
					return
				}
				if _, err := s.GetEnrollmentByID(ctx, enrollments[idx].ID); err != nil {
					t.Errorf("GetEnrollmentByID: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(500), "the hammer did no meaningful work")

	for i, u := range users {
		got, err := s.GetEnrollment(ctx, u, "totp")
		require.NoError(t, err)
		require.Equal(t, "SECRET", got.Secret, "a reader's write reached enrollment %d's stored TOTP secret", i)
		require.False(t, got.Verified, "a reader's write flipped enrollment %d to verified", i)
	}
}

// TestMemoryStore_GetEnrollmentDoesNotAliasStored is the sequential version,
// so a failure points at the copy rather than at the hammer.
func TestMemoryStore_GetEnrollmentDoesNotAliasStored(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	userID := id.NewUserID()
	require.NoError(t, s.CreateEnrollment(ctx, newEnrollment(userID, "totp")))

	got, err := s.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	got.Secret = "tampered"
	got.Verified = true

	fresh, err := s.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	require.Equal(t, "SECRET", fresh.Secret, "mutating a returned enrollment must not rewrite the stored TOTP secret")
	require.False(t, fresh.Verified)
}
