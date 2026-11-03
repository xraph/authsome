package passkey_test

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
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/passkey/passkeytest"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// twoStepFixture is an engine with both plugins and a user who has TOTP on.
type twoStepFixture struct {
	h      http.Handler
	u      *user.User
	secret string
}

func newTwoStepFixture(t *testing.T) twoStepFixture {
	t.Helper()
	pk := passkey.New(passkey.Config{RPID: "localhost"})
	pk.SetStore(passkey.NewMemoryStore())
	mfaStore := mfa.NewMemoryStore()
	m := mfa.New()
	m.SetStore(mfaStore)
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(m), authsome.WithPlugin(pk))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "two-step-passkey@example.com", Password: "SecureP@ss123",
	})
	require.NoError(t, err)
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "Test", AccountName: u.Email})
	require.NoError(t, err)
	require.NoError(t, mfaStore.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: key.Secret(), Verified: true,
	}))

	router := forge.NewRouter()
	require.NoError(t, pk.RegisterRoutes(router))
	return twoStepFixture{h: router.Handler(), u: u, secret: key.Secret()}
}

func (f twoStepFixture) send(t *testing.T, path string, body []byte, signedIn bool, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", testOrigin)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if signedIn {
		ctx := middleware.WithUser(req.Context(), f.u)
		ctx = middleware.WithUserID(ctx, f.u.ID)
		ctx = middleware.WithSession(ctx, &session.Session{ID: id.NewSessionID(), UserID: f.u.ID, CreatedAt: time.Now()})
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func optionsOf(t *testing.T, rec *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var out struct {
		Options json.RawMessage `json:"options"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out.Options
}

// register adds the device's passkey with a current TOTP code.
func (f twoStepFixture) register(t *testing.T, device *passkeytest.Authenticator) {
	t.Helper()
	code, err := mfa.GenerateTOTPCodeAt(f.secret, time.Now())
	require.NoError(t, err)
	body, err := json.Marshal(map[string]string{"display_name": "Key", "code": code})
	require.NoError(t, err)
	begin := f.send(t, "/v1/passkeys/register/begin", body, true)
	require.Equal(t, http.StatusOK, begin.Code, "body=%s", begin.Body.String())
	attestation, err := device.Create(optionsOf(t, begin))
	require.NoError(t, err)
	finish := f.send(t, "/v1/passkeys/register/finish", attestation, true)
	require.Equal(t, http.StatusOK, finish.Code, "body=%s", finish.Body.String())
}

func (f twoStepFixture) login(t *testing.T, device *passkeytest.Authenticator) *httptest.ResponseRecorder {
	t.Helper()
	begin := f.send(t, "/v1/passkeys/login/begin", []byte(`{}`), false)
	require.Equal(t, http.StatusOK, begin.Code, "body=%s", begin.Body.String())
	assertion, err := device.Get(optionsOf(t, begin))
	require.NoError(t, err)
	return f.send(t, "/v1/passkeys/login/finish", assertion, false, begin.Result().Cookies()...)
}

// With two-step on, a new passkey is a new way in that skips the code, so
// adding one takes a code, the same as changing the factor itself.
func TestRegisterBegin_TwoStepOn_NeedsACode(t *testing.T) {
	f := newTwoStepFixture(t)
	rec := f.send(t, "/v1/passkeys/register/begin", []byte(`{"display_name":"Key"}`), true)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
	rec = f.send(t, "/v1/passkeys/register/begin", []byte(`{"display_name":"Key","code":"000000"}`), true)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())

	f.register(t, passkeytest.NewAuthenticator(testOrigin, "localhost"))
}

// A passkey that verified its user signs in with no code; one that only
// proved presence is asked for the code like a password.
func TestDiscoverableLogin_TwoStepOn_CodeOnlyWithoutUserVerification(t *testing.T) {
	f := newTwoStepFixture(t)
	device := passkeytest.NewAuthenticator(testOrigin, "localhost")
	f.register(t, device)

	verified := f.login(t, device)
	require.Equal(t, http.StatusOK, verified.Code, "body=%s", verified.Body.String())

	device.NoUserVerification = true
	present := f.login(t, device)
	require.Equal(t, http.StatusForbidden, present.Code, "body=%s", present.Body.String())
	var refused struct {
		Type   string `json:"type"`
		Ticket string `json:"mfa_ticket"`
	}
	require.NoError(t, json.Unmarshal(present.Body.Bytes(), &refused))
	assert.Equal(t, "mfa_required", refused.Type)
	assert.NotEmpty(t, refused.Ticket)
	for _, c := range present.Result().Cookies() {
		assert.NotEqual(t, "authsome_session_token", c.Name, "no session cookie before the code")
	}
}
