package passkey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/internal/secutil"
)

// TestDiscoverableLoginFlow_RoutesWithoutPreAuth verifies the two behavioral
// fixes for passwordless passkey login end-to-end (minus the assertion crypto,
// which go-webauthn owns):
//   - /login/begin with no email sets a per-ceremony correlation cookie and
//     stores the ceremony under a unique key (no shared global slot).
//   - /login/finish carrying that cookie takes the discoverable path and does
//     NOT require a pre-authenticated session (the old behavior rejected it
//     with "authentication required", defeating passwordless login entirely).
func TestDiscoverableLoginFlow_RoutesWithoutPreAuth(t *testing.T) {
	p := New()
	_ = secutil.NewTestEngine(t, authsome.WithPlugin(p))

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	handler := router.Handler()

	// Begin (no email → discoverable).
	beginReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/passkeys/login/begin", strings.NewReader(`{}`))
	beginReq.Header.Set("Content-Type", "application/json")
	beginRec := httptest.NewRecorder()
	handler.ServeHTTP(beginRec, beginReq)
	require.Equal(t, http.StatusOK, beginRec.Code, "discoverable begin should succeed; body=%s", beginRec.Body.String())

	var ceremonyCookie *http.Cookie
	for _, c := range beginRec.Result().Cookies() {
		if c.Name == ceremonyCookieName {
			ceremonyCookie = c
		}
	}
	require.NotNil(t, ceremonyCookie, "begin must set a ceremony correlation cookie")
	require.NotEmpty(t, ceremonyCookie.Value)

	// The ceremony must be stored under a per-ceremony key (not a global one).
	_, err := p.ceremonies.Get(context.Background(), discoverableKey(ceremonyCookie.Value))
	require.NoError(t, err, "the ceremony session must be stored under its unique key")

	// Finish carrying the ceremony cookie takes the passwordless path. The
	// assertion body is invalid, so it fails — but with an assertion error, NOT
	// "authentication required" (which would mean pre-auth was still demanded).
	finishReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/passkeys/login/finish", strings.NewReader(`{}`))
	finishReq.Header.Set("Content-Type", "application/json")
	finishReq.AddCookie(ceremonyCookie)
	finishRec := httptest.NewRecorder()
	handler.ServeHTTP(finishRec, finishReq)

	body := finishRec.Body.String()
	assert.NotContains(t, body, "authentication required",
		"the discoverable path must not require a pre-authenticated session; body=%s", body)
}

// TestLoginFinish_IdentifiedPath_StillRequiresAuth confirms the identified
// (step-up) path is unchanged: without a ceremony cookie and without a session,
// finish still rejects with authentication required.
func TestLoginFinish_IdentifiedPath_StillRequiresAuth(t *testing.T) {
	p := New()
	_ = secutil.NewTestEngine(t, authsome.WithPlugin(p))

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	handler := router.Handler()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/passkeys/login/finish", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "authentication required")
}
