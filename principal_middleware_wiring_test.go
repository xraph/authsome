package authsome_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// TestBuildAuthMiddlewareWiresPrincipalResolver proves that
// Engine.buildAuthMiddleware actually connects Engine.ResolvePrincipal to the
// middleware, not just that the middleware plumbing (tested in
// middleware/principal_test.go) is correct in isolation.
//
// The check specifically reads Scopes, a field ToPrincipal only fills in
// from the service account row in the store. PrincipalRefFrom's fallback
// path (session subject reconstructed without a store round-trip) can only
// ever produce a bare Ref{Kind, ID}, so a non-empty Scopes slice here is
// proof the value came from a real store lookup through
// Engine.ResolvePrincipal, not from the fallback.
func TestBuildAuthMiddlewareWiresPrincipalResolver(t *testing.T) {
	eng, store := newTestEngine(t)
	ctx := context.Background()

	appID, err := id.ParseAppID(eng.DefaultAppID())
	require.NoError(t, err)

	svc, err := eng.CreateServiceAccount(ctx, appID, "ci-runner", "", []string{"ci:build", "ci:deploy"})
	require.NoError(t, err)

	sess := &session.Session{
		ID:               id.NewSessionID(),
		AppID:            appID,
		PrincipalKind:    principal.KindService,
		ServiceAccountID: svc.ID,
		Token:            "wiring-proof-token",
		ExpiresAt:        time.Now().Add(time.Hour),
	}
	require.NoError(t, store.CreateSession(ctx, sess))

	var (
		got   *principal.Principal
		gotOK bool
	)
	router := forge.NewRouter()
	router.Use(eng.AuthMiddleware())
	router.GET("/probe", func(fctx forge.Context) error {
		got, gotOK = principal.FromContext(fctx.Context())
		return fctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/probe", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer wiring-proof-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, gotOK, "the engine's own auth middleware must put a resolved "+
		"*principal.Principal on the context, not leave principal.FromContext empty")
	assert.Equal(t, principal.KindService, got.Kind)
	assert.Equal(t, svc.ID.String(), got.ID)
	assert.ElementsMatch(t, []string{"ci:build", "ci:deploy"}, got.Scopes,
		"Scopes only exists on the store row: seeing it here proves ResolvePrincipal ran")
}
