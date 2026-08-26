package sharedsignals

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/riskengine"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// This file is the one test in the package that stands up a real engine
// rather than a stub or a bare store: it registers the plugin the way a
// host application would, points a fake identity provider at the resulting
// push endpoint, and checks the result by reading the session store back --
// not by trusting a status code. Every other test in this package pokes one
// layer; this one is the wiring proof.
//
// It lives in package sharedsignals (not sharedsignals_test) for one
// concrete reason: after secutil.NewTestEngine has run the plugin through a
// real OnInit, the jwks client it built talks to jwksclient.ValidateURI's
// production rule of "https on a public address", which a loopback
// httptest.Server correctly fails. receiver_test.go's fixture works around
// this by overriding the unexported p.jwks field after OnInit runs; being in
// this package is what makes that override possible without adding any new
// exported production surface for it.

// e2eTransmitter is an identity provider: a keypair, a JWKS endpoint and the
// ability to sign a SET the way Okta does. The JWKS endpoint runs over TLS --
// unlike receiver_test.go's fixture, which reaches p.jwks directly and only
// needs to satisfy jwksclient's own (overridable) ValidateURI, this test also
// exercises CreateStream, and CreateStream's validateStreamState calls the
// package-level jwksclient.ValidateURI unconditionally as an SSRF guard on
// stream registration. That check requires https, so a plain-HTTP loopback
// server (as receiver_test.go uses) will not clear it.
type e2eTransmitter struct {
	key    *rsa.PrivateKey
	server *httptest.Server
	// jwksURI is the server's URL with its host rewritten from 127.0.0.1 to
	// localhost. jwksclient.ValidateURI parses the IP literal out of the
	// host and rejects loopback addresses; "localhost" is not a literal IP,
	// so it clears that check while still resolving straight back to the
	// same loopback server for the actual fetch.
	jwksURI string
	issuer  string
}

