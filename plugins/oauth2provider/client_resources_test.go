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

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// createClient drives POST /v1/admin/oauth/clients, the admin endpoint that
// registers a new OAuth2 client. newFixture wires no engine, so the group's
// AdminGuard middleware resolves to nil and the route is reachable directly
// in tests, the same way the fixture leaves every other route unauthenticated.
func createClient(t *testing.T, mux forge.Router, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/admin/oauth/clients", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// TestCreateClientResources covers admin registration of the RFC 8707
// resource allowlist: the field this whole feature is unreachable without,
// since the validator introduced earlier denies by default and an empty
// allowlist can target nothing.
func TestCreateClientResources(t *testing.T) {
	t.Run("valid resources are stored and echoed back", func(t *testing.T) {
		_, st, mux := newFixture(t)
		appID := id.NewAppID()

		rec := createClient(t, mux, map[string]any{
			"app_id":        appID.String(),
			"name":          "Resource Client",
			"redirect_uris": []string{registeredURI},
			"resources":     []string{"https://api.example.com"},
		})

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var resp oauth2provider.CreateClientResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, []string{"https://api.example.com"}, resp.Resources)

		clients, err := st.ListClients(context.Background(), appID)
		require.NoError(t, err)
		require.Len(t, clients, 1)
		assert.Equal(t, []string{"https://api.example.com"}, clients[0].Resources)
	})

	t.Run("a non-absolute URI is rejected", func(t *testing.T) {
		_, _, mux := newFixture(t)

		rec := createClient(t, mux, map[string]any{
			"app_id":        id.NewAppID().String(),
			"name":          "Bad Resource Client",
			"redirect_uris": []string{registeredURI},
			"resources":     []string{"not-a-uri"},
		})

		require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		// Pinning the specific wording proves the "must be absolute" rule
		// fired, and not some other 400 (missing name, bad app_id, ...) that
		// would also produce a 400 but for an unrelated reason.
		assert.Contains(t, rec.Body.String(), "is not an absolute URI")
		assert.NotContains(t, rec.Body.String(), "fragment")
	})

	t.Run("a fragment is rejected", func(t *testing.T) {
		_, _, mux := newFixture(t)

		rec := createClient(t, mux, map[string]any{
			"app_id":        id.NewAppID().String(),
			"name":          "Fragment Resource Client",
			"redirect_uris": []string{registeredURI},
			"resources":     []string{"https://api.example.com#frag"},
		})

		require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		// Pinning the specific wording proves the fragment rule fired, and
		// not the absolute-URI check: "https://api.example.com#frag" parses
		// as an absolute URI just fine, so only the fragment rule can be
		// what rejects it.
		assert.Contains(t, rec.Body.String(), "must not include a fragment")
		assert.NotContains(t, rec.Body.String(), "is not an absolute URI")
	})

	t.Run("no resources field succeeds with an empty allowlist", func(t *testing.T) {
		_, st, mux := newFixture(t)
		appID := id.NewAppID()

		rec := createClient(t, mux, map[string]any{
			"app_id":        appID.String(),
			"name":          "No Resources Client",
			"redirect_uris": []string{registeredURI},
		})

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var resp oauth2provider.CreateClientResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Empty(t, resp.Resources)

		clients, err := st.ListClients(context.Background(), appID)
		require.NoError(t, err)
		require.Len(t, clients, 1)
		assert.Empty(t, clients[0].Resources)
	})
}
