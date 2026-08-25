package tokenformat_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/tokenformat"
)

func newTestJWT(t *testing.T) tokenformat.Format {
	t.Helper()
	f, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-secret-at-least-32-bytes-long!!"),
	})
	require.NoError(t, err)
	return f
}

func TestActClaimNests(t *testing.T) {
	raw, err := json.Marshal(tokenformat.ActClaim{
		Subject: "workload:svc_outer",
		Act:     &tokenformat.ActClaim{Subject: "user:usr_inner"},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"sub":"workload:svc_outer","act":{"sub":"user:usr_inner"}}`, string(raw))
}

func TestJWTRoundTripsActClaim(t *testing.T) {
	f := newTestJWT(t)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID: "usr_1", AppID: "aapp_1", SessionID: "ases_1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
		Act: &tokenformat.ActClaim{Subject: "workload:svc_actor"},
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	require.NotNil(t, out.Act)
	assert.Equal(t, "workload:svc_actor", out.Act.Subject)
}

// Impersonation emits no act claim at all (RFC 8693 section 1.1), so a nil
// Act has to stay absent rather than serialising as null.
func TestJWTOmitsActWhenNil(t *testing.T) {
	f := newTestJWT(t)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID: "usr_1", AppID: "aapp_1", SessionID: "ases_1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Nil(t, out.Act)
}
