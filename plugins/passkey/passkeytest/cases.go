package passkeytest

import (
	"bytes"
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCredentialCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	require.NoError(t, f.Store.CreateCredential(ctx, c))

	got, err := f.Store.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, c.UserID, got.UserID)
	assert.Equal(t, c.AppID, got.AppID)
	assert.Equal(t, c.AttestationType, got.AttestationType)
	assert.Equal(t, c.DisplayName, got.DisplayName)
	assert.Equal(t, c.SignCount, got.SignCount)
}

func testCredentialNotFound(t *testing.T, f Fixture) {
	_, err := f.Store.GetCredential(context.Background(), uniqueBytes())
	require.Error(t, err, "an unregistered credential id must not resolve")
}

// testBinaryFieldsRoundTrip is the one that catches a backend storing raw
// bytes as text. A credential id containing a NUL, a 0xFF and an invalid
// UTF-8 pair survives a byte-safe column and is silently mangled by anything
// that round-trips it through a string.
func testBinaryFieldsRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	require.NoError(t, f.Store.CreateCredential(ctx, c))

	got, err := f.Store.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(c.CredentialID, got.CredentialID),
		"credential id changed in storage: wrote %x, read %x", c.CredentialID, got.CredentialID)
	assert.True(t, bytes.Equal(c.PublicKey, got.PublicKey),
		"public key changed in storage: wrote %x, read %x", c.PublicKey, got.PublicKey)
	assert.True(t, bytes.Equal(c.AAGUID, got.AAGUID),
		"AAGUID changed in storage: wrote %x, read %x", c.AAGUID, got.AAGUID)
}

func testTransportRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	c.Transport = []string{"usb", "nfc", "ble"}
	require.NoError(t, f.Store.CreateCredential(ctx, c))

	got, err := f.Store.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	assert.Equal(t, c.Transport, got.Transport, "transport hints must survive with order intact")
}

// testEmptyTransportRoundTrip checks that a credential registered with no
// transport hints reads back with none, rather than picking up a phantom
// empty string as its only entry. Splitting a comma-separated column is the
// obvious way to get that wrong.
//
// It deliberately asserts empty rather than non-nil. Every backend stores
// transports as one comma-separated string and decodes a blank column to a
// nil slice, so all four agree and there is no drift to catch here. That nil
// still serialises as JSON null rather than [], which is the same shape as
// the session-roles issue, but changing it is a contract decision for the API
// rather than something this suite should force.
func testEmptyTransportRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	c.Transport = nil
	require.NoError(t, f.Store.CreateCredential(ctx, c))

	got, err := f.Store.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	assert.Empty(t, got.Transport,
		"no transport hints must read back as none, not as one empty entry")
}

func testListIsScopedToUser(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	mine := newCredential(f, f.UserID)
	require.NoError(t, f.Store.CreateCredential(ctx, mine))
	theirs := newCredential(f, f.OtherUserID)
	require.NoError(t, f.Store.CreateCredential(ctx, theirs))

	got, err := f.Store.ListUserCredentials(ctx, f.UserID)
	require.NoError(t, err)

	var found bool
	for _, c := range got {
		assert.Equal(t, f.UserID, c.UserID, "listing leaked another user's credential")
		if bytes.Equal(c.CredentialID, mine.CredentialID) {
			found = true
		}
	}
	assert.True(t, found, "listing omitted a credential belonging to the queried user")
}

func testUpdateSignCount(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	require.NoError(t, f.Store.CreateCredential(ctx, c))

	require.NoError(t, f.Store.UpdateSignCount(ctx, c.CredentialID, 42))
	got, err := f.Store.GetCredential(ctx, c.CredentialID)
	require.NoError(t, err)
	assert.Equal(t, uint32(42), got.SignCount,
		"the sign counter is what detects a cloned authenticator; it has to persist")
}

// testSignCountHoldsFullUint32 pushes the counter to the top of its range.
// WebAuthn defines it as a uint32, but the SQL backends store it in a signed
// column, so anything above 2^31-1 is where a narrowing bug shows up.
func testSignCountHoldsFullUint32(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	require.NoError(t, f.Store.CreateCredential(ctx, c))

	for _, want := range []uint32{math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		require.NoError(t, f.Store.UpdateSignCount(ctx, c.CredentialID, want))
		got, err := f.Store.GetCredential(ctx, c.CredentialID)
		require.NoError(t, err)
		assert.Equal(t, want, got.SignCount, "sign count %d did not survive the round trip", want)
	}
}

func testDeleteCredential(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newCredential(f, f.UserID)
	require.NoError(t, f.Store.CreateCredential(ctx, c))
	require.NoError(t, f.Store.DeleteCredential(ctx, c.CredentialID))

	_, err := f.Store.GetCredential(ctx, c.CredentialID)
	assert.Error(t, err, "a deleted credential must not resolve; a passkey the user revoked has to stop working")

	got, err := f.Store.ListUserCredentials(ctx, f.UserID)
	require.NoError(t, err)
	for _, existing := range got {
		assert.False(t, bytes.Equal(existing.CredentialID, c.CredentialID),
			"a deleted credential still appears in the user's listing")
	}
}
