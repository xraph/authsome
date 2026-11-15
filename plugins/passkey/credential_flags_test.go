package passkey_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/passkey/passkeytest"
)

const (
	backupEligible = 0x08
	userVerified   = 0x04
)

// A synced passkey (iCloud Keychain, Google Password Manager) is backup
// eligible. go-webauthn refuses a sign-in whose backup-eligible bit differs
// from the stored credential's, so the flags have to be kept from
// registration on. It signs in, again and again, and with two-step on its
// user verification stands in for the code.
func TestDiscoverableLogin_SyncedPasskeySignsIn(t *testing.T) {
	f := newTwoStepFixture(t)
	device := passkeytest.NewAuthenticator(testOrigin, "localhost")
	device.Synced = true
	f.register(t, device)

	creds, err := f.store.ListUserCredentials(context.Background(), f.u.ID)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	require.NotNil(t, creds[0].Flags, "registration stores the flags")
	assert.NotZero(t, *creds[0].Flags&backupEligible, "stored flags %08b", *creds[0].Flags)

	for i := range 2 {
		rec := f.login(t, device)
		require.Equal(t, http.StatusOK, rec.Code, "sign-in %d: body=%s", i+1, rec.Body.String())
	}
}

// A passkey stored before authsome kept flags has none. Its first sign-in
// takes the backup bits from its own assertion and stores what go-webauthn
// settled on; a user-verifying assertion still needs no code.
func TestDiscoverableLogin_PasskeyStoredWithoutFlags(t *testing.T) {
	f := newTwoStepFixture(t)
	device := passkeytest.NewAuthenticator(testOrigin, "localhost")
	device.Synced = true
	f.register(t, device)

	ctx := context.Background()
	creds, err := f.store.ListUserCredentials(ctx, f.u.ID)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	legacy := creds[0]
	legacy.Flags = nil
	require.NoError(t, f.store.DeleteCredential(ctx, legacy.CredentialID))
	require.NoError(t, f.store.CreateCredential(ctx, legacy))

	rec := f.login(t, device)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	stored, err := f.store.GetCredential(ctx, legacy.CredentialID)
	require.NoError(t, err)
	require.NotNil(t, stored.Flags, "the first sign-in stores the flags")
	assert.NotZero(t, *stored.Flags&backupEligible, "stored flags %08b", *stored.Flags)

	rec = f.login(t, device)
	require.Equal(t, http.StatusOK, rec.Code, "second sign-in: body=%s", rec.Body.String())
}

// What a sign-in counts as verified comes from that sign-in's assertion,
// never from flags stored earlier: a passkey registered with user
// verification, then used with presence only, is asked for the code.
func TestDiscoverableLogin_StoredUVDoesNotStandInForTheAssertion(t *testing.T) {
	f := newTwoStepFixture(t)
	device := passkeytest.NewAuthenticator(testOrigin, "localhost")
	f.register(t, device)

	creds, err := f.store.ListUserCredentials(context.Background(), f.u.ID)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	require.NotNil(t, creds[0].Flags)
	require.NotZero(t, *creds[0].Flags&userVerified, "registration verified the user")

	device.NoUserVerification = true
	rec := f.login(t, device)
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}
