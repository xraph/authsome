package oauth2provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// TestUpdateClient_SQLiteRoundTrip pins that an edit actually reaches disk and
// comes back, on a backend that serialises rather than handing back the same
// pointer it stored.
//
// The memory store cannot prove this. MemoryStore.GetClientByID returns the
// live pointer from its map, so a handler that mutated the struct and never
// called UpdateClient would still satisfy a re-read there. SQLite and Postgres
// share oauth2ClientModel and the fromOAuth2Client/toOAuth2Client converters
// (see store_sqlite.go and store_postgres.go, neither of which defines its own
// column list), so this covers the mapping path Postgres uses too. Mongo has
// its own oauth2ClientDoc and is not covered here.
//
// Resources is the field this exercise is really about: it arrived with RFC
// 8707 while UpdateClient arrived with dynamic client registration, on
// separate branches. Nothing had yet asserted that a write through the second
// carries the first.
func TestUpdateClient_SQLiteRoundTrip(t *testing.T) {
	st, appID := newSQLiteClientStore(t)
	ctx := context.Background()

	clientPK := id.NewOAuth2ClientID()
	require.NoError(t, st.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID:           clientPK,
		AppID:        appID,
		Name:         "Before",
		ClientID:     "sqlite-update-client",
		RedirectURIs: []string{"https://app.example.com/cb"},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"authorization_code"},
		CreatedAt:    fixedTimestamp,
		UpdatedAt:    fixedTimestamp,
	}))

	// Dereference into a fresh value so the write can only reach the store
	// through UpdateClient, never through a pointer the store still holds.
	loaded, err := st.GetClientByID(ctx, clientPK)
	require.NoError(t, err)
	updated := *loaded
	updated.Name = "After"
	updated.Scopes = []string{"openid", "profile", "email"}
	updated.Resources = []string{"https://api.example.com"}

	require.NoError(t, st.UpdateClient(ctx, &updated))

	got, err := st.GetClientByID(ctx, clientPK)
	require.NoError(t, err)
	assert.Equal(t, "After", got.Name)
	assert.Equal(t, []string{"openid", "profile", "email"}, got.Scopes)
	assert.Equal(t, []string{"https://api.example.com"}, got.Resources)
	// The identifier is what every deployed integration is pinned to.
	assert.Equal(t, "sqlite-update-client", got.ClientID)
}
