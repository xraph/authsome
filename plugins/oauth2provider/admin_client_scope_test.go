package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// adminReq builds a request to an admin route carrying the caller's app on the
// context, which is what authprovider.BridgeToContext does in production after
// the group's SessionGuard resolves the session. Pass a nil callerApp to model
// a request that reached the handler without one.
func adminReq(t *testing.T, method, path string, callerApp *id.AppID, body any) *http.Request {
	t.Helper()
	ctx := context.Background()
	if callerApp != nil {
		ctx = middleware.WithAppID(ctx, *callerApp)
	}

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}

	req := httptest.NewRequestWithContext(ctx, method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func serve(mux forge.Router, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// seedClient stores a client owned by appID and returns it.
func seedClient(t *testing.T, st oauth2provider.Store, appID id.AppID, clientID string) *oauth2provider.OAuth2Client {
	t.Helper()
	c := &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     clientID,
		Name:         "Seeded " + clientID,
		RedirectURIs: []string{registeredURI},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"authorization_code"},
	}
	require.NoError(t, st.CreateClient(context.Background(), c))
	return c
}

// TestAdminClientAppScope covers the tenancy boundary on the OAuth2 admin
// client surface. manage:oauth2_client says the caller may administer clients;
// it does not say which app's clients, and every route here has to answer that
// second question against the caller's own app.
//
// The by-id route answers with 404 rather than 403 on purpose, matching
// api.assertAppScope: a 403 would confirm that another app's client exists.
func TestAdminClientAppScope(t *testing.T) {
	t.Run("create refuses an app_id that is not the caller's", func(t *testing.T) {
		_, st, mux := newFixture(t)
		caller, other := id.NewAppID(), id.NewAppID()

		rec := serve(mux, adminReq(t, http.MethodPost, "/v1/admin/oauth/clients", &caller, map[string]any{
			"app_id":        other.String(),
			"name":          "Cross-tenant client",
			"redirect_uris": []string{registeredURI},
		}))

		require.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())

		// The status alone would pass if the handler happened to 403 for an
		// unrelated reason after writing the row. Assert the row is absent.
		clients, err := st.ListClients(context.Background(), other)
		require.NoError(t, err)
		assert.Empty(t, clients, "no client should exist in the app the caller is not bound to")
	})

	t.Run("create refuses a request carrying no caller app", func(t *testing.T) {
		_, _, mux := newFixture(t)

		rec := serve(mux, adminReq(t, http.MethodPost, "/v1/admin/oauth/clients", nil, map[string]any{
			"app_id":        id.NewAppID().String(),
			"name":          "Unscoped client",
			"redirect_uris": []string{registeredURI},
		}))

		require.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("list refuses another app's clients", func(t *testing.T) {
		_, st, mux := newFixture(t)
		caller, other := id.NewAppID(), id.NewAppID()
		seedClient(t, st, other, "other-app-client")

		rec := serve(mux, adminReq(t, http.MethodGet,
			"/v1/admin/oauth/clients?app_id="+other.String(), &caller, nil))

		require.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "other-app-client",
			"the refusal must not leak the client it refused to list")
	})

	t.Run("list returns the caller's own clients", func(t *testing.T) {
		_, st, mux := newFixture(t)
		caller := id.NewAppID()
		seedClient(t, st, caller, "own-client")

		rec := serve(mux, adminReq(t, http.MethodGet,
			"/v1/admin/oauth/clients?app_id="+caller.String(), &caller, nil))

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var resp oauth2provider.ListClientsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Clients, 1)
		assert.Equal(t, "own-client", resp.Clients[0].ClientID)
	})

	t.Run("delete reports another app's client as not found", func(t *testing.T) {
		_, st, mux := newFixture(t)
		caller, other := id.NewAppID(), id.NewAppID()
		victim := seedClient(t, st, other, "victim-client")

		rec := serve(mux, adminReq(t, http.MethodDelete,
			"/v1/admin/oauth/clients/"+victim.ID.String(), &caller, nil))

		require.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())

		// The assertion the whole change exists for: the row survived.
		got, err := st.GetClientByID(context.Background(), victim.ID)
		require.NoError(t, err)
		assert.Equal(t, victim.ID, got.ID, "another app's client must not be deleted")
	})

	t.Run("delete removes the caller's own client", func(t *testing.T) {
		_, st, mux := newFixture(t)
		caller := id.NewAppID()
		own := seedClient(t, st, caller, "own-client")

		rec := serve(mux, adminReq(t, http.MethodDelete,
			"/v1/admin/oauth/clients/"+own.ID.String(), &caller, nil))

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		_, err := st.GetClientByID(context.Background(), own.ID)
		assert.True(t, errors.Is(err, oauth2provider.ErrClientNotFound),
			"the caller's own client should be gone, got err=%v", err)
	})

	t.Run("delete of an unknown id is not found", func(t *testing.T) {
		_, _, mux := newFixture(t)
		caller := id.NewAppID()

		rec := serve(mux, adminReq(t, http.MethodDelete,
			"/v1/admin/oauth/clients/"+id.NewOAuth2ClientID().String(), &caller, nil))

		require.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())
	})
}
