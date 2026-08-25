package authprovider_test

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	authmw "github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// SessionProvider is a second path from a token to an authenticated context,
// registered with the forge auth registry and consumed by plugin authz and
// several plugins. In a standard deployment the global auth middleware rejects
// a bad presentation before this runs, but the invariant is that enforcement
// follows the token, so it holds here too and does not depend on some other
// middleware having run first.
// ──────────────────────────────────────────────────

const dpopProviderToken = "bound-provider-token"

// newBoundProvider builds a SessionProvider over one session bound to jkt
// (empty for an unbound session).
func newBoundProvider(t *testing.T, jkt string, bind ...authmw.SessionBindingConfig) *authprovider.SessionProvider {
	t.Helper()

	sess := &session.Session{
		ID:        id.NewSessionID(),
		AppID:     id.NewAppID(),
		UserID:    id.NewUserID(),
		Token:     dpopProviderToken,
		ExpiresAt: time.Now().Add(time.Hour),
		DPoPJKT:   jkt,
	}
	u := &user.User{ID: sess.UserID, AppID: sess.AppID, Email: "dpop-provider@test.com"}

	p := authprovider.NewSessionProvider(
		func(token string) (*session.Session, error) {
			if token == sess.Token {
				return sess, nil
			}
			return nil, errors.New("invalid token")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == sess.UserID.String() {
				return u, nil
			}
			return nil, errors.New("user not found")
		},
		log.NewNoopLogger(),
	)
	if len(bind) > 0 {
		p = p.WithDPoP(bind[0])
	}
	return p
}

// providerRequest builds a GET carrying the token under scheme, with an
// optional proof header.
func providerRequest(t *testing.T, scheme, proof string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", scheme+" "+dpopProviderToken)
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	return req
}

// providerProof mints a proof covering the request providerRequest builds.
func providerProof(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()

	claims := dpoptest.ValidClaims(http.MethodGet, "http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash(dpopProviderToken)
	return dpoptest.MintProof(t, key, "ES256", claims)
}

// TestSessionProvider_UnboundTokenIsUnaffected is the regression guard: a
// session with no binding authenticates exactly as it did before.
func TestSessionProvider_UnboundTokenIsUnaffected(t *testing.T) {
	t.Parallel()

	p := newBoundProvider(t, "")

	authCtx, err := p.Authenticate(context.Background(), providerRequest(t, "Bearer", ""))

	require.NoError(t, err)
	assert.NotNil(t, authCtx)
}

// TestSessionProvider_BoundTokenWithoutProofIsRefused is the gap this closes.
func TestSessionProvider_BoundTokenWithoutProofIsRefused(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	p := newBoundProvider(t, dpoptest.Thumbprint(t, key), authmw.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	_, err := p.Authenticate(context.Background(), providerRequest(t, "DPoP", ""))

	require.Error(t, err, "a bound token with no proof must not authenticate")
}

// TestSessionProvider_BoundTokenUnderBearerSchemeIsRefused mirrors the strict
// scheme matching the middleware applies.
func TestSessionProvider_BoundTokenUnderBearerSchemeIsRefused(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	p := newBoundProvider(t, dpoptest.Thumbprint(t, key), authmw.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	_, err := p.Authenticate(context.Background(), providerRequest(t, "Bearer", providerProof(t, key)))

	require.Error(t, err, "a bound token presented as Bearer must not authenticate")
}

// TestSessionProvider_WrongKeyIsRefused covers the attacker holding the token
// but not the key.
func TestSessionProvider_WrongKeyIsRefused(t *testing.T) {
	t.Parallel()

	bound := dpoptest.Key(t)
	attacker := dpoptest.Key(t)
	p := newBoundProvider(t, dpoptest.Thumbprint(t, bound), authmw.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	_, err := p.Authenticate(context.Background(), providerRequest(t, "DPoP", providerProof(t, attacker)))

	require.Error(t, err, "a proof for the wrong key must not authenticate")
}

// TestSessionProvider_ValidProofAuthenticates proves the legitimate client
// still gets through this path.
func TestSessionProvider_ValidProofAuthenticates(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	p := newBoundProvider(t, dpoptest.Thumbprint(t, key), authmw.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	authCtx, err := p.Authenticate(context.Background(), providerRequest(t, "DPoP", providerProof(t, key)))

	require.NoError(t, err)
	assert.NotNil(t, authCtx)
}

// TestSessionProvider_NoValidatorFailsClosed covers a provider that was never
// handed a validator. It cannot check the binding, so it refuses.
func TestSessionProvider_NoValidatorFailsClosed(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	p := newBoundProvider(t, dpoptest.Thumbprint(t, key))

	_, err := p.Authenticate(context.Background(), providerRequest(t, "DPoP", providerProof(t, key)))

	require.Error(t, err, "a bound token with no validator configured must not authenticate")
}
