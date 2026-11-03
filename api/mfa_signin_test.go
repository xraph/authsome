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
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/store/memory"
)

// A user who turned TOTP on is stopped at /v1/signin with a ticket, and
// the challenge that completes it sets the session cookie, the same as a
// password sign-in, so a cookie-session browser is signed in afterwards.
func TestSignIn_EnrolledUser_ChallengeSetsSessionCookie(t *testing.T) {
	t.Parallel()

	mfaStore := mfa.NewMemoryStore()
	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfaStore)

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(authsome.WithConfig(testConfig()), authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithPlugin(mfaPlugin),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)

	rawUser, _, _ := signUp(t, eng, "two-step@example.com", "SecureP@ss123")
	var u struct {
		ID id.UserID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(*rawUser, &u))
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "Test", AccountName: "two-step@example.com"})
	require.NoError(t, err)
	require.NoError(t, mfaStore.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: key.Secret(), Verified: true,
	}))

	root := forge.NewRouter()
	require.NoError(t, api.New(eng, root).RegisterRoutes(root))
	require.NoError(t, mfaPlugin.RegisterRoutes(root))
	h := withTestKey(root.Handler())

	post := func(path string, body any) *httptest.ResponseRecorder {
		raw, merr := json.Marshal(body)
		require.NoError(t, merr)
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	signin := post("/v1/signin", map[string]string{"email": "two-step@example.com", "password": "SecureP@ss123"})
	require.Equal(t, http.StatusForbidden, signin.Code, "body=%s", signin.Body.String())
	assert.Empty(t, signin.Result().Cookies(), "no session cookie before the second factor")
	var refused struct {
		Type   string `json:"type"`
		Ticket string `json:"mfa_ticket"`
	}
	require.NoError(t, json.NewDecoder(signin.Body).Decode(&refused))
	require.Equal(t, "mfa_required", refused.Type)

	code, err := mfa.GenerateTOTPCodeAt(key.Secret(), time.Now())
	require.NoError(t, err)
	challenge := post("/v1/mfa/challenge", map[string]string{"mfa_ticket": refused.Ticket, "code": code})
	require.Equal(t, http.StatusOK, challenge.Code, "body=%s", challenge.Body.String())
	var issued struct {
		SessionToken string `json:"session_token"`
	}
	require.NoError(t, json.NewDecoder(challenge.Body).Decode(&issued))

	var cookie *http.Cookie
	for _, c := range challenge.Result().Cookies() {
		if c.Name == "authsome_session_token" {
			cookie = c
		}
	}
	require.NotNil(t, cookie, "challenge must set the session cookie, got %v", challenge.Result().Cookies())
	assert.Equal(t, issued.SessionToken, cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.Positive(t, cookie.MaxAge)
}
