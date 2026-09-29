package phone_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/ratelimit"
)

// One number receives only so many codes per window, from however many
// client addresses the requests come; each code is a billed SMS.
func TestStart_NumberBudgetHoldsAcrossAddresses(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.ResendVerificationLimit = 1
	sms := &mockSMS{}
	p := phone.New(phone.Config{SMSSender: sms})
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	start := func(addr string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/phone/start",
			jsonBody(t, map[string]string{"phone": "+14155551234", "app_id": cfg.AppID}))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.Equal(t, http.StatusOK, start("203.0.113.1:1000"))
	assert.Equal(t, http.StatusTooManyRequests, start("203.0.113.2:1000"), "a second address does not buy a second code for the same number")
}
