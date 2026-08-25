package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// TestSignIn_MFAChallenge_UnderRequiredMode_IssuesBoundSession is the
// regression test for the sharpest case the adversarial review found:
// enabling MFA silently disabled DPoP for that user, so the more secure
// configuration produced the weaker token.
//
// The mechanism was that sign-in proves a thumbprint but IssueSession returns
// MFARequiredError before minting anything. The MFA plugin then completed the
// ceremony and issued its own session with no thumbprint at all — and because
// enforcement follows the token (an empty jkt means the middleware demands
// nothing), that session was exempt from proof-of-possession for its whole
// life, on an app configured to require it.
//
// So this drives the real chain over HTTP — signin with a proof, 403 + ticket,
// challenge with the ticket — and asserts the session that comes out the far
// end carries the same thumbprint the first factor proved.
func TestSignIn_MFAChallenge_UnderRequiredMode_IssuesBoundSession(t *testing.T) {
	t.Parallel()

	mfaStore := mfa.NewMemoryStore()
	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfaStore)

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithPlugin(mfaPlugin),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	const email = "mfa-dpop@example.com"
	_, signupToken, _ := signUp(t, eng, email, "SecureP@ss1")
	require.NotEmpty(t, signupToken)

	u, err := eng.Store().GetUserByEmail(context.Background(), appID, id.Nil, email)
	require.NoError(t, err)

	totpKey, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{
		Issuer:      "TestApp",
		AccountName: email,
	})
	require.NoError(t, err)
	require.NoError(t, mfaStore.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    u.ID,
		Method:    "totp",
		Secret:    totpKey.Secret(),
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	// Both mandates go on after signup, so signup itself hits neither gate.
	tru := true
	require.NoError(t, eng.Store().SetAppClientConfig(context.Background(), &appclientconfig.Config{
		ID:          id.NewAppClientConfigID(),
		AppID:       appID,
		MFARequired: &tru,
	}))
	setSignInDPoPMode(t, eng, "required")

	rootRouter := forge.NewRouter()
	a := api.New(eng, rootRouter)
	require.NoError(t, a.RegisterRoutes(rootRouter))
	require.NoError(t, mfaPlugin.RegisterRoutes(rootRouter))
	router := withTestKey(rootRouter.Handler())

	// Step 1: sign in carrying a proof. The password leg succeeds and the
	// thumbprint is validated, then the MFA gate fires.
	key := dpoptest.Key(t)
	jkt := dpoptest.Thumbprint(t, key)
	proof := dpoptest.MintProof(t, key, "ES256",
		dpoptest.ValidClaims(http.MethodPost, dpopSignInEndpoint))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signin",
		bytes.NewReader([]byte(`{"email":"`+email+`","password":"SecureP@ss1"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DPoP", proof)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code,
		"a proof-carrying signin must still reach the MFA gate; body=%s", rec.Body.String())
	var gate map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&gate))
	ticket, _ := gate["mfa_ticket"].(string)
	require.NotEmpty(t, ticket)

	// Step 2: complete the second factor.
	code, err := mfa.GenerateTOTPCode(totpKey.Secret())
	require.NoError(t, err)
	chBody, _ := json.Marshal(map[string]string{"mfa_ticket": ticket, "code": code})
	rec = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/mfa/challenge", bytes.NewReader(chBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"challenge must succeed for a ticket minted from a bound signin; body=%s", rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	sessionTok, _ := resp["session_token"].(string)
	require.NotEmpty(t, sessionTok)

	// Step 3: the point of the whole test.
	sess, err := eng.Store().GetSessionByToken(context.Background(), sessionTok)
	require.NoError(t, err)
	assert.Equal(t, jkt, sess.DPoPJKT,
		"the post-MFA session must carry the thumbprint the first factor proved, not an empty one")
}

// TestMFAChallenge_UnboundTicketUnderRequiredMode_Refuses covers the ticket
// that predates the mandate: an operator flips dpop.mode to required while
// partial-auth tickets minted under optional are still outstanding.
//
// Those tickets carry no thumbprint, so completing them would mint exactly the
// unbound session the mandate exists to prevent. The refusal must reach the
// client as the 400 an RFC 9449 client can act on — the MFA plugin used to
// wrap every issuance failure in InternalError, which turned an actionable
// "present a proof" into an opaque 500.
func TestMFAChallenge_UnboundTicketUnderRequiredMode_Refuses(t *testing.T) {
	t.Parallel()

	mfaStore := mfa.NewMemoryStore()
	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfaStore)

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithPlugin(mfaPlugin),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	const email = "mfa-dpop-stale@example.com"
	_, signupToken, _ := signUp(t, eng, email, "SecureP@ss1")
	require.NotEmpty(t, signupToken)

	u, err := eng.Store().GetUserByEmail(context.Background(), appID, id.Nil, email)
	require.NoError(t, err)

	totpKey, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: email})
	require.NoError(t, err)
	require.NoError(t, mfaStore.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    u.ID,
		Method:    "totp",
		Secret:    totpKey.Secret(),
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	tru := true
	require.NoError(t, eng.Store().SetAppClientConfig(context.Background(), &appclientconfig.Config{
		ID:          id.NewAppClientConfigID(),
		AppID:       appID,
		MFARequired: &tru,
	}))

	rootRouter := forge.NewRouter()
	a := api.New(eng, rootRouter)
	require.NoError(t, a.RegisterRoutes(rootRouter))
	require.NoError(t, mfaPlugin.RegisterRoutes(rootRouter))
	router := withTestKey(rootRouter.Handler())

	// Sign in while the app is still on optional, so no proof is demanded
	// and the ticket is minted unbound.
	setSignInDPoPMode(t, eng, "optional")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signin",
		bytes.NewReader([]byte(`{"email":"`+email+`","password":"SecureP@ss1"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	var gate map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&gate))
	ticket, _ := gate["mfa_ticket"].(string)
	require.NotEmpty(t, ticket)

	// The mandate arrives while the ticket is in flight.
	setSignInDPoPMode(t, eng, "required")

	code, err := mfa.GenerateTOTPCode(totpKey.Secret())
	require.NoError(t, err)
	chBody, _ := json.Marshal(map[string]string{"mfa_ticket": ticket, "code": code})
	rec = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/mfa/challenge", bytes.NewReader(chBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"an unbound ticket must be refused, not completed; body=%s", rec.Body.String())
	var refusal map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&refusal))
	assert.Equal(t, "invalid_dpop_proof", refusal["error"],
		"the client needs the RFC 9449 error code, not an opaque server fault")
}
