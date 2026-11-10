package authclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authclient "github.com/xraph/authsome/sdk/go"
)

// The consent methods used to be generated with no arguments, so a client
// could neither say which parked request it meant nor send the decision.
func TestClient_ConsentPageSendsCID(t *testing.T) {
	server, seen := captureRequest(t, `{"cid":"c-1","csrf":"k","client_id":"acme","client_name":"Acme","scopes":["openid"],"action":"/v1/oauth/consent"}`)
	client := authclient.NewClient(server.URL)

	view, err := client.Oauth2ConsentPage(context.Background(), &authclient.Oauth2ConsentPageParams{Cid: "c-1"})
	require.NoError(t, err)

	req := <-seen
	assert.Equal(t, http.MethodGet, req.Method)
	assert.Equal(t, "c-1", req.URL.Query().Get("cid"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"), "the server answers JSON only when asked")
	assert.Equal(t, "Acme", view.ClientName)
	assert.Equal(t, "k", view.Csrf)
	assert.Equal(t, []string{"openid"}, view.Scopes)
}

func TestClient_ConsentDecisionSendsTheBody(t *testing.T) {
	server, seen := captureJSONRequest(t, `{"redirect_url":"https://app.example.com/cb?code=x"}`)
	client := authclient.NewClient(server.URL)

	res, err := client.Oauth2ConsentDecision(context.Background(), &authclient.Oauth2ConsentDecisionRequest{
		Cid: "c-1", Csrf: "k", Decision: "approve",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/cb?code=x", res.RedirectURL)

	got := <-seen
	assert.Equal(t, map[string]string{"cid": "c-1", "csrf": "k", "decision": "approve"}, got)
}

// captureJSONRequest records one JSON request body.
func captureJSONRequest(t *testing.T, body string) (*httptest.Server, chan map[string]string) {
	t.Helper()
	seen := make(chan map[string]string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in)
		seen <- in
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, seen
}
