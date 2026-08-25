package authsome_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/tokenformat"
)

// refreshDPoPEndpoint is the URL a DPoP proof for a refresh request must be
// bound to (htu). It doesn't need to be a real route since Engine.Refresh
// only compares it against the RequestURL carried in RefreshOpts.
const refreshDPoPEndpoint = "http://example.com/v1/refresh"

// ──────────────────────────────────────────────────
// Proof-minting helpers, copied from api/dpop_signin_test.go /
// plugins/oauth2provider/dpop_issuance_test.go rather than exported across
// packages.
// ──────────────────────────────────────────────────

func refreshDPoPKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

func refreshDPoPThumbprint(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""
	jkt, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	return jkt
}

// refreshDPoPProof mints a proof bound to POST against refreshDPoPEndpoint.
func refreshDPoPProof(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)

	claims := map[string]any{
		"jti": fmt.Sprintf("refresh-proof-%d", time.Now().UnixNano()),
		"htm": http.MethodPost,
		"htu": refreshDPoPEndpoint,
		"iat": time.Now().Unix(),
	}
	cb, err := json.Marshal(claims)
	require.NoError(t, err)

	signing := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(cb)

	method := jwt.GetSigningMethod("ES256")
	require.NotNil(t, method)
	sig, err := method.Sign(signing, key)
	require.NoError(t, err)

	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// refreshOptsForProof builds the RefreshOpts a real handler would construct
// for a request carrying the given raw DPoP header value.
func refreshOptsForProof(dpopHeader string) authsome.RefreshOpts {
	return authsome.RefreshOpts{
		DPoPProof:  dpopHeader,
		Method:     http.MethodPost,
		RequestURL: refreshDPoPEndpoint,
	}
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

// TestRefresh_UnboundSessionUnaffected is the regression guard: a session
// with no DPoP binding refreshes exactly as it did before this feature
// existed, proof or no proof.
func TestRefresh_UnboundSessionUnaffected(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "unbound-refresh@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Unbound User",
	})
	require.NoError(t, err)
	require.Empty(t, sess.DPoPJKT, "sanity: this session must not be bound")

	refreshed, err := eng.Refresh(ctx, sess.RefreshToken)
	require.NoError(t, err, "an unbound session must refresh with no DPoP header, exactly as today")
	assert.Empty(t, refreshed.DPoPJKT)
}

// TestRefresh_BoundSessionWithoutProofIsRefused: a session bound to key A
// presents no DPoP header on refresh. Without a proof there is nothing to
// check the binding against, so the refresh must be refused.
func TestRefresh_BoundSessionWithoutProofIsRefused(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	keyA := refreshDPoPKey(t)
	jktA := refreshDPoPThumbprint(t, keyA)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "bound-noproof@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Bound User",
		DPoPJKT:   jktA,
	})
	require.NoError(t, err)
	require.Equal(t, jktA, sess.DPoPJKT, "sanity: session must be bound to key A")

	_, err = eng.Refresh(ctx, sess.RefreshToken)
	assert.ErrorIs(t, err, account.ErrInvalidCredentials,
		"a bound session refreshed with no DPoP proof must be refused")
}

// TestRefresh_BoundSessionWrongKeyIsRefused: a session bound to key A
// presents a structurally valid proof signed by key B. This is the
// anti-laundering check itself: the whole point of stamping a binding at
// issuance is worthless if a refresh doesn't verify it belongs to the same
// key.
func TestRefresh_BoundSessionWrongKeyIsRefused(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	ch := secutil.NewBufferedChronicle()
	secutil.AttachChronicle(t, eng, ch)

	keyA := refreshDPoPKey(t)
	jktA := refreshDPoPThumbprint(t, keyA)
	keyB := refreshDPoPKey(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "bound-wrongkey@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Bound User",
		DPoPJKT:   jktA,
	})
	require.NoError(t, err)

	wrongProof := refreshDPoPProof(t, keyB)
	_, err = eng.Refresh(ctx, sess.RefreshToken, refreshOptsForProof(wrongProof))
	assert.ErrorIs(t, err, account.ErrInvalidCredentials,
		"a refresh proof signed by the wrong key must be refused")

	secutil.AssertAuditEvent(t, ch, "dpop_key_mismatch", func(ev *bridge.AuditEvent) {
		require.NotNil(t, ev)
		assert.Equal(t, jktA, ev.Metadata["bound_jkt"])
	})
}

// TestRefresh_BindingIsInherited is the anti-laundering property this task
// exists to close: a session bound to key A, refreshed with a valid proof
// for key A, must hand back a rotated session that is STILL bound to key A,
// both in the session row and in the re-minted JWT's own cnf claim.
//
// The row alone is not enough: middleware.tryJWTAuth (middleware/auth.go)
// validates a JWT-format access token statelessly from its own cnf claim, not
// from the session row in the store. A rotation that carried the thumbprint
// into the DB but not into the freshly-minted JWT would leave the row saying
// "bound" while the token that's actually presented on every subsequent
// request says "unbound", and the token wins, because that's the one
// middleware reads. That is exactly the laundering path this task is
// supposed to close, so the assertion below decodes the actual returned JWT
// rather than trusting the session struct.
func TestRefresh_BindingIsInherited(t *testing.T) {
	appIDStr := "aapp_01jf0000000000000000000000"
	signingKey := []byte("test-jwt-hmac-signing-key-at-least-32-bytes!!")
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    signingKey,
		VerifyKey:     signingKey,
	})
	require.NoError(t, err)

	eng, _ := newTestEngine(t, authsome.WithJWTFormat(appIDStr, jwtFmt))
	ctx := context.Background()
	appID := testAppID(t)

	keyA := refreshDPoPKey(t)
	jktA := refreshDPoPThumbprint(t, keyA)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "bound-inherit@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Bound User",
		DPoPJKT:   jktA,
	})
	require.NoError(t, err)
	require.True(t, tokenformat.IsJWT(sess.Token), "sanity: this app is JWT-configured")

	validProof := refreshDPoPProof(t, keyA)
	refreshed, err := eng.Refresh(ctx, sess.RefreshToken, refreshOptsForProof(validProof))
	require.NoError(t, err, "a proof from the bound key must be accepted")

	// The row: a rotation that dropped the binding would hand back an
	// unbound token, which is exactly the attack this task prevents.
	assert.Equal(t, jktA, refreshed.DPoPJKT, "the rotated session row must keep the original binding")

	// The token: decode the JWT actually handed back and check its own cnf
	// claim, not the session struct.
	require.True(t, tokenformat.IsJWT(refreshed.Token), "the refreshed access token must remain a JWT")
	claims, err := jwtFmt.ValidateAccessToken(refreshed.Token)
	require.NoError(t, err, "the re-minted JWT must validate against the app's signing key")
	assert.Equal(t, jktA, claims.DPoPJKT,
		"the re-minted JWT's own cnf.jkt claim must carry the thumbprint, not just the session row")
}
