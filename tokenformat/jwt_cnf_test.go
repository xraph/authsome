package tokenformat_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/tokenformat"
)

func newCnfTestJWT(t *testing.T) *tokenformat.JWT {
	t.Helper()
	f, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-at-least-32-bytes!!"),
	})
	require.NoError(t, err)
	return f
}

// TestJWT_CnfRoundTrip: a stateless validator has no session row to consult,
// so the binding has to travel inside the token or it cannot be enforced.
func TestJWT_CnfRoundTrip(t *testing.T) {
	f := newCnfTestJWT(t)
	const jkt = "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"

	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:    "user_1",
		AppID:     "app_1",
		SessionID: "sess_1",
		DPoPJKT:   jkt,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	got, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Equal(t, jkt, got.DPoPJKT)
}

// TestJWT_NoCnfWhenUnbound keeps unbound tokens byte-for-byte as they are
// today. An empty cnf object in the payload would change every existing
// token's shape for no reason.
func TestJWT_NoCnfWhenUnbound(t *testing.T) {
	f := newCnfTestJWT(t)

	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:    "user_1",
		SessionID: "sess_1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	got, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Empty(t, got.DPoPJKT)

	parsed, _, err := jwt.NewParser().ParseUnverified(tok, jwt.MapClaims{})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	_, present := claims["cnf"]
	assert.False(t, present, "an unbound token must carry no cnf member at all")
}
