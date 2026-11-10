package social

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"
)

// discoveryServer serves only a discovery document, which is all saving a
// provider and resolving one needs. It counts the fetches.
func discoveryServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestIsPublicIP(t *testing.T) {
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111", "100.128.0.1"} {
		assert.True(t, isPublicIP(net.ParseIP(s)), s)
	}
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254",
		"100.64.0.1", "100.127.255.255", "0.0.0.0", "::1", "fe80::1", "fc00::1", "224.0.0.1", "ff02::1"} {
		assert.False(t, isPublicIP(net.ParseIP(s)), s)
	}
}

func TestPublicHTTPClient_RefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, http.NoBody)
	require.NoError(t, err)
	resp, err := publicHTTPClient().Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	assert.ErrorIs(t, err, errPrivateAddress)
	assert.Same(t, publicHTTPClient(), publicHTTPClient(), "one shared client keeps connection reuse")
}

func TestOIDCDiscoveryCache_ReusesUntilExpiry(t *testing.T) {
	srv, hits := discoveryServer(t)
	c := newOIDCDiscoveryCache()
	now := time.Now()
	c.now = func() time.Time { return now }
	ctx := context.Background()

	first, err := c.get(ctx, srv.Client(), srv.URL)
	require.NoError(t, err)
	second, err := c.get(ctx, srv.Client(), srv.URL)
	require.NoError(t, err)
	assert.Same(t, first, second)
	assert.Equal(t, int32(1), hits.Load())

	now = now.Add(oidcDiscoveryTTL)
	_, err = c.get(ctx, srv.Client(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, int32(2), hits.Load())

	_, err = c.get(ctx, srv.Client(), srv.URL+"/missing")
	assert.Error(t, err)
}

func TestProviderSettingFromRequest(t *testing.T) {
	srv, _ := discoveryServer(t)
	p := New(Config{HTTPClient: srv.Client()})
	ctx := context.Background()
	on, off := true, false

	t.Run("generic provider", func(t *testing.T) {
		got, err := p.providerSettingFromRequest(ctx, &AdminUpsertProviderRequest{
			Provider: " Muono ", Type: "OIDC", Issuer: srv.URL + "/", DisplayName: " Muono ",
			ClientID: "cid", ClientSecret: "sec", Enabled: &off,
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, ProviderSetting{
			Name: "muono", ClientID: "cid", ClientSecret: "sec", Enabled: false,
			Type: ProviderTypeOIDC, Issuer: srv.URL, DisplayName: "Muono",
		}, got)
	})

	t.Run("built-in keeps the saved secret", func(t *testing.T) {
		got, err := p.providerSettingFromRequest(ctx, &AdminUpsertProviderRequest{
			Provider: "github", ClientID: "cid", Enabled: &on,
		}, []ProviderSetting{{Name: "GitHub", ClientSecret: "kept"}})
		require.NoError(t, err)
		assert.Equal(t, "kept", got.ClientSecret)
		assert.True(t, got.Enabled)
		assert.Empty(t, got.Type)
	})

	for name, req := range map[string]*AdminUpsertProviderRequest{
		"no provider":              {ClientID: "cid"},
		"unknown built-in":         {Provider: "muono", ClientID: "cid"},
		"issuer on a built-in":     {Provider: "github", Issuer: srv.URL, ClientID: "cid"},
		"unknown type":             {Provider: "muono", Type: "saml", Issuer: srv.URL, ClientID: "cid"},
		"generic with a taken id":  {Provider: "google", Type: "oidc", Issuer: srv.URL, ClientID: "cid"},
		"generic over http":        {Provider: "muono", Type: "oidc", Issuer: "http://id.example.com", ClientID: "cid"},
		"generic without issuer":   {Provider: "muono", Type: "oidc", ClientID: "cid"},
		"no client id":             {Provider: "muono", Type: "oidc", Issuer: srv.URL},
		"issuer that won't answer": {Provider: "muono", Type: "oidc", Issuer: srv.URL + "/missing", ClientID: "cid"},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := p.providerSettingFromRequest(ctx, req, nil)
			assert.Error(t, err)
		})
	}
}

func TestProviderFor_GenericProvider(t *testing.T) {
	srv, _ := discoveryServer(t)
	ctx := context.Background()
	p := New(Config{HTTPClient: srv.Client()})
	tl := log.NewTestLogger()
	p.logger = tl

	prov := p.providerFor(ctx, ProviderSetting{Name: "muono", Type: ProviderTypeOIDC, Issuer: srv.URL, ClientID: "cid"})
	require.NotNil(t, prov)
	assert.Equal(t, "muono", prov.Name())
	assert.Equal(t, srv.URL+"/token", prov.OAuth2Config().Endpoint.TokenURL)

	// A stored entry that no longer validates, or whose issuer is down, is
	// unavailable; the second case is logged.
	assert.Nil(t, p.providerFor(ctx, ProviderSetting{Name: "github", Type: ProviderTypeOIDC, Issuer: srv.URL}))
	assert.Nil(t, p.providerFor(ctx, ProviderSetting{Name: "muono", Type: ProviderTypeOIDC, Issuer: "http://x"}))
	assert.Empty(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"))
	assert.Nil(t, p.providerFor(ctx, ProviderSetting{Name: "muono", Type: ProviderTypeOIDC, Issuer: srv.URL + "/down"}))
	assert.Len(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"), 1)

	p.logger = nil
	assert.Nil(t, p.providerFor(ctx, ProviderSetting{Name: "muono", Type: ProviderTypeOIDC, Issuer: srv.URL + "/down"}))

	// Without a configured client the guarded default is used.
	assert.Same(t, publicHTTPClient(), New().oidcHTTPClient())
}

func TestFlexibleBool_RejectsBadJSON(t *testing.T) {
	var b flexibleBool
	assert.Error(t, b.UnmarshalJSON([]byte("{")))
}
