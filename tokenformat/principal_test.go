package tokenformat_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/tokenformat"
)

// A machine token carries the principal kind so the JWT auth path can tell it
// apart from a human one. Before this claim existed, sub was the only signal
// and it was empty for a service account, which made the auth middleware panic
// rather than refuse.
func TestJWTRoundTripsPrincipalClaims(t *testing.T) {
	f := newTestJWT(t)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:        "svc_deploy",
		AppID:         "aapp_1",
		SessionID:     "ases_1",
		PrincipalKind: "workload",
		PrincipalID:   "svc_deploy",
		IssuedAt:      now,
		ExpiresAt:     now.Add(time.Minute),
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "workload", out.PrincipalKind)
	assert.Equal(t, "svc_deploy", out.PrincipalID)
	assert.Equal(t, "svc_deploy", out.UserID, "sub carries the principal id")
}

// A user token has to be unchanged on the wire, so both fields stay absent
// and every token minted before they existed keeps validating.
func TestJWTOmitsPrincipalClaimsForUsers(t *testing.T) {
	f := newTestJWT(t)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:    "usr_1",
		AppID:     "aapp_1",
		SessionID: "ases_1",
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Empty(t, out.PrincipalKind)
	assert.Empty(t, out.PrincipalID)
	assert.Equal(t, "usr_1", out.UserID)
}
