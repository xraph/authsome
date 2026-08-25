package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscoveryResourceIndicators asserts that the discovery document served
// at /.well-known/openid-configuration (the only well-known route this
// plugin registers; there is no /.well-known/oauth-authorization-server)
// advertises resource_indicators_supported. Clients look for this field to
// decide whether it is worth sending a "resource" parameter at all.
func TestDiscoveryResourceIndicators(t *testing.T) {
	_, _, mux := newFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))

	got, ok := raw["resource_indicators_supported"]
	require.True(t, ok, "discovery document must include resource_indicators_supported")
	assert.Equal(t, true, got)
}
