package oauth2provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/ratelimit"
)

// The token, revoke and device endpoints share the token budget; authorize
// has its own.
func TestOAuth2_TokenRoutesAreRateLimited(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.OAuthTokenLimit = 1
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()

	hit := func(path string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader("grant_type=client_credentials"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.NotEqual(t, http.StatusTooManyRequests, hit("/v1/oauth/token"), "the first request fits the budget")
	assert.Equal(t, http.StatusTooManyRequests, hit("/v1/oauth/revoke"), "the budget is shared with revoke")
}
