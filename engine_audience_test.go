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

// audienceRequest builds a bearer-authenticated request whose context
// already carries the app ID, the way PublishableKeyMiddleware would
// populate it ahead of AuthMiddleware in the production router chain.
// resolveExpectedAudience reads the app ID off exactly this context value.
func audienceRequest(appID id.AppID, token string) *http.Request {
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
	audienceRouter(eng).ServeHTTP(rec, audienceRequest(appID, sess.Token))

	assert.Equal(t, http.StatusOK, rec.Code,
		"setting unset must disable the audience check: a session audienced elsewhere still authenticates")
}

// TestEngineExpectedAudience_SetEnforcesMatch: once session.resource_identifier
// is set for the app, a token audienced at a different resource is refused,
// and a token audienced at the configured resource still authenticates. Both
// halves matter: proving only the rejection would be satisfied by a check
// that rejects everything.
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
	router.ServeHTTP(recWrong, audienceRequest(appID, wrongSess.Token))
	assert.Equal(t, http.StatusUnauthorized, recWrong.Code,
		"a token audienced at a different resource must not authenticate once the setting is set")

	recRight := httptest.NewRecorder()
	router.ServeHTTP(recRight, audienceRequest(appID, rightSess.Token))
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
	audienceRouter(eng).ServeHTTP(rec, audienceRequest(appID, sess.Token))

	assert.Equal(t, http.StatusOK, rec.Code,
		"a session with no audience must still authenticate even when the setting is set")
}
