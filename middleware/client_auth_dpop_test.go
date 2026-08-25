package middleware_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
)

// ──────────────────────────────────────────────────
// Client mode (remote introspection) enforcement
//
// A client-mode service resolves tokens by calling the identity server's
// introspection endpoint, so the binding reaches it as the cnf claim in the
// introspection response (RFC 9449 section 7.3) rather than as a session row.
// The rule it applies is the same one middleware/auth.go applies, because
// enforcement is a property of the token and not of which service holds it.
// ──────────────────────────────────────────────────

// clientAuthTestSetup stands up a fake identity server whose introspection
// endpoint reports token as active and bound to jkt (empty for unbound), then
// wires ClientAuthMiddleware against it.
func clientAuthTestSetup(t *testing.T, token, jkt string, bind ...middleware.SessionBindingConfig) forge.Router {
	t.Helper()

	userID := id.NewUserID().String()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		resp := authclient.IntrospectResponse{}
		if body.Token == token {
			resp.Active = true
			resp.UserID = userID
			resp.AppID = id.NewAppID().String()
			resp.SessionID = id.NewSessionID().String()
			resp.User = &authclient.IntrospectUser{ID: userID, Email: "dpop-client@test.com"}
			if jkt != "" {
				resp.Cnf = &authclient.IntrospectConfirmation{Jkt: jkt}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(identity.Close)

	client := authclient.NewClient(identity.URL)

	router := forge.NewRouter()
	router.Use(middleware.ClientAuthMiddleware(client, log.NewNoopLogger(), bind...))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})
	return router
}

// clientAuthRequest issues a GET /test carrying token under scheme, with an
// optional proof header.
func clientAuthRequest(t *testing.T, router forge.Router, scheme, token, proof string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", scheme+" "+token)
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// boundToken is the token every test in this file presents. The stub identity
// server reports it as bound; anything else introspects as inactive.
const boundToken = "bound-token"

// clientAuthProof mints a proof for the GET /test route bound to boundToken.
func clientAuthProof(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()

	claims := dpoptest.ValidClaims(http.MethodGet, "http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash(boundToken)
	return dpoptest.MintProof(t, key, "ES256", claims)
}

// TestClientAuthDPoP_UnboundTokenIsUnaffected is the regression guard. A token
// introspected without a cnf claim takes the path it took before enforcement
// existed.
func TestClientAuthDPoP_UnboundTokenIsUnaffected(t *testing.T) {
	router := clientAuthTestSetup(t, "unbound-token", "")

	rec := clientAuthRequest(t, router, "Bearer", "unbound-token", "")

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestClientAuthDPoP_BoundTokenWithoutProofIs401 is the gap this closes. A
// stolen bound token presented to a client-mode service used to be honoured
// with no proof at all.
func TestClientAuthDPoP_BoundTokenWithoutProofIs401(t *testing.T) {
	key := dpoptest.Key(t)
	router := clientAuthTestSetup(t, boundToken, dpoptest.Thumbprint(t, key), middleware.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	rec := clientAuthRequest(t, router, "DPoP", boundToken, "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a bound token with no proof must be refused in client mode")
}

// TestClientAuthDPoP_BoundTokenUnderBearerSchemeIs401 mirrors the strict
// scheme matching the engine path applies.
func TestClientAuthDPoP_BoundTokenUnderBearerSchemeIs401(t *testing.T) {
	key := dpoptest.Key(t)
	router := clientAuthTestSetup(t, boundToken, dpoptest.Thumbprint(t, key), middleware.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	rec := clientAuthRequest(t, router, "Bearer", boundToken, clientAuthProof(t, key))

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a bound token presented as Bearer must be refused")
}

// TestClientAuthDPoP_WrongKeyIs401 covers the attacker who holds the token and
// mints proofs with a key of their own.
func TestClientAuthDPoP_WrongKeyIs401(t *testing.T) {
	bound := dpoptest.Key(t)
	attacker := dpoptest.Key(t)
	router := clientAuthTestSetup(t, boundToken, dpoptest.Thumbprint(t, bound), middleware.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	rec := clientAuthRequest(t, router, "DPoP", boundToken, clientAuthProof(t, attacker))

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a proof for the wrong key must be refused")
}

// TestClientAuthDPoP_ValidProofPasses proves enforcement does not break the
// legitimate client that actually holds the key.
func TestClientAuthDPoP_ValidProofPasses(t *testing.T) {
	key := dpoptest.Key(t)
	router := clientAuthTestSetup(t, boundToken, dpoptest.Thumbprint(t, key), middleware.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})

	rec := clientAuthRequest(t, router, "DPoP", boundToken, clientAuthProof(t, key))

	assert.Equal(t, http.StatusOK, rec.Code, "a correct proof must pass in client mode")
}

// TestClientAuthDPoP_NoValidatorFailsClosed covers the service that upgraded
// without wiring a validator. It cannot check the binding, so it refuses
// rather than admitting the token unchecked.
func TestClientAuthDPoP_NoValidatorFailsClosed(t *testing.T) {
	key := dpoptest.Key(t)
	router := clientAuthTestSetup(t, boundToken, dpoptest.Thumbprint(t, key))

	rec := clientAuthRequest(t, router, "DPoP", boundToken, clientAuthProof(t, key))

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a bound token with no validator configured must be refused")
}