func newE2ETransmitter(t *testing.T) *e2eTransmitter {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"k1","n":%q,"e":%q}]}`, n, e)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)
	u.Host = "localhost:" + port

	return &e2eTransmitter{key: key, server: srv, jwksURI: u.String(), issuer: "https://idp.test.example"}
}

// sessionRevokedSET signs a session-revoked event in Okta's shape, meaning
// the subject arrives under "subject" rather than "sub_id".
func (f *e2eTransmitter) sessionRevokedSET(t *testing.T, audience, subject string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": f.issuer,
		"aud": audience,
		"jti": "jti-" + id.NewSSFEventID().String(),
		"iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": f.issuer, "sub": subject,
				},
				"reason_admin":    map[string]any{"en": "Account compromised"},
				"event_timestamp": time.Now().UnixMilli(),
			},
		},
	})
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = "k1"
	signed, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// newE2EEngine stands up a real engine with the sharedsignals plugin
// registered exactly the way a host application would (authsome.WithPlugin),
// then swaps in a permissive jwks client so the fake transmitter's loopback
// JWKS endpoint is reachable in a test -- see the package doc comment above
// for why that override needs this file to live in-package.
func newE2EEngine(t *testing.T) (*authsome.Engine, *Plugin, id.AppID) {
	t.Helper()
	ssf := New(Config{Audience: "https://authsome.test/ssf"})
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(ssf))

	// The transmitter's certificate is self-signed and issued for
	// 127.0.0.1/::1, not for "localhost" -- InsecureSkipVerify sidesteps
	// hostname/chain verification for this test client only. ValidateURI is
	// also relaxed here so the actual fetch (as opposed to CreateStream's
	// separate, unconditional SSRF guard -- see e2eTransmitter's doc
	// comment) accepts the loopback address.
	ssf.jwks = jwksclient.New(jwksclient.Options{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		ValidateURI: func(string) error { return nil },
	})

	appID, err := id.ParseAppID(eng.DefaultAppID())
	require.NoError(t, err)
	return eng, ssf, appID
}

// createE2EVictim seeds a user with the given number of live sessions,
// through the engine's own store the way real sign-ins would populate it.
func createE2EVictim(ctx context.Context, t *testing.T, eng *authsome.Engine,
	appID id.AppID, sessions int) *user.User {
	t.Helper()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID,
		Email: "victim@corp.com", EmailVerified: true,
	}
	require.NoError(t, eng.Store().CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID,
		Email: u.Email, Verified: true, IsPrimary: true,
	}))
	for i := 0; i < sessions; i++ {
		require.NoError(t, eng.Store().CreateSession(ctx, &session.Session{
			ID: id.NewSessionID(), AppID: appID, UserID: u.ID,
			Token:     fmt.Sprintf("live-token-%d-%d", sessions, i),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}))
	}
	return u
}

func e2eRiskRequest(appID id.AppID, email string) *riskengine.RiskRequest {
	return &riskengine.RiskRequest{AppID: appID.String(), Email: email}
}

// The whole point of the feature, in one test: an upstream compromise ends
// the sessions authsome issued. This is the test that would catch the whole
// feature being wired up wrong -- a real engine, a real plugin registration,
// a real (signed) SET, and an assertion made by reading the session store
// back, not by trusting the HTTP response code.
func TestEndToEnd_UpstreamRevocationKillsLiveSessions(t *testing.T) {
	ctx := context.Background()
	idp := newE2ETransmitter(t)

	eng, ssf, appID := newE2EEngine(t)

	// A user with two live sessions.
	u := createE2EVictim(ctx, t, eng, appID, 2)

	live, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, live, 2, "the user starts with two live sessions")

	// Register the IdP as a stream and link its subject to our user, which is
	// what an SSO sign-in would have done.
	created, err := ssf.CreateStream(ctx, appID, id.Nil, CreateStreamRequest{
		Name:    "test-idp",
		Issuer:  idp.issuer,
		JWKSURI: idp.jwksURI,
	})
	require.NoError(t, err)
	require.NoError(t, ssf.LinkSubject(ctx, appID, id.Nil,
		idp.issuer, "idp-user-1", u.ID, SourceSSO))

	// The IdP decides the account is compromised and pushes the event.
	body := idp.sessionRevokedSET(t, "https://authsome.test/ssf", "idp-user-1")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/ssf/streams/"+created.PushURLPath+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	req.Header.Set("Authorization", "Bearer "+created.PushToken)
	rec := httptest.NewRecorder()

	ssf.servePushForTest(rec, req, created.PushURLPath)
	require.Equal(t, http.StatusAccepted, rec.Code, "body: %s", rec.Body.String())

	// The sessions are gone from the store, not merely marked.
	after, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, after, "an upstream revocation must end every live session")

	// And a durable signal is left behind for the next sign-in to score.
	signal, err := ssf.EvaluateRisk(ctx, e2eRiskRequest(appID, u.Email))
	require.NoError(t, err)
	// The signal is deliberately capped into riskengine's challenge band:
	// a confirmed compromise steps the next sign-in up, it does not bar it.
	assert.GreaterOrEqual(t, signal.Score, 60,
		"the revocation must leave a signal strong enough to challenge")
	assert.Less(t, signal.Score, 85,
		"but not strong enough to block the sign-in outright")

	// A valid SET replayed a second time must not revoke anything a second
	// time -- there is nothing left to revoke, and the dedupe row must
	// answer 202 rather than reprocessing the event.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/ssf/streams/"+created.PushURLPath+"/events", stringReader(body))
	req2.Header.Set("Content-Type", "application/secevent+jwt")
	req2.Header.Set("Authorization", "Bearer "+created.PushToken)
	ssf.servePushForTest(rec2, req2, created.PushURLPath)
	assert.Equal(t, http.StatusAccepted, rec2.Code, "a replay is a success, not an error")

	stillGone, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, stillGone, "a replayed SET must not resurrect or otherwise disturb sessions")
}

// A SET signed by a key the stream does not trust changes nothing: the
// signature check fails before any subject is even resolved, so the
// session survives and the transmitter is told this was rejected, not
// accepted.
func TestEndToEnd_ForgedSETLeavesSessionsAlone(t *testing.T) {
	ctx := context.Background()
	realIDP := newE2ETransmitter(t)
	attacker := newE2ETransmitter(t)
	attacker.issuer = realIDP.issuer // same issuer claim, different key

	eng, ssf, appID := newE2EEngine(t)

	u := createE2EVictim(ctx, t, eng, appID, 1)

	created, err := ssf.CreateStream(ctx, appID, id.Nil, CreateStreamRequest{
		Name: "test-idp", Issuer: realIDP.issuer, JWKSURI: realIDP.jwksURI,
	})
	require.NoError(t, err)
	require.NoError(t, ssf.LinkSubject(ctx, appID, id.Nil,
		realIDP.issuer, "idp-user-1", u.ID, SourceSSO))

	// Signed by the attacker's key, but the stream only trusts the real IdP's
	// JWKS, so the signature check fails.
	body := attacker.sessionRevokedSET(t, "https://authsome.test/ssf", "idp-user-1")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/ssf/streams/"+created.PushURLPath+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	req.Header.Set("Authorization", "Bearer "+created.PushToken)
	rec := httptest.NewRecorder()

	ssf.servePushForTest(rec, req, created.PushURLPath)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	after, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, after, 1, "a forged SET must leave every session alone")
}

// The durable half of the feature only pays off if the risk engine actually
// asks us. It never did: riskengine only accepted contributors through New,
// NewWithConfig and AddContributor, and nothing in the tree ever handed it
// this plugin, so every stored signal sat in the table unread. This test
// registers both plugins the way a host application would and then makes the
// risk engine score a sign-in, with no manual wiring in between.
func TestEndToEnd_StoredSignalReachesTheRiskEngine(t *testing.T) {
	ctx := context.Background()
	idp := newE2ETransmitter(t)

	ssf := New(Config{Audience: "https://authsome.test/ssf"})
	// A deliberately low block threshold, so "was the contributor reached?"
	// stays observable through OnBeforeSignIn. The default policy is asserted
	// separately below.
	risk := riskengine.NewWithConfig(riskengine.Config{HighThreshold: 70})
	eng := secutil.NewTestEngine(t,
		authsome.WithPlugin(ssf), authsome.WithPlugin(risk))

	ssf.jwks = jwksclient.New(jwksclient.Options{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		ValidateURI: func(string) error { return nil },
	})

	appID, err := id.ParseAppID(eng.DefaultAppID())
	require.NoError(t, err)
	u := createE2EVictim(ctx, t, eng, appID, 1)

	created, err := ssf.CreateStream(ctx, appID, id.Nil, CreateStreamRequest{
		Name: "test-idp", Issuer: idp.issuer, JWKSURI: idp.jwksURI,
	})
	require.NoError(t, err)
	require.NoError(t, ssf.LinkSubject(ctx, appID, id.Nil,
		idp.issuer, "idp-user-1", u.ID, SourceSSO))

	// A sign-in before anything has happened is unremarkable.
	require.NoError(t, risk.OnBeforeSignIn(ctx, &account.SignInRequest{
		AppID: appID, Email: u.Email, IPAddress: "203.0.113.10",
	}), "a user with no signals must not be blocked")

	// The IdP reports the compromise.
	body := idp.sessionRevokedSET(t, "https://authsome.test/ssf", "idp-user-1")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/ssf/streams/"+created.PushURLPath+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	req.Header.Set("Authorization", "Bearer "+created.PushToken)
	rec := httptest.NewRecorder()
	ssf.servePushForTest(rec, req, created.PushURLPath)
	require.Equal(t, http.StatusAccepted, rec.Code, "body: %s", rec.Body.String())

	// Now the same sign-in is scored on the signal that event left behind,
	// through a contributor nobody wired by hand.
	err = risk.OnBeforeSignIn(ctx, &account.SignInRequest{
		AppID: appID, Email: u.Email, IPAddress: "203.0.113.10",
	})
	require.Error(t, err,
		"a session-revoked signal must reach the risk engine through a contributor nobody wired by hand")
	assert.Contains(t, err.Error(), "riskengine")

	// And with the shipped defaults the same signal challenges rather than
	// blocking, so the user can still authenticate and be stepped up.
	defaults := riskengine.New(ssf)
	require.NoError(t, defaults.OnBeforeSignIn(ctx, &account.SignInRequest{
		AppID: appID, Email: u.Email, IPAddress: "203.0.113.10",
	}), "on default thresholds a confirmed compromise must challenge, not block")
}
