package authsome_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/tokenformat"
)

// ──────────────────────────────────────────────────
// dpop.nonce_required must not resolve to false quietly
//
// The old resolver answered false whenever no nonce signer existed, so an
// operator who switched nonces on for a deployment with no HMAC JWT key and no
// AUTHSOME_DASHBOARD_NONCE_SECRET got one Warn line at startup and a control
// that was on in the dashboard and off in reality. These tests pin that the
// misconfiguration now fails closed instead.
// ──────────────────────────────────────────────────

// setNonceRequired switches dpop.nonce_required on at global scope, which is
// what an operator does for a single-tenant deployment. The bare test engine
// has no platform app row, so app scope has nothing to hang off and global is
// the scope that actually resolves here.
func setNonceRequired(t *testing.T, eng *authsome.Engine, on bool) {
	t.Helper()

	mgr := eng.Settings()
	require.NotNil(t, mgr)
	raw, err := json.Marshal(on)
	require.NoError(t, err)
	require.NoError(t, mgr.Set(context.Background(), "dpop.nonce_required", raw,
		settings.ScopeGlobal, "", "", "", "test"))
}

// TestDPoPNonceRequired_DefaultsToFalse is the migration guard. Untouched
// setting, no signer, and nothing changes for anybody.
func TestDPoPNonceRequired_DefaultsToFalse(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	require.Nil(t, eng.DPoPNonceSigner())

	assert.False(t, eng.DPoPNonceRequiredForApp(context.Background(), eng.PlatformAppID()))
}

// TestDPoPNonceRequired_WithoutSignerFailsClosed is the finding. The setting
// is on, no secret is derivable, and the resolver used to answer false and
// disable the control. It must answer true so the requests it was meant to
// protect are refused rather than served unprotected.
func TestDPoPNonceRequired_WithoutSignerFailsClosed(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	require.Nil(t, eng.DPoPNonceSigner(), "this test is only meaningful with no derivable secret")

	setNonceRequired(t, eng, true)

	assert.True(t, eng.DPoPNonceRequiredForApp(context.Background(), eng.PlatformAppID()),
		"nonces switched on with nothing to mint them from is a misconfiguration, not permission to stop enforcing")
}

// TestDPoPNonceRequired_WithSignerIsHonoured keeps the working case working: a
// deployment that can mint nonces resolves the setting the ordinary way.
func TestDPoPNonceRequired_WithSignerIsHonoured(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-jwt-hmac-signing-key-at-least-32-bytes!!"),
	})
	require.NoError(t, err)

	eng := secutil.NewTestEngine(t, authsome.WithJWTFormat("some-app", jwtFmt))
	require.NotNil(t, eng.DPoPNonceSigner())

	ctx := context.Background()
	assert.False(t, eng.DPoPNonceRequiredForApp(ctx, eng.PlatformAppID()))

	setNonceRequired(t, eng, true)
	assert.True(t, eng.DPoPNonceRequiredForApp(ctx, eng.PlatformAppID()))
}
