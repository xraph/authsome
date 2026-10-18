package authsome_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
)

// An API key authentication is a span and a count by result. An unknown
// prefix and a wrong secret are both "invalid", so neither the trace nor
// the counter says which prefixes exist.
func TestAPIKeyAuthenticationIsTracedAndCounted(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	eng, s := newTestEngine(t, authsome.WithMetricsSink(reg), authsome.WithPlugin(apikeyPlugin.New()))
	ctx := context.Background()
	appID := signInApp(t)
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "apikey-trace@example.com", Password: "SecureP@ss123", FirstName: "K"})
	require.NoError(t, err)
	rawKey, key := e2eCreateAPIKey(t, s, appID, u.ID)

	router := e2eRouter(eng)
	require.NoError(t, router.GET("/test", func(c forge.Context) error { return c.NoContent(http.StatusOK) }))
	call := func(key string) {
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", http.NoBody)
		req.Header.Set("X-API-Key", key)
		req.Header.Set("X-App-ID", appID.String())
		router.ServeHTTP(httptest.NewRecorder(), req)
	}
	// Built from the real key at runtime: one with the right prefix and a
	// wrong secret, one with a prefix nobody holds.
	wrongSecret := rawKey[:len(rawKey)-4] + "zzzz"
	marker := key.KeyPrefix[:len(key.KeyPrefix)-8]
	other := strings.Repeat("0", 8)
	if strings.HasSuffix(key.KeyPrefix, other) {
		other = strings.Repeat("1", 8)
	}
	unknownPrefix := marker + other + rawKey[len(key.KeyPrefix):]

	n := len(rec.Ended())
	call(rawKey)
	call(wrongSecret)
	call(unknownPrefix)

	auths := telemetrytest.Named(spansSince(rec, n), "authenticate api key")
	require.Len(t, auths, 3)
	results := make([]string, 0, len(auths))
	for _, a := range auths {
		results = append(results, telemetrytest.Attrs(a)["authsome.result"])
		assert.Equal(t, codes.Unset, a.Status().Code, "a refused key is an answer, not a failure")
		for _, k := range []string{rawKey, wrongSecret, unknownPrefix} {
			assert.NotContains(t, telemetrytest.Text(a), k)
		}
	}
	assert.Equal(t, []string{"ok", "invalid", "invalid"}, results)
	body := scrape()
	assert.Contains(t, body, `forge_authsome_apikey_authentications_total{result="ok"} 1`)
	assert.Contains(t, body, `forge_authsome_apikey_authentications_total{result="invalid"} 2`)
}
