package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/browserbind"
)

// failingProvider gets past state and cookie checks and then refuses the
// callback, so a landing that reaches it reports auth_failed rather than a
// state error.
type failingProvider struct{ stubProvider }

func (failingProvider) HandleCallback(context.Context, map[string]string) (*User, error) {
	return nil, errors.New("stub refuses")
}

func TestOIDCRedirect_BrowserStartRequiresTheStateCookie(t *testing.T) {
	// The browser landing serves database connections, so the login is
	// started against one whose issuer never resolves: a landing that gets
	// past the state checks fails at the provider, which is the signal.
	p := New(Config{PublicBaseURL: "https://api.example.com"})
	p.ceremonies = ceremony.NewMemory()
	p.SetSSOStore(NewMemoryStore())
	appID := id.NewAppID()
	conn := &Connection{ID: id.NewSSOConnectionID(), AppID: appID, Provider: "stub", Protocol: "oidc", Domain: "corp.example", Issuer: "https://idp.invalid", ClientID: "c", Active: true}
	require.NoError(t, p.ssoStore.CreateConnection(context.Background(), conn))

	browserStart := func() (*LoginResponse, []*http.Cookie) {
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/sso/login", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		resp, err := p.startLogin(context.Background(), appID, failingProvider{}, "stub", conn.ID.String(), "", rec, req)
		require.NoError(t, err)
		return resp, rec.Result().Cookies()
	}

	router := forge.NewRouter()
	require.NoError(t, router.GET("/redirect/:provider", p.handleOIDCRedirect))
	land := func(state string, cookies []*http.Cookie) string {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/redirect/stub?code=c&state="+state, nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusFound, rec.Code)
		return rec.Header().Get("Location")
	}

	resp, cookies := browserStart()
	require.Len(t, cookies, 1, "a browser start sets the binding cookie")
	assert.Equal(t, browserbind.CookieName, cookies[0].Name)
	assert.Contains(t, land(resp.State, nil), "sso_error=state_cookie_missing", "a landing without the cookie is refused before the provider is asked")

	resp, cookies = browserStart()
	loc := land(resp.State, cookies)
	assert.False(t, strings.Contains(loc, "state_cookie_missing"), "with the cookie the landing reaches the provider: %s", loc)
	assert.Contains(t, loc, "sso_error=auth_failed")

	// A start from a native client sets no cookie and its landing needs none.
	nativeReq := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/sso/login", nil)
	nativeRec := httptest.NewRecorder()
	resp, err := p.startLogin(context.Background(), appID, failingProvider{}, "stub", conn.ID.String(), "", nativeRec, nativeReq)
	require.NoError(t, err)
	assert.Empty(t, nativeRec.Result().Cookies())
	assert.Contains(t, land(resp.State, nil), "sso_error=auth_failed")
}
