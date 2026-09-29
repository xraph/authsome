package apikey_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/ratelimit"
)

// Guessing keys is throttled per client address: once the failed-attempt
// budget is spent, the strategy refuses before it looks anything up.
func TestStrategy_FailedAttemptsAreThrottledPerAddress(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.APIKeyFailureLimit = 2
	p := apikeyPlugin.New()
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	s := p.Strategy()

	appID := id.NewAppID()
	attempt := func(remoteAddr string) error {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
		req.RemoteAddr = remoteAddr
		req.Header.Set("Authorization", "Bearer sk_test_0123456789abcdef0123456789abcdef")
		req.Header.Set("X-App-ID", appID.String())
		_, err := s.Authenticate(context.Background(), req)
		return err
	}

	for i := 0; i < 2; i++ {
		err := attempt("203.0.113.9:4000")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "too many failed attempts", "attempt %d is a plain refusal", i+1)
	}
	err := attempt("203.0.113.9:4001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many failed attempts", "the budget is spent for this address")

	err = attempt("198.51.100.7:4000")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "too many failed attempts", "another address has its own budget")
}
