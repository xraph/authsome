package browserbind

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAndMatch_HTTPSUsesHostPrefixedCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "https://auth.example/v1/sso/login", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	SetStateCookie(rec, r, "state-1")
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	assert.Equal(t, CookieName, c.Name)
	assert.True(t, c.Secure)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, "/", c.Path)
	assert.Equal(t, http.SameSiteNoneMode, c.SameSite)
	assert.Equal(t, Digest("state-1"), c.Value)
	assert.NotEqual(t, "state-1", c.Value, "the cookie holds a digest, not the state")

	cb := httptest.NewRequest(http.MethodGet, "https://auth.example/cb?state=state-1", nil)
	cb.AddCookie(c)
	assert.True(t, Matches(cb, "state-1"))
	assert.False(t, Matches(cb, "state-2"), "a cookie for one login does not bind another")
	bare := httptest.NewRequest(http.MethodGet, "https://auth.example/cb", nil)
	assert.False(t, Matches(bare, "state-1"), "no cookie, no binding")
}

func TestSetAndMatch_PlainHTTPFallsBackToUnprefixedLax(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "http://localhost:8080/v1/sso/login", nil)
	rec := httptest.NewRecorder()
	SetStateCookie(rec, r, "s")
	c := rec.Result().Cookies()[0]
	assert.Equal(t, InsecureCookieName, c.Name)
	assert.False(t, c.Secure)
	assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
	cb := httptest.NewRequest(http.MethodGet, "http://localhost:8080/cb", nil)
	cb.AddCookie(c)
	assert.True(t, Matches(cb, "s"))
}

func TestIsBrowser(t *testing.T) {
	native := httptest.NewRequest(http.MethodPost, "/", nil)
	assert.False(t, IsBrowser(native))
	fetch := httptest.NewRequest(http.MethodPost, "/", nil)
	fetch.Header.Set("Sec-Fetch-Site", "same-origin")
	assert.True(t, IsBrowser(fetch))
	cors := httptest.NewRequest(http.MethodPost, "/", nil)
	cors.Header.Set("Origin", "https://app.example")
	assert.True(t, IsBrowser(cors))
	assert.False(t, IsBrowser(nil))
}
