package mfatest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/mfa"
)

// suffixer gives each test its own token hashes, so a backend reused between
// runs (a local mongo, say) never finds a row an earlier run left behind.
func suffixer() func(string) string {
	suffix := id.NewTrustedDeviceID().String()
	return func(s string) string { return s + "-" + suffix }
}

func newTrustedDevice(userID id.UserID, tokenHash string) *mfa.TrustedDevice {
	n := now()
	return &mfa.TrustedDevice{
		ID:         id.NewTrustedDeviceID(),
		UserID:     userID,
		TokenHash:  tokenHash,
		Label:      "Firefox on Linux",
		UserAgent:  "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
		IPAddress:  "203.0.113.7",
		CreatedAt:  n,
		LastUsedAt: n,
		ExpiresAt:  n.Add(30 * 24 * time.Hour),
	}
}

// testTrustedDeviceRoundTrip covers the lookup the sign-in gate makes: by the
// token's hash, with every field the settings page shows intact.
func testTrustedDeviceRoundTrip(t *testing.T, f Fixture) {
	h := suffixer()
	ctx := context.Background()
	d := newTrustedDevice(f.UserID, h("hash-roundtrip"))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, d))

	got, err := f.Store.GetTrustedDeviceByTokenHash(ctx, h("hash-roundtrip"))
	require.NoError(t, err)
	assert.Equal(t, d.ID, got.ID)
	assert.Equal(t, d.UserID, got.UserID)
	assert.Equal(t, d.TokenHash, got.TokenHash)
	assert.Equal(t, d.Label, got.Label)
	assert.Equal(t, d.UserAgent, got.UserAgent)
	assert.Equal(t, d.IPAddress, got.IPAddress)
	assert.True(t, d.CreatedAt.Equal(got.CreatedAt), "created_at: want %v, got %v", d.CreatedAt, got.CreatedAt)
	assert.True(t, d.ExpiresAt.Equal(got.ExpiresAt), "expires_at: want %v, got %v", d.ExpiresAt, got.ExpiresAt)
}

// testTrustedDeviceUnknownToken: a token nobody holds must not resolve, and
// must say so with the sentinel the gate tells apart from a store failure.
func testTrustedDeviceUnknownToken(t *testing.T, f Fixture) {
	h := suffixer()
	_, err := f.Store.GetTrustedDeviceByTokenHash(context.Background(), h("no-such-hash"))
	assert.ErrorIs(t, err, mfa.ErrTrustedDeviceNotFound)
}

func testTouchTrustedDevice(t *testing.T, f Fixture) {
	h := suffixer()
	ctx := context.Background()
	d := newTrustedDevice(f.UserID, h("hash-touch"))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, d))

	later := d.CreatedAt.Add(2 * time.Hour)
	require.NoError(t, f.Store.TouchTrustedDevice(ctx, d.ID, later))

	got, err := f.Store.GetTrustedDeviceByTokenHash(ctx, h("hash-touch"))
	require.NoError(t, err)
	assert.True(t, later.Equal(got.LastUsedAt), "last_used_at: want %v, got %v", later, got.LastUsedAt)
	assert.True(t, d.CreatedAt.Equal(got.CreatedAt), "touching must not move created_at")
}

func testListTrustedDevicesIsScopedToUser(t *testing.T, f Fixture) {
	h := suffixer()
	ctx := context.Background()
	older := newTrustedDevice(f.UserID, h("hash-mine-1"))
	newer := newTrustedDevice(f.UserID, h("hash-mine-2"))
	newer.CreatedAt = older.CreatedAt.Add(time.Minute)
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, older))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, newer))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, newTrustedDevice(f.OtherUserID, h("hash-theirs"))))

	mine, err := f.Store.ListTrustedDevices(ctx, f.UserID)
	require.NoError(t, err)
	require.Len(t, mine, 2, "only the user's own devices are listed")
	assert.Equal(t, newer.ID, mine[0].ID, "newest first")
	assert.Equal(t, older.ID, mine[1].ID)
}

// testDeleteTrustedDeviceIsScopedToUser: removing by id names the owner, so a
// guessed id cannot strip someone else's trust (or keep it alive).
func testDeleteTrustedDeviceIsScopedToUser(t *testing.T, f Fixture) {
	h := suffixer()
	ctx := context.Background()
	mine := newTrustedDevice(f.UserID, h("hash-del-mine"))
	theirs := newTrustedDevice(f.OtherUserID, h("hash-del-theirs"))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, mine))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, theirs))

	require.NoError(t, f.Store.DeleteTrustedDevice(ctx, f.UserID, theirs.ID))
	_, err := f.Store.GetTrustedDeviceByTokenHash(ctx, h("hash-del-theirs"))
	require.NoError(t, err, "another user's device survives a delete in my name")

	require.NoError(t, f.Store.DeleteTrustedDevice(ctx, f.UserID, mine.ID))
	_, err = f.Store.GetTrustedDeviceByTokenHash(ctx, h("hash-del-mine"))
	assert.ErrorIs(t, err, mfa.ErrTrustedDeviceNotFound)
}

func testDeleteTrustedDevices(t *testing.T, f Fixture) {
	h := suffixer()
	ctx := context.Background()
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, newTrustedDevice(f.UserID, h("hash-all-1"))))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, newTrustedDevice(f.UserID, h("hash-all-2"))))
	require.NoError(t, f.Store.CreateTrustedDevice(ctx, newTrustedDevice(f.OtherUserID, h("hash-all-theirs"))))

	require.NoError(t, f.Store.DeleteTrustedDevices(ctx, f.UserID))

	mine, err := f.Store.ListTrustedDevices(ctx, f.UserID)
	require.NoError(t, err)
	assert.Empty(t, mine)
	theirs, err := f.Store.ListTrustedDevices(ctx, f.OtherUserID)
	require.NoError(t, err)
	assert.Len(t, theirs, 1, "another user's devices stay")
}
