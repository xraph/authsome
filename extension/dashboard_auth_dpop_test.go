package extension

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/store/memory"
)

// ──────────────────────────────────────────────────
// The dashboard auth checkers
//
// Both of them turn a token into a dashboard identity, which makes each of
// them a path from a credential to the highest-privilege surface in the
// product. Neither consulted the binding: the engine-mode checker ignored
// sess.DPoPJKT and the client-mode one read the introspection response and
// threw resp.Cnf away one struct field from where ClientAuthMiddleware reads
// it. Chain ordering was the only thing closing either gap, and ordering is
// not an invariant.
// ──────────────────────────────────────────────────

const dashCheckerURL = "http://example.com/dashboard"

// dashCheckerRequest builds the GET a dashboard page load looks like, with the
// token in the auth_token cookie the way a browser sends it, plus an optional
// proof.
func dashCheckerRequest(t *testing.T, token, proof string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, dashCheckerURL, nil)
	req.AddCookie(&http.Cookie{Name: dashboardCookieName, Value: token})
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	return req
}

// dashCheckerProof mints a proof covering the dashboard page load, with ath
// over the token the request carries.
func dashCheckerProof(t *testing.T, key *ecdsa.PrivateKey, token string) string {
	t.Helper()

	claims := dpoptest.ValidClaims(http.MethodGet, dashCheckerURL)
	claims["ath"] = dpop.AccessTokenHash(token)
	return dpoptest.MintProof(t, key, "ES256", claims)
}

// ──────────────────────────────────────────────────
// Client mode
// ──────────────────────────────────────────────────

// newDashClientChecker stands up a stub identity server that reports token as
// active and bound to jkt (empty for unbound), and returns a checker wired the
// way RegisterDashboardAuth wires one.
func newDashClientChecker(t *testing.T, token, jkt string) *clientAuthChecker {
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
			resp.User = &authclient.IntrospectUser{ID: userID, Email: "dash@test.com"}
			if jkt != "" {
				resp.Cnf = &authclient.IntrospectConfirmation{Jkt: jkt}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(identity.Close)

	return &clientAuthChecker{
		client:  authclient.NewClient(identity.URL),
		binding: middleware.SessionBindingConfig{DPoPValidator: dpop.NewValidator(dpop.Config{})},
		logger:  log.NewNoopLogger(),
	}
}

// TestClientAuthChecker_UnboundTokenIsUnaffected is the migration guard. A
// token introspected with no cnf claim resolves an identity exactly as before.
func TestClientAuthChecker_UnboundTokenIsUnaffected(t *testing.T) {
	t.Parallel()

	c := newDashClientChecker(t, "dash-unbound", "")

	info, err := c.CheckAuth(context.Background(), dashCheckerRequest(t, "dash-unbound", ""))

	require.NoError(t, err)
	assert.NotNil(t, info)
}

// TestClientAuthChecker_BoundTokenWithoutProofIsRefused is the gap. cnf said
// the token was bound and the checker handed back a dashboard identity anyway.
func TestClientAuthChecker_BoundTokenWithoutProofIsRefused(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	c := newDashClientChecker(t, "dash-bound", dpoptest.Thumbprint(t, key))

	info, err := c.CheckAuth(context.Background(), dashCheckerRequest(t, "dash-bound", ""))

	require.NoError(t, err)
	assert.Nil(t, info, "a bound token with no proof must not resolve a dashboard identity")
}

// TestClientAuthChecker_BoundTokenWithProofSucceeds keeps the refusal above
// from being a blanket lockout: present the key the token is bound to and the
// dashboard opens.
func TestClientAuthChecker_BoundTokenWithProofSucceeds(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	c := newDashClientChecker(t, "dash-bound-ok", dpoptest.Thumbprint(t, key))

	info, err := c.CheckAuth(context.Background(),
		dashCheckerRequest(t, "dash-bound-ok", dashCheckerProof(t, key, "dash-bound-ok")))

	require.NoError(t, err)
	assert.NotNil(t, info)
}

// TestClientAuthChecker_ProofFromAnotherKeyIsRefused: holding a DPoP key of
// your own does not let you use somebody else's stolen cookie.
func TestClientAuthChecker_ProofFromAnotherKeyIsRefused(t *testing.T) {
	t.Parallel()

	bound := dpoptest.Key(t)
	attacker := dpoptest.Key(t)
	c := newDashClientChecker(t, "dash-bound-wrong", dpoptest.Thumbprint(t, bound))

	info, err := c.CheckAuth(context.Background(),
		dashCheckerRequest(t, "dash-bound-wrong", dashCheckerProof(t, attacker, "dash-bound-wrong")))

	require.NoError(t, err)
	assert.Nil(t, info)
}

// ──────────────────────────────────────────────────
// Engine mode
// ──────────────────────────────────────────────────

const dashCheckerAppID = "aapp_01jf0000000000000000000000"

// newDashEngineChecker builds an engine over a memory store and returns a
// checker over it, plus a session bound to jkt (empty for unbound).
func newDashEngineChecker(t *testing.T, jkt, email string) (*authChecker, string) {
	t.Helper()

	s := memory.New()
	appID, err := id.ParseAppID(dashCheckerAppID)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID:         appID,
		Name:       "Platform",
		Slug:       "platform",
		IsPlatform: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}))

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(dashCheckerAppID),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  "SecureP@ss1",
		FirstName: "Dash",
		DPoPJKT:   jkt,
	})
	require.NoError(t, err)

	return &authChecker{engine: eng}, sess.Token
}

