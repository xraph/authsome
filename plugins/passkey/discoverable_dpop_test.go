package passkey

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/settings"
)

// issueSessionProbeRequest is the empty body of the test-only route below.
type issueSessionProbeRequest struct{}

// issueSessionProbeResponse carries back what the test needs to assert on.
type issueSessionProbeResponse struct {
	DPoPJKT string `json:"dpop_jkt"`
}

// TestPasskeyIssueSession_UnderRequiredMode_BindsSession covers the passkey
// path's binding.
//
// A full /login/finish round-trip is not available to test: completing it
// needs a real WebAuthn assertion, which go-webauthn owns and no test in this
// package fakes. So this drives Plugin.issueSession — the single place passkey
// mints a session, and the whole of what this change touches — over a real
// forge request carrying a proof, through a route registered only here.
func TestPasskeyIssueSession_UnderRequiredMode_BindsSession(t *testing.T) {
	p := New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	u, _, signUpErr := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:    appID,
		Email:    "passkey-dpop@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, signUpErr)

	// Raise the mandate after signup so signup itself isn't refused.
	mgr := eng.Settings()
	require.NotNil(t, mgr)
	raw, err := json.Marshal("required")
	require.NoError(t, err)
	require.NoError(t, mgr.Set(context.Background(), "dpop.mode", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	router := forge.NewRouter()
	require.NoError(t, router.POST("/test/passkey-issue",
		func(ctx forge.Context, _ *issueSessionProbeRequest) (*issueSessionProbeResponse, error) {
			sess, issueErr := p.issueSession(ctx, u)
			if issueErr != nil {
				return nil, issueErr
			}
			return &issueSessionProbeResponse{DPoPJKT: sess.DPoPJKT}, nil
		}))
	handler := router.Handler()

	const endpoint = "http://example.com/test/passkey-issue"
	key := dpoptest.Key(t)
	jkt := dpoptest.Thumbprint(t, key)
	proof := dpoptest.MintProof(t, key, "ES256",
		dpoptest.ValidClaims(http.MethodPost, endpoint))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/test/passkey-issue", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"a proof-carrying passkey login must mint a session; body=%s", rec.Body.String())
	var resp issueSessionProbeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, jkt, resp.DPoPJKT, "the passkey session must be bound to the proven key")

}
