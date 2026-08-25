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
	c.Resources = nil
	require.NoError(t, f.Store.CreateClient(ctx, c))

	got, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.Empty(t, got.RedirectURIs)
	assert.Empty(t, got.Scopes)
	assert.Empty(t, got.GrantTypes)
	assert.NotNil(t, got.RedirectURIs, "nil redirect URIs must read back as [], not null")
	assert.NotNil(t, got.Scopes, "nil scopes must read back as [], not null")
	assert.NotNil(t, got.GrantTypes, "nil grant types must read back as [], not null")
	assert.Empty(t, got.Resources)
	assert.NotNil(t, got.Resources, "nil resources must read back as [], not null")
}

func testClientSlicesRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	c.RedirectURIs = []string{"https://a.test/cb", "https://b.test/cb"}
	c.Scopes = []string{"openid", "profile", "email"}
	c.GrantTypes = []string{"authorization_code", "refresh_token"}
	c.Resources = []string{"https://api.example.test", "https://admin.example.test"}
	c.Public = true
	require.NoError(t, f.Store.CreateClient(ctx, c))

	got, err := f.Store.GetClient(ctx, c.ClientID)
	require.NoError(t, err)
	assert.Equal(t, c.RedirectURIs, got.RedirectURIs, "redirect URI order and contents must survive")
	assert.Equal(t, c.Scopes, got.Scopes)
	assert.Equal(t, c.GrantTypes, got.GrantTypes)
	assert.Equal(t, c.Resources, got.Resources,
		"the resource allow-list decides which audiences a token may be issued for")
	assert.True(t, got.Public)
}

// testUpdateClientPersistsEveryField covers the full-record replace that the
// admin update and secret-rotation handlers both write through. It builds a
// separate struct rather than mutating the one it created, so a backend that
// hands back the pointer it already holds cannot pass without writing.
func testUpdateClientPersistsEveryField(t *testing.T, f Fixture) {
	ctx := context.Background()
	original := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, original))

	updated := &oauth2provider.OAuth2Client{
		ID:           original.ID,
		AppID:        original.AppID,
		Name:         "Renamed Client",
		ClientID:     original.ClientID,
		ClientSecret: "rotated-secret",
		RedirectURIs: []string{"https://new.example.test/cb"},
		Scopes:       []string{"openid", "offline_access"},
		GrantTypes:   []string{"client_credentials"},
		Resources:    []string{"https://other.example.test"},
		Public:       !original.Public,
		CreatedAt:    original.CreatedAt,
		UpdatedAt:    now(),
	}
	require.NoError(t, f.Store.UpdateClient(ctx, updated))

	got, err := f.Store.GetClient(ctx, original.ClientID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed Client", got.Name)
	assert.Equal(t, "rotated-secret", got.ClientSecret,
		"secret rotation writes through this path; a dropped write leaves the old secret live")
	assert.Equal(t, updated.RedirectURIs, got.RedirectURIs)
	assert.Equal(t, updated.Scopes, got.Scopes)
	assert.Equal(t, updated.GrantTypes, got.GrantTypes)
	assert.Equal(t, updated.Resources, got.Resources,
		"the resource allow-list must survive a full-record update")
	assert.Equal(t, updated.Public, got.Public)
}

// testUpdateClientOnMissingClient pins the failure mode. Updating a client
// that is not there must say so, not report success against nothing.
func testUpdateClientOnMissingClient(t *testing.T, f Fixture) {
	ghost := newClient(f.AppID)
	err := f.Store.UpdateClient(context.Background(), ghost)
	require.Error(t, err, "updating a client that was never created must not succeed")
	assert.True(t, errors.Is(err, oauth2provider.ErrClientNotFound),
		"a missing client must report ErrClientNotFound, got %v", err)
}
