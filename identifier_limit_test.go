package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/ratelimit"
)

// A distributed attacker rotates addresses, so the engine also budgets
// attempts per target identifier.

func TestSignIn_IdentifierBudget(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	eng.SetRateLimiter(ratelimit.NewMemoryLimiter())
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	ctx := context.Background()
	_, _, err = eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "target@example.com", Password: "SecureP@ss123", FirstName: "T"})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "target@example.com", Password: "wrong-password"})
		assert.ErrorIs(t, err, account.ErrInvalidCredentials, "attempt %d is a plain refusal", i+1)
	}
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "target@example.com", Password: "SecureP@ss123"})
	assert.ErrorIs(t, err, account.ErrRateLimited, "the sixth attempt in a window is refused even with the right password")

	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "someone-else@example.com", Password: "x"})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials, "another identifier has its own budget")
}

func TestForgotPassword_IdentifierBudget(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	eng.SetRateLimiter(ratelimit.NewMemoryLimiter())
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err = eng.ForgotPassword(ctx, appID, "victim@example.com")
		assert.NoError(t, err, "request %d fits the budget", i+1)
	}
	_, err = eng.ForgotPassword(ctx, appID, "victim@example.com")
	assert.ErrorIs(t, err, account.ErrRateLimited)
}
