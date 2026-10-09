package passkey

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

// Every ceremony route shares the passkey budget, keyed by address.
func TestPasskey_CeremonyRoutesAreRateLimited(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.PasskeyLimit = 1
	p := New(Config{RPID: "localhost"})
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()

	hit := func(path string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.NotEqual(t, http.StatusTooManyRequests, hit("/v1/passkeys/login/begin"), "the first ceremony fits the budget")
	assert.Equal(t, http.StatusTooManyRequests, hit("/v1/passkeys/register/begin"), "the budget is shared across ceremony routes")
}