// TestAuthChecker_UnboundSessionIsUnaffected is the migration guard for the
// engine-mode checker.
func TestAuthChecker_UnboundSessionIsUnaffected(t *testing.T) {
	t.Parallel()

	c, token := newDashEngineChecker(t, "", "dash-unbound@example.com")

	info, err := c.CheckAuth(context.Background(), dashCheckerRequest(t, token, ""))

	require.NoError(t, err)
	assert.NotNil(t, info)
}

// TestAuthChecker_BoundSessionWithoutProofIsRefused is the gap. sess.DPoPJKT
// was never read, so a stolen dashboard cookie opened the dashboard.
func TestAuthChecker_BoundSessionWithoutProofIsRefused(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	c, token := newDashEngineChecker(t, dpoptest.Thumbprint(t, key), "dash-bound@example.com")

	info, err := c.CheckAuth(context.Background(), dashCheckerRequest(t, token, ""))

	require.NoError(t, err)
	assert.Nil(t, info, "a bound session with no proof must not resolve a dashboard identity")
}

// TestAuthChecker_BoundSessionWithProofSucceeds: the holder of the key still
// gets in, over the cookie transport a browser actually uses.
func TestAuthChecker_BoundSessionWithProofSucceeds(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	c, token := newDashEngineChecker(t, dpoptest.Thumbprint(t, key), "dash-bound-ok@example.com")

	info, err := c.CheckAuth(context.Background(),
		dashCheckerRequest(t, token, dashCheckerProof(t, key, token)))

	require.NoError(t, err)
	assert.NotNil(t, info)
}

// TestAuthChecker_ProofFromAnotherKeyIsRefused pins the key comparison rather
// than mere presence of a DPoP header.
func TestAuthChecker_ProofFromAnotherKeyIsRefused(t *testing.T) {
	t.Parallel()

	bound := dpoptest.Key(t)
	attacker := dpoptest.Key(t)
	c, token := newDashEngineChecker(t, dpoptest.Thumbprint(t, bound), "dash-bound-wrong@example.com")

	info, err := c.CheckAuth(context.Background(),
		dashCheckerRequest(t, token, dashCheckerProof(t, attacker, token)))

	require.NoError(t, err)
	assert.Nil(t, info)
}
