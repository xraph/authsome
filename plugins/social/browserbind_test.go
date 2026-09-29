package social_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/internal/browserbind"
)

// A login a browser started must come back from that browser.
func TestHandleCallback_BrowserStartRequiresTheStateCookie(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	start := func(browser bool) (string, []*http.Cookie) {
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/social/google", nil)
		if browser {
			req.Header.Set("Sec-Fetch-Site", "same-origin")
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		return extractQueryParam(t, resp["auth_url"], "state"), rec.Result().Cookies()
	}
	callback := func(state string, cookies []*http.Cookie) (int, string) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/social/google/callback?code=abc&state="+state, nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		msg, _ := body["error"].(string)
		return rec.Code, msg
	}

	state, cookies := start(true)
	require.Len(t, cookies, 1, "a browser start sets the binding cookie")
	assert.Equal(t, browserbind.InsecureCookieName, cookies[0].Name, "plain http gets the unprefixed cookie")
	code, msg := callback(state, nil)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, msg, "different browser")

	state, cookies = start(true)
	_, msg = callback(state, cookies)
	assert.NotContains(t, msg, "different browser", "the bound browser gets past the check")

	state, cookies = start(false)
	assert.Empty(t, cookies, "a native start sets no cookie")
	_, msg = callback(state, nil)
	assert.NotContains(t, msg, "different browser", "a native login needs no cookie")
}
