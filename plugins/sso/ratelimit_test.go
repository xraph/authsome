package sso

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
	"github.com/xraph/authsome/ratelimit"
)

// The public SSO surface shares one per-address budget.
func TestSSO_PublicRoutesAreRateLimited(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.SSOLimit = 1
	p := New()
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()

	hit := func() int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/sso/login", strings.NewReader(`{"email":"someone@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.NotEqual(t, http.StatusTooManyRequests, hit(), "the first request fits the budget")
	assert.Equal(t, http.StatusTooManyRequests, hit(), "the second request in the window is refused")
}
