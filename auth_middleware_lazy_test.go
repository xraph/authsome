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
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store/memory"
)

// AuthMiddleware returned nil until the engine was initialized, so a caller
// wiring its router before Start mounted a nil middleware. It now hands back
// one that resolves the real middleware on each request.
func TestAuthMiddleware_BeforeStartResolvesLazily(t *testing.T) {
	s := memory.New()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID: appID, Name: "Platform", Slug: "platform", PublishableKey: "pk_test_lazy_mw",
		IsPlatform: true, CreatedAt: now, UpdatedAt: now,
	}))
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(testEngineConfig()),
		authsome.WithAppID(appID.String()),
	)
	require.NoError(t, err)

	mw := eng.AuthMiddleware()
	require.NotNil(t, mw, "AuthMiddleware must not return nil before Start")

	var signedIn bool
	router := forge.NewRouter()
	router.Use(mw)
	require.NoError(t, router.GET("/probe", func(fctx forge.Context) error {
		_, signedIn = middleware.UserIDFrom(fctx.Context())
		return fctx.NoContent(http.StatusOK)
	}))
	probe := func(token string) int {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/probe", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	// Before the engine is up, requests pass through unauthenticated.
	assert.Equal(t, http.StatusOK, probe("not-yet"))
	assert.False(t, signedIn)

	require.NoError(t, eng.Start(context.Background()))
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })
	secutil.RelaxAuthDefaults(t, eng)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "lazy@example.com", Password: "SecureP@ss123", FirstName: "Lazy",
	})
	require.NoError(t, err)
	require.NotNil(t, sess)

	// The same middleware value, mounted before Start, now authenticates.
	assert.Equal(t, http.StatusOK, probe(sess.Token))
	assert.True(t, signedIn, "the lazily resolved middleware must authenticate once the engine has started")
}
