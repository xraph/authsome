package oauth2test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

// testGrantLifecycle proves a grant is keyed by app, user and client: an
// upsert creates it once and then replaces its scopes, lookups answer only
// for that key, and a revocation removes it.
func testGrantLifecycle(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))

	_, err := f.Store.GetGrant(ctx, f.AppID, f.UserID, c.ClientID)
	require.ErrorIs(t, err, oauth2provider.ErrGrantNotFound)

	g := &oauth2provider.Grant{AppID: f.AppID, UserID: f.UserID, ClientID: c.ClientID, Scopes: []string{"openid", "profile"}}
	require.NoError(t, f.Store.UpsertGrant(ctx, g))
	assert.False(t, g.ID.IsNil(), "upsert assigns an id")
	first, err := f.Store.GetGrant(ctx, f.AppID, f.UserID, c.ClientID)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile"}, first.Scopes)
	assert.True(t, first.Covers([]string{"openid"}))
	assert.False(t, first.Covers([]string{"openid", "email"}))

	again := &oauth2provider.Grant{AppID: f.AppID, UserID: f.UserID, ClientID: c.ClientID, Scopes: []string{"openid", "profile", "email"}}
	require.NoError(t, f.Store.UpsertGrant(ctx, again))
	assert.Equal(t, first.ID.String(), again.ID.String(), "the second approval widens the same grant")
	widened, err := f.Store.GetGrant(ctx, f.AppID, f.UserID, c.ClientID)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile", "email"}, widened.Scopes)
	assert.Equal(t, first.CreatedAt.Unix(), widened.CreatedAt.Unix(), "creation time survives the update")

	listed, err := f.Store.ListGrantsByUser(ctx, f.AppID, f.UserID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, c.ClientID, listed[0].ClientID)
	other, err := f.Store.ListGrantsByUser(ctx, f.AppID, f.OtherUserID())
	require.NoError(t, err)
	assert.Empty(t, other, "another user's list does not carry this grant")

	require.NoError(t, f.Store.DeleteGrant(ctx, f.AppID, f.UserID, c.ClientID))
	_, err = f.Store.GetGrant(ctx, f.AppID, f.UserID, c.ClientID)
	require.ErrorIs(t, err, oauth2provider.ErrGrantNotFound)
	assert.ErrorIs(t, f.Store.DeleteGrant(ctx, f.AppID, f.UserID, c.ClientID), oauth2provider.ErrGrantNotFound)
}

// testClientFirstPartyRoundTrip proves the flag that skips consent survives
// the store on create and update.
func testClientFirstPartyRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	c.FirstParty = true
	require.NoError(t, f.Store.CreateClient(ctx, c))
	got, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.True(t, got.FirstParty)
	got.FirstParty = false
	require.NoError(t, f.Store.UpdateClient(ctx, got))
	again, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.False(t, again.FirstParty)
}
