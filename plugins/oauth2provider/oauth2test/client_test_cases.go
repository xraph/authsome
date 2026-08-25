package oauth2test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func testClientCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))

	// Lookup by the public client_id, which is what the token endpoint uses.
	got, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, c.AppID, got.AppID)
	assert.Equal(t, c.Name, got.Name)
	assert.Equal(t, c.ClientSecret, got.ClientSecret)
	assert.Equal(t, c.Public, got.Public)

	// Lookup by the internal id, which is what the dashboard uses.
	byID, err := f.Store.GetClientByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.ClientID, byID.ClientID)

	require.NoError(t, f.Store.DeleteClient(ctx, c.ID))
	_, err = f.Store.GetClient(ctx, c.ClientID)
	require.Error(t, err, "client must be gone after delete")
	assert.True(t, errors.Is(err, oauth2provider.ErrClientNotFound),
		"delete then get must report ErrClientNotFound, got %v", err)
}

func testClientNotFound(t *testing.T, f Fixture) {
	ctx := context.Background()

	_, err := f.Store.GetClient(ctx, unique("missing"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, oauth2provider.ErrClientNotFound),
		"GetClient on an unknown client_id must return ErrClientNotFound, got %v", err)

	_, err = f.Store.GetClientByID(ctx, id.NewOAuth2ClientID())
	require.Error(t, err)
	assert.True(t, errors.Is(err, oauth2provider.ErrClientNotFound),
		"GetClientByID on an unknown id must return ErrClientNotFound, got %v", err)
}

// testClientListIsAppScoped is the app-tenancy boundary: one app must never
// see another app's clients. This is the same class of bug as a missing
// app_id predicate on any other listing query.
func testClientListIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	mine := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, mine))
	theirs := newClient(f.OtherAppID)
	require.NoError(t, f.Store.CreateClient(ctx, theirs))

	got, err := f.Store.ListClients(ctx, f.AppID)
	require.NoError(t, err)
	for _, c := range got {
		assert.Equal(t, f.AppID, c.AppID, "ListClients leaked a client from another app")
		assert.NotEqual(t, theirs.ID, c.ID, "ListClients returned the other tenant's client")
	}
	var found bool
	for _, c := range got {
		if c.ID == mine.ID {
			found = true
		}
	}
	assert.True(t, found, "ListClients omitted a client belonging to the queried app")
}

// testClientEmptySlicesRoundTrip pins the empty-vs-null contract. A client
// with no redirect URIs must read back as an empty list, not a nil one: nil
// serialises to JSON null, and null is what breaks consumers that expect an
// array. Same failure the core session-roles fix addressed.
func testClientEmptySlicesRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	c.RedirectURIs = nil
	c.Scopes = nil
	c.GrantTypes = nil
	require.NoError(t, f.Store.CreateClient(ctx, c))

	got, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.Empty(t, got.RedirectURIs)
	assert.Empty(t, got.Scopes)
	assert.Empty(t, got.GrantTypes)
	assert.NotNil(t, got.RedirectURIs, "nil redirect URIs must read back as [], not null")
	assert.NotNil(t, got.Scopes, "nil scopes must read back as [], not null")
	assert.NotNil(t, got.GrantTypes, "nil grant types must read back as [], not null")
}

func testClientSlicesRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	c.RedirectURIs = []string{"https://a.test/cb", "https://b.test/cb"}
	c.Scopes = []string{"openid", "profile", "email"}
	c.GrantTypes = []string{"authorization_code", "refresh_token"}
	c.Public = true
	require.NoError(t, f.Store.CreateClient(ctx, c))

	got, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.Equal(t, c.RedirectURIs, got.RedirectURIs, "redirect URI order and contents must survive")
	assert.Equal(t, c.Scopes, got.Scopes)
	assert.Equal(t, c.GrantTypes, got.GrantTypes)
	assert.True(t, got.Public)
}
