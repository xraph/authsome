package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// patchClient drives PATCH /v1/admin/oauth/clients/:clientId, the admin
// endpoint that edits an existing client in place. As with createClient,
// newFixture wires no engine, so the group's AdminGuard middleware resolves
// to nil and the route is reachable directly in tests.
//
// The path carries the client's internal OAuth2ClientID (the primary key),
// not the OAuth2 client_id string, matching the existing DELETE route.
// callerApp goes on the request context the way authprovider.BridgeToContext
// does in production once SessionGuard has resolved the session. The handler
// requires it, so these tests pass the app that owns the client under edit.
// The tenancy rule itself is covered separately, below.
func patchClient(t *testing.T, mux forge.Router, callerApp id.AppID, clientPK string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	ctx := middleware.WithAppID(context.Background(), callerApp)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch,
		"/v1/admin/oauth/clients/"+clientPK, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// TestUpdateClient_RenamesAClient is the first cut at the admin update route:
// an operator changes one field and the change takes effect without the
// client_id rotating. Today the only way to change a name is to delete and
// recreate, which mints a new client_id and breaks every deployed integration.
//
// The assertion is on the response body rather than a follow-up store read.
// MemoryStore.GetClient hands back the pointer it stores, so a handler that
// mutated that pointer and never called UpdateClient would still satisfy a
// re-read. Persistence proper is pinned separately against a backend that
// round-trips through serialisation.
func TestUpdateClient_RenamesAClient(t *testing.T) {
	_, st, mux := newFixture(t)

	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	require.Equal(t, "Confidential", before.Name)

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{"name": "Renamed"})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "Renamed", got["name"])
	// The whole point of the endpoint: editing a client must not rotate its
	// credentials the way delete-and-recreate does.
	assert.Equal(t, confidentialID, got["client_id"])
}

// A client id that parses but matches nothing is a client error, not a server
// error. Without this the store's ErrClientNotFound falls through to the
// generic 500 wrapper and an operator editing a stale id sees "internal
// error", which reads like the service is broken rather than like a typo.
func TestUpdateClient_UnknownClientIs404(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := patchClient(t, mux, id.NewAppID(), id.NewOAuth2ClientID().String(), map[string]any{"name": "Nope"})
	assert.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())
}

// Both new routes have to honour the same tenancy rule handleDeleteClient
// applies. Without it they would be the only unscoped admin client routes in
// the package, and a caller holding manage:oauth2_client on one app could edit
// or re-credential another app's clients by guessing a primary key.
//
// A mismatch answers 404, not 403, for the reason main's delete handler gives:
// a 403 would confirm the client exists and turn the route into a probe.
func TestAdminClientRoutes_RefuseAnotherAppsClient(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	otherApp := id.NewAppID()
	require.NotEqual(t, otherApp, before.AppID)

	t.Run("patch", func(t *testing.T) {
		rec := patchClient(t, mux, otherApp, before.ID.String(), map[string]any{"name": "Stolen"})
		require.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())

		after, err := st.GetClient(context.Background(), confidentialID)
		require.NoError(t, err)
		assert.Equal(t, "Confidential", after.Name, "a refused edit must not apply")
	})

	t.Run("rotate secret", func(t *testing.T) {
		fresh, err := st.GetClient(context.Background(), confidentialID)
		require.NoError(t, err)
		oldHash := fresh.ClientSecret

		rec := rotateSecret(t, mux, otherApp, fresh.ID.String())
		require.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())

		after, err := st.GetClient(context.Background(), confidentialID)
		require.NoError(t, err)
		assert.Equal(t, oldHash, after.ClientSecret, "a refused rotation must not change the secret")
	})
}

