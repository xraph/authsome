package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// TestDiscovery_AdvertisesDPoPAlgs. Advertised unconditionally: handleDiscovery
// is not app scoped, and a server that can validate ES256 proofs can do so
// whoever is asking. Per-client mode is discovered at registration.
//
// The assertion runs against the marshaled JSON, not just the Go struct.
// The whole point of this field is that a client reads it off the wire, so a
// struct-level check would still pass with a misspelled json tag or an
// unexported field, which would leave the feature invisible to every real
// client while looking correct here.
func TestDiscovery_AdvertisesDPoPAlgs(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw), "body: %s", rec.Body.String())

	rawAlgs, ok := raw["dpop_signing_alg_values_supported"]
	require.True(t, ok, "discovery document is missing dpop_signing_alg_values_supported: %s", rec.Body.String())

	rawList, ok := rawAlgs.([]any)
	require.True(t, ok, "dpop_signing_alg_values_supported is not a JSON array: %v", rawAlgs)

	algs := make([]string, 0, len(rawList))
	for _, a := range rawList {
		s, ok := a.(string)
		require.True(t, ok, "dpop_signing_alg_values_supported entry is not a string: %v", a)
		algs = append(algs, s)
	}
	assert.Equal(t, dpop.SupportedAlgs(), algs,
		"the wire value must match dpop.SupportedAlgs(), the promise the server is making about which proofs it will accept")

	var resp oauth2provider.DiscoveryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, dpop.SupportedAlgs(), resp.DPoPSigningAlgValuesSupported)
}
