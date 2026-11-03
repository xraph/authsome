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
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/passkey/passkeytest"
	"github.com/xraph/authsome/session"
)

const testOrigin = "http://localhost:3000"

// A passkey registered on the account signs a browser in with no password,
// and the response sets the session cookie, as a password sign-in does, so
// a browser in cookie session mode is signed in by it.
func TestDiscoverableLogin_SetsSessionCookie(t *testing.T) {
	p := passkey.New(passkey.Config{RPID: "localhost"})
	p.SetStore(passkey.NewMemoryStore())
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "passkey-cookie@example.com", Password: "SecureP@ss123",
	})
	require.NoError(t, err)

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()
	signedIn := func(r *http.Request) *http.Request {
		ctx := middleware.WithUser(r.Context(), u)
		ctx = middleware.WithUserID(ctx, u.ID)
		ctx = middleware.WithSession(ctx, &session.Session{ID: id.NewSessionID(), UserID: u.ID, CreatedAt: time.Now()})
		return r.WithContext(ctx)
	}
	send := func(path string, body []byte, wrap func(*http.Request) *http.Request, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", testOrigin)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		if wrap != nil {
			req = wrap(req)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	options := func(rec *httptest.ResponseRecorder) json.RawMessage {
		var out struct {
			Options json.RawMessage `json:"options"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		return out.Options
	}

	device := passkeytest.NewAuthenticator(testOrigin, "localhost")
	begin := send("/v1/passkeys/register/begin", []byte(`{"display_name":"Laptop"}`), signedIn)
	require.Equal(t, http.StatusOK, begin.Code, "body=%s", begin.Body.String())
	attestation, err := device.Create(options(begin))
	require.NoError(t, err)
	finish := send("/v1/passkeys/register/finish", attestation, signedIn)
	require.Equal(t, http.StatusOK, finish.Code, "body=%s", finish.Body.String())

	loginBegin := send("/v1/passkeys/login/begin", []byte(`{}`), nil)
	require.Equal(t, http.StatusOK, loginBegin.Code, "body=%s", loginBegin.Body.String())
	assertion, err := device.Get(options(loginBegin))
	require.NoError(t, err)
	login := send("/v1/passkeys/login/finish", assertion, nil, loginBegin.Result().Cookies()...)
	require.Equal(t, http.StatusOK, login.Code, "body=%s", login.Body.String())

	var issued passkey.LoginFinishResponse
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &issued))
	require.NotEmpty(t, issued.SessionToken)
	var cookie *http.Cookie
	for _, c := range login.Result().Cookies() {
		if c.Name == "authsome_session_token" {
			cookie = c
		}
	}
	require.NotNil(t, cookie, "login must set the session cookie, got %v", login.Result().Cookies())
	assert.Equal(t, issued.SessionToken, cookie.Value)
	assert.True(t, cookie.HttpOnly)
}
