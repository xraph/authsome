package socialtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConnectionCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))

	got, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, c.UserID, got.UserID)
	assert.Equal(t, c.AppID, got.AppID)
	assert.Equal(t, c.Provider, got.Provider)
	assert.Equal(t, c.ProviderUserID, got.ProviderUserID)
	assert.Equal(t, c.Email, got.Email)
}

func testConnectionNotFound(t *testing.T, f Fixture) {
	_, err := f.Store.GetOAuthConnection(context.Background(), "google", unique("absent"))
	assert.Error(t, err, "a provider identity nobody has connected must not resolve")
}

// testTokensRoundTrip is the one that matters most here. These tokens are
// what let the app act at the provider on the user's behalf, and neither is
// ever echoed back through JSON, so a backend mangling one produces a
// connection that looks healthy and fails at the provider.
func testTokensRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	c.AccessToken = "ya29.a0AfH6SM-_x/+=" + unique("at")
	c.RefreshToken = "1//0eXaMpLe-_x/+=" + unique("rt")
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))

	got, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	require.NoError(t, err)
	assert.Equal(t, c.AccessToken, got.AccessToken, "the access token must survive byte for byte")
	assert.Equal(t, c.RefreshToken, got.RefreshToken,
		"the refresh token is the long-lived one; losing it means the user has to reconnect")
}

// testExpiryRoundTrip guards the moment the access token stops working. It is
// what decides whether a refresh happens, so a value that shifts by a
// timezone offset either refreshes constantly or too late.
func testExpiryRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	c.ExpiresAt = now().Add(45 * time.Minute)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))

	got, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	require.NoError(t, err)
	assert.WithinDuration(t, c.ExpiresAt, got.ExpiresAt, time.Second,
		"token expiry moved in storage: wrote %v, read %v", c.ExpiresAt, got.ExpiresAt)
}

func testMetadataRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	c.Metadata = map[string]string{
		"avatar_url": "https://example.test/a.png",
		"locale":     "en-GB",
	}
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))

	got, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	require.NoError(t, err)
	assert.Equal(t, c.Metadata, got.Metadata)
}

func testEmptyMetadataRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	c.Metadata = nil
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))

	got, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	require.NoError(t, err)
	assert.Empty(t, got.Metadata, "a connection with no metadata must not gain phantom entries")
}

// testProviderLookupMatchesTheKeys pins what this lookup can promise. It
// takes no app id, so it is a global lookup on a provider identity, and the
// contract it can hold on every backend is that whatever comes back actually
// matches the keys asked for. Two tenants may both have the same Google user
// connected; this checks the lookup does not answer with a different identity
// entirely, and that a second identity under the same provider is not
// confused with the first.
func testProviderLookupMatchesTheKeys(t *testing.T, f Fixture) {
	ctx := context.Background()

	a := newConnection(f.UserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, a))
	b := newConnection(f.UserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, b))

	for _, want := range []string{a.ProviderUserID, b.ProviderUserID} {
		got, err := f.Store.GetOAuthConnection(ctx, "google", want)
		require.NoError(t, err)
		assert.Equal(t, want, got.ProviderUserID,
			"lookup for provider user %s answered with %s", want, got.ProviderUserID)
		assert.Equal(t, "google", got.Provider)
	}
}

func testListByUserIsScopedToUser(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	mine := newConnection(f.UserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, mine))
	theirs := newConnection(f.OtherUserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, theirs))

	got, err := f.Store.GetOAuthConnectionsByUserID(ctx, f.UserID)
	require.NoError(t, err)

	var found bool
	for _, c := range got {
		assert.Equal(t, f.UserID, c.UserID,
			"listing leaked another user's connection, which would expose their provider tokens")
		if c.ID == mine.ID {
			found = true
		}
	}
	assert.True(t, found, "listing omitted a connection belonging to the queried user")
}

// testUpdateReplacesTokens is the refresh path. When the provider hands back
// a new access token the old one has to be gone, or the next call keeps using
// a token that has already expired.
func testUpdateReplacesTokens(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))

	oldAccess := c.AccessToken
	refreshed := *c
	refreshed.AccessToken = "ya29.refreshed-" + unique("at")
	refreshed.ExpiresAt = now().Add(2 * time.Hour)
	refreshed.UpdatedAt = now()
	require.NoError(t, f.Store.UpdateOAuthConnection(ctx, &refreshed))

	got, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	require.NoError(t, err)
	assert.Equal(t, refreshed.AccessToken, got.AccessToken, "the refreshed access token was not written")
	assert.NotEqual(t, oldAccess, got.AccessToken, "the stale access token survived a refresh")
	assert.WithinDuration(t, refreshed.ExpiresAt, got.ExpiresAt, time.Second,
		"the new expiry was not written, so this token will be treated as expiring on the old schedule")
	assert.Equal(t, c.RefreshToken, got.RefreshToken,
		"a refresh that does not rotate the refresh token must leave it in place")
}

func testDeleteConnection(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.UserID, f.AppID)
	require.NoError(t, f.Store.CreateOAuthConnection(ctx, c))
	require.NoError(t, f.Store.DeleteOAuthConnection(ctx, c.ID))

	_, err := f.Store.GetOAuthConnection(ctx, c.Provider, c.ProviderUserID)
	assert.Error(t, err, "a disconnected provider must stop resolving; the user asked for it to be gone")

	list, err := f.Store.GetOAuthConnectionsByUserID(ctx, f.UserID)
	require.NoError(t, err)
	for _, existing := range list {
		assert.NotEqual(t, c.ID, existing.ID, "a deleted connection still appears in the user's listing")
	}
}
