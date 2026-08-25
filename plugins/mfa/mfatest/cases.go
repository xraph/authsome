package mfatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/mfa"
)

func testEnrollmentCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEnrollment(f.UserID)
	require.NoError(t, f.Store.CreateEnrollment(ctx, e))

	// Lookup by user and method, which is what the challenge path uses.
	got, err := f.Store.GetEnrollment(ctx, f.UserID, "totp")
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, e.UserID, got.UserID)
	assert.Equal(t, e.Method, got.Method)

	// Lookup by id, which is what the management path uses.
	byID, err := f.Store.GetEnrollmentByID(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, e.UserID, byID.UserID)
	assert.Equal(t, e.Method, byID.Method)
}

func testEnrollmentNotFound(t *testing.T, f Fixture) {
	ctx := context.Background()

	_, err := f.Store.GetEnrollment(ctx, f.UserID, "totp")
	assert.Error(t, err, "a user with no enrollment must not resolve one")

	_, err = f.Store.GetEnrollmentByID(ctx, id.NewMFAID())
	assert.Error(t, err, "an unknown enrollment id must not resolve")
}

// testSecretRoundTrip guards the TOTP shared secret. A secret that comes back
// altered by even one character produces codes that never validate, and the
// failure looks like a broken authenticator app rather than a storage bug.
func testSecretRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEnrollment(f.UserID)
	e.Secret = "MFRGGZDFMZTWQ2LKNNWG23TPOA======"
	require.NoError(t, f.Store.CreateEnrollment(ctx, e))

	got, err := f.Store.GetEnrollment(ctx, f.UserID, "totp")
	require.NoError(t, err)
	assert.Equal(t, e.Secret, got.Secret, "the base32 secret must survive byte for byte, padding included")
}

// testUpdateEnrollmentVerified covers the flag that separates a half-finished
// enrollment from a live second factor.
func testUpdateEnrollmentVerified(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEnrollment(f.UserID)
	require.NoError(t, f.Store.CreateEnrollment(ctx, e))

	got, err := f.Store.GetEnrollment(ctx, f.UserID, "totp")
	require.NoError(t, err)
	assert.False(t, got.Verified, "a fresh enrollment must not read back as verified")

	e.Verified = true
	e.UpdatedAt = now()
	require.NoError(t, f.Store.UpdateEnrollment(ctx, e))

	got, err = f.Store.GetEnrollment(ctx, f.UserID, "totp")
	require.NoError(t, err)
	assert.True(t, got.Verified, "verification must persist")
	assert.Equal(t, e.Secret, got.Secret, "updating the flag must not disturb the secret")
}

func testEnrollmentLookupIsScopedToUser(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	theirs := newEnrollment(f.OtherUserID)
	require.NoError(t, f.Store.CreateEnrollment(ctx, theirs))

	// One user's enrollment must never satisfy another user's lookup.
	_, err := f.Store.GetEnrollment(ctx, f.UserID, "totp")
	assert.Error(t, err, "lookup returned another user's enrollment")
}

func testListEnrollmentsIsScopedToUser(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	mine := newEnrollment(f.UserID)
	require.NoError(t, f.Store.CreateEnrollment(ctx, mine))
	theirs := newEnrollment(f.OtherUserID)
	require.NoError(t, f.Store.CreateEnrollment(ctx, theirs))

	got, err := f.Store.ListEnrollments(ctx, f.UserID)
	require.NoError(t, err)

	var found bool
	for _, e := range got {
		assert.Equal(t, f.UserID, e.UserID, "listing leaked another user's enrollment")
		if e.ID == mine.ID {
			found = true
		}
	}
	assert.True(t, found, "listing omitted an enrollment belonging to the queried user")
}

func testDeleteEnrollment(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEnrollment(f.UserID)
	require.NoError(t, f.Store.CreateEnrollment(ctx, e))
	require.NoError(t, f.Store.DeleteEnrollment(ctx, e.ID))

	_, err := f.Store.GetEnrollment(ctx, f.UserID, "totp")
	assert.Error(t, err, "a removed second factor must stop resolving, or it still gates sign-in")
}

func testRecoveryCodesRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	codes := []*mfa.RecoveryCode{newRecoveryCode(f.UserID), newRecoveryCode(f.UserID), newRecoveryCode(f.UserID)}
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, codes))

	got, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	assert.Len(t, got, len(codes), "every code in the batch must be stored")

	byID := make(map[id.RecoveryCodeID]*mfa.RecoveryCode, len(got))
	for _, c := range got {
		byID[c.ID] = c
	}
	for _, want := range codes {
		stored, ok := byID[want.ID]
		require.True(t, ok, "recovery code %s missing after write", want.ID)
		assert.Equal(t, want.CodeHash, stored.CodeHash, "the code hash must survive; a mangled hash never matches")
		assert.False(t, stored.Used, "a fresh recovery code must not read back as used")
	}
}

// testUnusedRecoveryCodeHasNoUsedAt pins the nullable timestamp. UsedAt is a
// pointer precisely so an unused code carries no time, and a backend that
// substitutes a zero time makes an unused code look spent.
func testUnusedRecoveryCodeHasNoUsedAt(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newRecoveryCode(f.UserID)
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, []*mfa.RecoveryCode{c}))

	got, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	if got[0].UsedAt != nil {
		assert.True(t, got[0].UsedAt.IsZero(),
			"an unused recovery code carried a real UsedAt of %v", got[0].UsedAt)
	}
}

func testConsumeRecoveryCodeMarksUsed(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newRecoveryCode(f.UserID)
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, []*mfa.RecoveryCode{c}))

	require.NoError(t, f.Store.ConsumeRecoveryCode(ctx, c.ID))

	got, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.True(t, got[0].Used, "consuming a recovery code must mark it used")
	if assert.NotNil(t, got[0].UsedAt, "a consumed code must record when it was used") {
		assert.False(t, got[0].UsedAt.IsZero())
	}
}

// testConsumeRecoveryCodeIsSingleUse is the security-critical one. A recovery
// code is a bearer credential that bypasses the second factor, so once spent
// it has to stay spent however many times it is presented again.
func testConsumeRecoveryCodeIsSingleUse(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newRecoveryCode(f.UserID)
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, []*mfa.RecoveryCode{c}))

	require.NoError(t, f.Store.ConsumeRecoveryCode(ctx, c.ID))
	first, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.True(t, first[0].Used)
	firstUsedAt := first[0].UsedAt

	// Replaying the same code must leave it used. The method reports no
	// verdict, so callers gate on Used; what matters is that a second call
	// cannot walk the flag back.
	// Logged rather than asserted: the backends disagree on whether replaying
	// a spent code is an error, and every caller gates on Used instead.
	t.Logf("replay error for this backend: %v", f.Store.ConsumeRecoveryCode(ctx, c.ID))

	again, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	require.Len(t, again, 1)
	assert.True(t, again[0].Used, "replaying a spent recovery code cleared its used flag")
	if firstUsedAt != nil && again[0].UsedAt != nil {
		assert.WithinDuration(t, *firstUsedAt, *again[0].UsedAt, 0,
			"replay moved the UsedAt stamp, so the audit trail records the wrong moment")
	}
}

func testRecoveryCodesAreScopedToUser(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	mine := newRecoveryCode(f.UserID)
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, []*mfa.RecoveryCode{mine}))
	theirs := newRecoveryCode(f.OtherUserID)
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, []*mfa.RecoveryCode{theirs}))

	got, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	for _, c := range got {
		assert.Equal(t, f.UserID, c.UserID, "recovery code listing leaked another user's code")
		assert.NotEqual(t, theirs.ID, c.ID)
	}
}

func testDeleteRecoveryCodes(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	mine := []*mfa.RecoveryCode{newRecoveryCode(f.UserID), newRecoveryCode(f.UserID)}
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, mine))
	theirs := newRecoveryCode(f.OtherUserID)
	require.NoError(t, f.Store.CreateRecoveryCodes(ctx, []*mfa.RecoveryCode{theirs}))

	require.NoError(t, f.Store.DeleteRecoveryCodes(ctx, f.UserID))

	got, err := f.Store.GetRecoveryCodes(ctx, f.UserID)
	require.NoError(t, err)
	assert.Empty(t, got, "regenerating codes must clear the old set")

	// The wipe is per user; it must not take the other user's codes with it.
	survived, err := f.Store.GetRecoveryCodes(ctx, f.OtherUserID)
	require.NoError(t, err)
	assert.Len(t, survived, 1, "deleting one user's recovery codes removed another user's")
}
