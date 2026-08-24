package authsome_test

// Tests for the RFC 8707 resource-indicator audience check as wired through
// the real engine: session.resource_identifier resolved per app via
// Engine.resolveExpectedAudience, and plugged into the production
// middleware config in Engine.buildAuthMiddleware.
//
// middleware/auth_audience_test.go already proves the audience-check logic
// itself is correct given an arbitrary ExpectedAudienceResolver. What is
// untested there, and is the whole point of this file, is that the engine
// actually supplies one: that turning the app setting on and off changes
// what a real request through eng.AuthMiddleware() does.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
)

// audienceRouter wires the engine's fully-built auth middleware (the
// production path, including resolveExpectedAudience) ahead of a protected
// endpoint that 401s when no authenticated user landed on the context. A
// subtest here genuinely distinguishes "authenticated" from "not
// authenticated": it never asserts a status code from a handler that
// doesn't require auth.
func audienceRouter(eng *authsome.Engine) forge.Router {
	router := forge.NewRouter()
	router.Use(eng.AuthMiddleware())
	router.GET("/test", func(ctx forge.Context) error {
		if _, ok := middleware.UserFrom(ctx.Context()); !ok {
			return ctx.NoContent(http.StatusUnauthorized)
		}
		return ctx.NoContent(http.StatusOK)
	})
	return router
}

// audienceRequest builds a bearer-authenticated request whose context carries
// no app ID at all.
//
// This is the ordinary case, not an edge case. PublishableKeyMiddleware is
// what puts an app ID on the context, and it is a no-op when the caller omits
// X-Publishable-Key; a resource server built on eng.AuthMiddleware() need
// never install it. So the audience check has to work from what the TOKEN
// says, and every test below that does not say otherwise runs in this
// configuration on purpose.
func audienceRequest(token string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// audienceRequestWithAppID is audienceRequest plus the context value
// PublishableKeyMiddleware would set, so both halves of the fallback question
// stay covered: adding a publishable key must not change the outcome.
func audienceRequestWithAppID(appID id.AppID, token string) *http.Request {
	ctx := middleware.WithAppID(context.Background(), appID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// setResourceIdentifier sets session.resource_identifier at app scope,
// mirroring how an operator would configure it for their app.
func setResourceIdentifier(t *testing.T, eng *authsome.Engine, appID id.AppID, value string) {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(context.Background(), "session.resource_identifier", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
}

// TestEngineExpectedAudience_UnsetDisablesCheck: with the setting left at
// its empty default, a session audienced at some other resource must still
// authenticate. Unset means no check, which is the backwards-compatibility
// default for every app that never touches this setting.
func TestEngineExpectedAudience_UnsetDisablesCheck(t *testing.T) {
	eng, memStore := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "audience-unset@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	sess.Audience = []string{"https://other.example.com"}
	require.NoError(t, memStore.UpdateSession(ctx, sess))

	rec := httptest.NewRecorder()
	audienceRouter(eng).ServeHTTP(rec, audienceRequest(sess.Token))

	assert.Equal(t, http.StatusOK, rec.Code,
		"setting unset must disable the audience check: a session audienced elsewhere still authenticates")
}

// TestEngineExpectedAudience_SetEnforcesMatch: once session.resource_identifier
// is set for the app, a token audienced at a different resource is refused,
// and a token audienced at the configured resource still authenticates. Both
// halves matter: proving only the rejection would be satisfied by a check
// that rejects everything.
//
// The requests here carry no app ID on their context. That is the whole point.
// The setting lives at App scope, so resolving it needs an app ID from
// somewhere, and the only one always available is the one stamped on the token
// itself. Resolve it from the request instead and this test's rejection case
// authenticates, which is how the check managed to look wired while doing
// nothing.
func TestEngineExpectedAudience_SetEnforcesMatch(t *testing.T) {
	eng, memStore := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	setResourceIdentifier(t, eng, appID, "https://api.example.com")

	_, wrongSess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "audience-wrong@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	wrongSess.Audience = []string{"https://other.example.com"}
	require.NoError(t, memStore.UpdateSession(ctx, wrongSess))

	_, rightSess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "audience-right@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	rightSess.Audience = []string{"https://api.example.com"}
	require.NoError(t, memStore.UpdateSession(ctx, rightSess))

	router := audienceRouter(eng)

	recWrong := httptest.NewRecorder()
	router.ServeHTTP(recWrong, audienceRequest(wrongSess.Token))
	assert.Equal(t, http.StatusUnauthorized, recWrong.Code,
		"a token audienced at a different resource must not authenticate once the setting is set")

	recRight := httptest.NewRecorder()
	router.ServeHTTP(recRight, audienceRequest(rightSess.Token))
	assert.Equal(t, http.StatusOK, recRight.Code,
		"a token audienced at the configured resource must still authenticate")
}

// TestEngineExpectedAudience_SetButSessionHasNoAudience: with the setting
// set, a session that carries no audience at all still authenticates. This
// is the regression guard for every session issued before RFC 8707 support
// existed; getting this wrong would break every deployment the moment an
// operator sets the setting.
func TestEngineExpectedAudience_SetButSessionHasNoAudience(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	setResourceIdentifier(t, eng, appID, "https://api.example.com")

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "audience-none@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	require.Empty(t, sess.Audience, "a session minted without a resource parameter carries no audience")

	rec := httptest.NewRecorder()
	audienceRouter(eng).ServeHTTP(rec, audienceRequest(sess.Token))

	assert.Equal(t, http.StatusOK, rec.Code,
		"a session with no audience must still authenticate even when the setting is set")
}

// TestEngineExpectedAudience_SetEnforcesMatchWithPublishableKey repeats the
// enforcement case with an app ID on the request context, the way
// PublishableKeyMiddleware leaves it.
//
// Sending a publishable key must not change the answer either way. Keeping
// both configurations covered is what stops a future change from quietly
// going back to reading the request: that change would still pass here and
// fail the no-app-ID test above.
func TestEngineExpectedAudience_SetEnforcesMatchWithPublishableKey(t *testing.T) {
	eng, memStore := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	setResourceIdentifier(t, eng, appID, "https://api.example.com")

	_, wrongSess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "audience-pk-wrong@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	wrongSess.Audience = []string{"https://other.example.com"}
	require.NoError(t, memStore.UpdateSession(ctx, wrongSess))

	_, rightSess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "audience-pk-right@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	rightSess.Audience = []string{"https://api.example.com"}
	require.NoError(t, memStore.UpdateSession(ctx, rightSess))

	router := audienceRouter(eng)

	recWrong := httptest.NewRecorder()
	router.ServeHTTP(recWrong, audienceRequestWithAppID(appID, wrongSess.Token))
	assert.Equal(t, http.StatusUnauthorized, recWrong.Code,
		"a token audienced at a different resource must not authenticate with a publishable key either")

	recRight := httptest.NewRecorder()
	router.ServeHTTP(recRight, audienceRequestWithAppID(appID, rightSess.Token))
	assert.Equal(t, http.StatusOK, recRight.Code,
		"a token audienced at the configured resource must still authenticate with a publishable key")
}