// updateClientBody mirrors the response shape for assertions. Decoding into a
// typed value rather than map[string]any keeps the slice comparisons honest: a
// map decode yields []any, which never compares equal to a []string.
type updateClientBody struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"client_id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"`
	Resources    []string `json:"resources"`
	Public       bool     `json:"public"`
}

// The sparse contract: a field the caller did not send must survive the write
// untouched. This is the difference between PATCH and PUT, and it is what
// stops an operator who came to widen scopes from silently wiping the redirect
// URIs they never mentioned.
func TestUpdateClient_OmittedFieldsSurvive(t *testing.T) {
	_, st, mux := newFixture(t)

	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	wantURIs := append([]string(nil), before.RedirectURIs...)
	wantGrants := append([]string(nil), before.GrantTypes...)
	wantName := before.Name

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"scopes": []string{"openid", "email", "offline_access"},
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got updateClientBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, []string{"openid", "email", "offline_access"}, got.Scopes)
	assert.Equal(t, wantName, got.Name)
	assert.Equal(t, wantURIs, got.RedirectURIs)
	assert.Equal(t, wantGrants, got.GrantTypes)
}

// Present-but-empty is a real instruction, distinct from absent. An operator
// clearing a public client's redirect URIs must be able to say so, and the
// pointer fields are what let the handler tell the two apart.
func TestUpdateClient_ExplicitEmptyClears(t *testing.T) {
	_, st, mux := newFixture(t)

	before, err := st.GetClient(context.Background(), publicID)
	require.NoError(t, err)
	require.NotEmpty(t, before.RedirectURIs)

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"redirect_uris": []string{},
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got updateClientBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Empty(t, got.RedirectURIs)
}

// The identity and credential-shape fields are refused rather than ignored.
//
// Silently dropping them is the dangerous option: an operator who sends
// {"public": true} and gets a 200 back reasonably concludes the client is now
// public, when in fact it still demands a secret. Rotating a secret or
// flipping the client type is a credential operation and lives on its own
// route, and client_id is the one value the whole endpoint exists to preserve.
func TestUpdateClient_ImmutableFieldsAreRejected(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	for _, tc := range []struct {
		field string
		value any
		want  string
	}{
		{"client_id", "attacker-chosen-id", "client_id"},
		{"app_id", id.NewAppID().String(), "app_id"},
		{"public", true, "public"},
		// Public and TokenEndpointAuthMethod encode one fact between them,
		// and models.go is explicit that every write site keeps them in sync
		// by deriving one from the other. Accepting this field alone would
		// create exactly the disagreement that comment warns about.
		{"token_endpoint_auth_method", "none", "token_endpoint_auth_method"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{tc.field: tc.value})
			require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
			// Pin the field name so a 400 raised for some unrelated reason
			// cannot masquerade as this rule firing.
			assert.Contains(t, rec.Body.String(), tc.want)
		})
	}

	// None of the refused requests may have partially applied.
	after, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	assert.Equal(t, confidentialID, after.ClientID)
	assert.False(t, after.Public)
}

// A confidential client with no redirect URI can never complete an
// authorization code flow, and handleCreateClient already refuses to create
// one. The update path has to hold the same line or it becomes a way to reach
// a state creation forbids.
func TestUpdateClient_ConfidentialClientKeepsARedirectURI(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"redirect_uris": []string{},
	})
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "redirect_uris")

	after, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	assert.NotEmpty(t, after.RedirectURIs)
}

// Registration-time redirect URI rules apply to edits too. Without this an
// operator could not register an http:// URI on a non-loopback host at create
// time but could reach the same state with a follow-up edit.
func TestUpdateClient_RedirectURIIsValidated(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"redirect_uris": []string{"http://evil.example.com/cb"},
	})
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "loopback")
}

// The admin grant allowlist is deliberately wider than clampGrantTypes, which
// governs the open /v1/oauth/register endpoint and exists to keep anonymous
// callers away from client_credentials. An operator holding manage:oauth2_client
// is not an anonymous caller, and the admin create path has always been able to
// register that grant. Reusing the dynamic clamp here would make grant_types
// uneditable for exactly the service clients that need it.
func TestUpdateClient_AdminMaySetClientCredentials(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"grant_types": []string{"client_credentials"},
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got updateClientBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, []string{"client_credentials"}, got.GrantTypes)
}

// A grant the token endpoint does not implement is refused rather than
// stored. Accepting it would leave a client advertising a capability that
// fails at first use with "unsupported grant_type".
func TestUpdateClient_UnknownGrantTypeIsRejected(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"grant_types": []string{"implicit"},
	})
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "implicit")
}

// The reason this endpoint exists. RFC 8707 gave every client a resource
// allowlist that denies by default, so a client registered before that merge
// can target nothing and had no way to be given an entry.
func TestUpdateClient_SetsResourceAllowlist(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	require.Empty(t, before.Resources, "fixture client starts with the deny-by-default empty allowlist")

	rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
		"resources": []string{"https://api.example.com"},
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got updateClientBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, []string{"https://api.example.com"}, got.Resources)
}

// Resource syntax is checked with the same helper the create path and the
// request-time resolver use, so an allowlist entry can never be a value a
// client would then be refused for requesting.
func TestUpdateClient_ResourceURIIsValidated(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	t.Run("non-absolute is rejected", func(t *testing.T) {
		rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
			"resources": []string{"not-a-uri"},
		})
		require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "is not an absolute URI")
	})

	t.Run("a fragment is rejected", func(t *testing.T) {
		rec := patchClient(t, mux, before.AppID, before.ID.String(), map[string]any{
			"resources": []string{"https://api.example.com#frag"},
		})
		require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "must not include a fragment")
	})
}

// Create and update must enforce the same redirect URI rule. Without this the
// admin create path stays more permissive than the update path, so an operator
// could register a URI at creation that they are then forbidden from editing,
// and a value the dynamic registration endpoint rejects outright could still
// be introduced through the admin surface.
func TestCreateClient_RedirectURIIsValidated(t *testing.T) {
	_, st, mux := newFixture(t)
	appID := id.NewAppID()

	rec := createClient(t, mux, appID, map[string]any{
		"app_id":        appID.String(),
		"name":          "Loose Client",
		"redirect_uris": []string{"http://evil.example.com/cb"},
	})
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "loopback")

	clients, err := st.ListClients(context.Background(), appID)
	require.NoError(t, err)
	assert.Empty(t, clients, "a refused create must not persist a client")
}

// rotateSecret drives POST /v1/admin/oauth/clients/:clientId/secret.
func rotateSecret(t *testing.T, mux forge.Router, callerApp id.AppID, clientPK string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithAppID(context.Background(), callerApp)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost,
		"/v1/admin/oauth/clients/"+clientPK+"/secret", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// Rotation is the other half of "you should not have to delete and recreate".
// It lives on its own route rather than as a flag on PATCH so that the one
// endpoint which hands back a credential stays small and separately auditable.
func TestRotateClientSecret_IssuesAWorkingSecretOnce(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	oldHash := before.ClientSecret
	require.NotEmpty(t, oldHash)

	rec := rotateSecret(t, mux, before.AppID, before.ID.String())
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.NotEmpty(t, got.ClientSecret, "the raw secret is returned exactly once")
	assert.Equal(t, confidentialID, got.ClientID, "rotating a secret must not rotate the client_id")

	after, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	// Stored as a bcrypt hash, never in the clear.
	assert.NotEqual(t, got.ClientSecret, after.ClientSecret)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(after.ClientSecret), []byte(got.ClientSecret)),
		"the returned secret must verify against the newly stored hash")
	// The previous secret must no longer authenticate.
	assert.Error(t, bcrypt.CompareHashAndPassword([]byte(after.ClientSecret), []byte(confidentialSecret)))
}

// A public client has no secret by definition, so there is nothing to rotate.
// Minting one would contradict Public and TokenEndpointAuthMethod: "none".
func TestRotateClientSecret_PublicClientIsRejected(t *testing.T) {
	_, st, mux := newFixture(t)
	before, err := st.GetClient(context.Background(), publicID)
	require.NoError(t, err)

	rec := rotateSecret(t, mux, before.AppID, before.ID.String())
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())

	after, err := st.GetClient(context.Background(), publicID)
	require.NoError(t, err)
	assert.Empty(t, after.ClientSecret, "a refused rotation must not leave a secret behind")
}
