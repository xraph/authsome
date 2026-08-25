package authsome_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/tokenformat"
)

func TestEngine_DPoPDefaultsToOff(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	require.NotNil(t, eng.DPoPValidator(), "a validator must always exist so nothing has to nil-check it")
	assert.Equal(t, dpop.ModeOff, eng.DPoPModeForApp(context.Background(), eng.PlatformAppID()))
}

// TestEngine_DPoPNonceSignerAbsentByDefault pins the real default for a bare
// secutil test engine: no HMAC JWT format, no AUTHSOME_DASHBOARD_NONCE_SECRET,
// so Engine.NonceSecret has nothing to derive from and DPoPNonceSigner must be
// nil. This is the direction that matters most, because it's what every test
// in this suite runs under and it's the one that has to fail closed.
func TestEngine_DPoPNonceSignerAbsentByDefault(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	assert.Nil(t, eng.DPoPNonceSigner(), "no HMAC JWT key and no env override means no derivable secret")
}

// TestEngine_DPoPNonceSignerPresentWhenSecretAvailable configures the test
// engine with an HMAC JWT format via the option the codebase actually
// provides (WithJWTFormat), the same input Engine.NonceSecret reads from in
// production. This is the real "secret available" branch: unlike
// secutil.InitTestNonceSigner (which only touches the unrelated dashboard
// package's own nonce signer), this actually puts a key where NonceSecret
// looks for one.
func TestEngine_DPoPNonceSignerPresentWhenSecretAvailable(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-jwt-hmac-signing-key-at-least-32-bytes!!"),
	})
	require.NoError(t, err)

	eng := secutil.NewTestEngine(t, authsome.WithJWTFormat("some-app", jwtFmt))

	s := eng.DPoPNonceSigner()
	require.NotNil(t, s, "an HMAC JWT format gives NonceSecret a key to derive the DPoP nonce secret from")

	n := s.Issue("jkt-abc")
	assert.True(t, s.Verify("jkt-abc", n), "a nonce must verify for the key it was issued to")
	assert.False(t, s.Verify("jkt-different", n), "a nonce must not verify for a different key")
}
