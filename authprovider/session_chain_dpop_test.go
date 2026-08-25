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
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	authmw "github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Two enforcement points, one request.
//
// Nothing else in the suite runs the global auth middleware and the registry
// middleware in a single chain, and that is the shape a real deployment has:
// extension.Middlewares() applies AuthMiddleware to everything, and routes
// declared with forge.WithAuth("session") add RegistryMiddleware on top. Both
// call into RFC 9449 enforcement, both share the engine's one validator and
// therefore its one replay cache, so a proof validated by the first was being
// reported as a replay by the second and the route answered 401 with a
// perfectly good proof.
//
// Every test below runs the same chain. What varies is what the client
// presents, so a fix that simply stopped enforcing at the second point would
// fail the negative cases.
// ──────────────────────────────────────────────────

const chainToken = "chain-bound-token"

const chainURL = "http://example.com/chained"

// chainSession is the one bound session both enforcement points resolve.
func chainSession(jkt string) *session.Session {
	return &session.Session{
		ID:        id.NewSessionID(),
		AppID:     id.NewAppID(),
		UserID:    id.NewUserID(),
		Token:     chainToken,
		ExpiresAt: time.Now().Add(time.Hour),
		DPoPJKT:   jkt,
	}
}

// chainRouter wires AuthMiddleware and RegistryMiddleware into one chain over
// a single shared SessionBindingConfig, the way Engine.buildAuthMiddleware and
// Engine.registerSessionAuthProvider both take theirs from
// Engine.DPoPBindingConfig(). Sharing it is the point: the validator, and so
// the replay cache behind it, is one object seen by both.
func chainRouter(t *testing.T, sess *session.Session, bind authmw.SessionBindingConfig) http.Handler {
	t.Helper()

	u := &user.User{ID: sess.UserID, AppID: sess.AppID, Email: "chain@test.com"}

	resolveSession := func(token string) (*session.Session, error) {
		if token == sess.Token {
			return sess, nil
		}
		return nil, errors.New("invalid token")
	}
	resolveUser := func(userIDStr string) (*user.User, error) {
		if userIDStr == sess.UserID.String() {
			return u, nil
		}
		return nil, errors.New("user not found")
	}

	provider := authprovider.
		NewSessionProvider(resolveSession, resolveUser, log.NewNoopLogger()).
		WithDPoP(bind)

	registry := auth.NewRegistry(nil, forge.NewNoopLogger())
	require.NoError(t, registry.Register(provider))

	router := forge.NewRouter()
	router.Use(authmw.AuthMiddleware(resolveSession, resolveUser, log.NewNoopLogger(), bind))
	router.Use(authprovider.RegistryMiddleware(registry, "session"))
	router.GET("/chained", func(ctx forge.Context) error {
		if _, ok := authmw.UserFrom(ctx.Context()); !ok {
			return forge.InternalError(errors.New("handler reached without user context"))
		}
		return ctx.NoContent(http.StatusOK)
	})

	return router.Handler()
}

// chainRequest builds a GET carrying the token under scheme, with an optional
// proof header.
func chainRequest(t *testing.T, scheme, proof string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, chainURL, nil)
	req.Header.Set("Authorization", scheme+" "+chainToken)
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	return req
}

// chainProof mints a proof for the chained route, carrying ath over the token
// the request presents. jti is a parameter so a test can send the same proof
// twice or two distinct proofs.
func chainProof(t *testing.T, key *ecdsa.PrivateKey, jti string) string {
	t.Helper()

	claims := dpoptest.ValidClaims(http.MethodGet, chainURL)
	claims["jti"] = jti
	claims["ath"] = dpop.AccessTokenHash(chainToken)
	return dpoptest.MintProof(t, key, "ES256", claims)
}

// TestChain_BoundSessionPassesBothEnforcementPoints is the regression this
// wave exists for. Before the per-request proof scope, AuthMiddleware recorded
// the proof's jti in the shared replay cache and the registry middleware's
// second validation of the very same proof failed as ErrReplayed, so this
// returned 401.
func TestChain_BoundSessionPassesBothEnforcementPoints(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	sess := chainSession(dpoptest.Thumbprint(t, key))
	bind := authmw.SessionBindingConfig{DPoPValidator: dpop.NewValidator(dpop.Config{})}

	rec := httptest.NewRecorder()
	chainRouter(t, sess, bind).ServeHTTP(rec, chainRequest(t, "DPoP", chainProof(t, key, "chain-proof-1")))

	assert.Equal(t, http.StatusOK, rec.Code,
		"one proof must satisfy both enforcement points on one request; body: %s", rec.Body.String())
}

// TestChain_BoundSessionWithoutProofIsStillRefused proves the fix did not turn
// the second enforcement point off. Nothing validates here, so nothing is
// recorded on the request scope, and both points refuse.
func TestChain_BoundSessionWithoutProofIsStillRefused(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	sess := chainSession(dpoptest.Thumbprint(t, key))
	bind := authmw.SessionBindingConfig{DPoPValidator: dpop.NewValidator(dpop.Config{})}

	rec := httptest.NewRecorder()
	chainRouter(t, sess, bind).ServeHTTP(rec, chainRequest(t, "DPoP", ""))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestChain_ProofFromAnotherKeyIsRefused covers the case where the request
// scope exists (a proof is present) but the proof does not match the binding.
// A scope that recorded before validating rather than after would let this
// through at the second point.
func TestChain_ProofFromAnotherKeyIsRefused(t *testing.T) {
	t.Parallel()

	bound := dpoptest.Key(t)
	attacker := dpoptest.Key(t)
	sess := chainSession(dpoptest.Thumbprint(t, bound))
	bind := authmw.SessionBindingConfig{DPoPValidator: dpop.NewValidator(dpop.Config{})}

	rec := httptest.NewRecorder()
	chainRouter(t, sess, bind).ServeHTTP(rec, chainRequest(t, "DPoP", chainProof(t, attacker, "chain-proof-wrong-key")))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestChain_ReplayedProofOnASecondRequestIsRefused is the property the fix
// must not cost us. The scope lives in one request's context, so a proof
// captured and re-sent arrives with nothing recorded and meets the replay
// cache exactly as it did before.
func TestChain_ReplayedProofOnASecondRequestIsRefused(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	sess := chainSession(dpoptest.Thumbprint(t, key))
	bind := authmw.SessionBindingConfig{DPoPValidator: dpop.NewValidator(dpop.Config{})}
	handler := chainRouter(t, sess, bind)

	proof := chainProof(t, key, "chain-proof-replayed")

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, chainRequest(t, "DPoP", proof))
	require.Equal(t, http.StatusOK, first.Code, "body: %s", first.Body.String())

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, chainRequest(t, "DPoP", proof))

	assert.Equal(t, http.StatusUnauthorized, second.Code,
		"the same proof on a second request is a replay and must be refused")
}
