package authclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authclient "github.com/xraph/authsome/sdk/go"
)

// RFC 8707 lets a client name more than one target service in one request, and
// the way it says so is by sending resource once per value. The server reads
// every value off the raw request, so what matters here is what actually
// reaches the wire: two values under one name, not one value containing both.
//
// This is the end of the chain the parameter has to travel. It was declared on
// the routes with an option the OpenAPI generator never read, so it reached no
// document, no SDK and no request. These two tests are what "a generated client
// can send two resource values" means concretely.
const (
	firstResource  = "https://api.example.com"
	secondResource = "https://files.example.com"
)

// captureRequest runs a server that records one request and returns it.
func captureRequest(t *testing.T, body string) (*httptest.Server, chan *http.Request) {
	t.Helper()

	seen := make(chan *http.Request, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse before handing the request over: the body is closed once the
		// handler returns, so PostForm has to be populated here.
		_ = r.ParseForm()

		seen <- r

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server, seen
}

// The authorization endpoint is a GET, so both values ride in the query string.
func TestClient_AuthorizeSendsEveryResourceValue(t *testing.T) {
	server, seen := captureRequest(t, `{}`)

	client := authclient.NewClient(server.URL)

	err := client.Oauth2Authorize(context.Background(), &authclient.Oauth2AuthorizeParams{
		ResponseType: "code",
		ClientID:     "client-123",
		Resource:     []string{firstResource, secondResource},
	})
	require.NoError(t, err)

	got := <-seen

	assert.Equal(t, []string{firstResource, secondResource}, got.URL.Query()["resource"],
		"both values should arrive under one repeated name")
}

// The device authorization endpoint is a POST, so both values ride in the
// urlencoded body, which is where RFC 6749 puts a POST endpoint's parameters.
func TestClient_DeviceAuthorizeSendsEveryResourceValue(t *testing.T) {
	server, seen := captureRequest(t, `{"device_code":"d","user_code":"u","verification_uri":"v","expires_in":1,"interval":1}`)

	client := authclient.NewClient(server.URL)

	_, err := client.Oauth2DeviceAuthorize(context.Background(), &authclient.Oauth2DeviceAuthorizeRequest{
		ClientID: "client-123",
		Resource: []string{firstResource, secondResource},
	})
	require.NoError(t, err)

	got := <-seen

	assert.Equal(t, []string{firstResource, secondResource}, got.PostForm["resource"],
		"both values should arrive under one repeated name")
}

// The token endpoint already worked, because the request struct carried the
// field and the form encoder already looped over it. It is here so a change to
// the shared query or form emission cannot quietly take it away again.
func TestClient_TokenSendsEveryResourceValue(t *testing.T) {
	server, seen := captureRequest(t, `{"access_token":"a","token_type":"Bearer"}`)

	client := authclient.NewClient(server.URL)

	_, err := client.Oauth2Token(context.Background(), &authclient.Oauth2TokenRequest{
		GrantType: "client_credentials",
		ClientID:  "client-123",
		Resource:  []string{firstResource, secondResource},
	})
	require.NoError(t, err)

	got := <-seen

	assert.Equal(t, []string{firstResource, secondResource}, got.PostForm["resource"],
		"both values should arrive under one repeated name")
}
