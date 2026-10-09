package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/ratelimit"
)

// Reset mail for one address is capped whatever client address asks: the
// per-address route limit is sidestepped here by changing the remote
// address on every request, and the identifier budget still refuses.
func TestForgotPassword_IdentifierBudgetAnswers429(t *testing.T) {
	a, eng := newTestAPI(t)
	eng.SetRateLimiter(ratelimit.NewMemoryLimiter())
	h := withTestKey(a.Handler())

	var last int
	for i := 0; i < 4; i++ {
		addr := "203.0.113." + string(rune('1'+i)) + ":5000"
		last = postJSONRaw(t, h, "/v1/forgot-password", `{"email":"flooded@test.com"}`, func(r *http.Request) *http.Request {
			r.RemoteAddr = addr
			return r
		}).Code
		if i < 3 {
			require.Equal(t, http.StatusOK, last, "request %d is answered as usual", i+1)
		}
	}
	assert.Equal(t, http.StatusTooManyRequests, last, "the fourth request for one address in a window is refused")
}
