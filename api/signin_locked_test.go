package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/lockout"
)

// A locked account answers 423 with a Retry-After header and a stable code,
// never a 500.
func TestSignIn_LockedAnswers423WithRetryAfter(t *testing.T) {
	a, eng := newTestAPI(t)
	eng.SetLockoutTracker(lockout.NewMemoryTracker(lockout.WithMaxAttempts(2), lockout.WithLockoutDuration(10*time.Minute)))
	h := withTestKey(a.Handler())
	signUp(t, eng, "locked-http@test.com", "SecureP@ss123")

	wrong := `{"email":"locked-http@test.com","password":"wrong-password"}`
	for i := 0; i < 2; i++ {
		require.Equal(t, http.StatusUnauthorized, postJSONRaw(t, h, "/v1/signin", wrong, nil).Code, "attempt %d", i+1)
	}

	rec := postJSONRaw(t, h, "/v1/signin", `{"email":"locked-http@test.com","password":"SecureP@ss123"}`, nil)
	assert.Equal(t, http.StatusLocked, rec.Code, "body=%s", rec.Body.String())
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), "account_locked")
	assert.Contains(t, rec.Body.String(), "retry_after")
}
