package magiclink_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/ratelimit"
)

// One address receives only so many links per window, from however many
// client addresses the requests come.
func TestSend_AddressBudgetHoldsAcrossClientAddresses(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = testAppIDStr
	cfg.RateLimit.ResendVerificationLimit = 1
	mailer := &mockMailer{}
	p := magiclink.New(magiclink.Config{Mailer: mailer, TokenTTL: 5 * time.Minute})
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	send := func(addr string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/magic-link/send",
			jsonBody(t, map[string]string{"email": "flooded@example.com", "app_id": testAppIDStr}))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	first := send("203.0.113.1:1000")
	assert.NotEqual(t, http.StatusTooManyRequests, first, "the first request fits the budget")
	assert.Equal(t, http.StatusTooManyRequests, send("203.0.113.2:1000"), "a second address does not buy a second link for the same email")
}
