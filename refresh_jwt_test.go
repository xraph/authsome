package authsome_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/tokenformat"
)

// TestRefresh_PreservesJWTAccessToken pins that refreshing a session for a
// JWT-configured app keeps the access token a JWT. Previously Refresh delegated
// token minting to account.RefreshSession, which always issues an opaque token,
// silently downgrading JWT apps and breaking stateless verification / JWT
// revocation binding.
func TestRefresh_PreservesJWTAccessToken(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)

	eng, _ := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "jwt-refresh@example.com",
		Password:  "SecureP@ss1",
		FirstName: "JWT User",
	})
	require.NoError(t, err)
	require.True(t, tokenformat.IsJWT(sess.Token), "login access token should be a JWT for a JWT-configured app")

	refreshed, err := eng.Refresh(ctx, sess.RefreshToken)
	require.NoError(t, err)
	assert.True(t, tokenformat.IsJWT(refreshed.Token), "refreshed access token must remain a JWT, not downgrade to opaque")
}
