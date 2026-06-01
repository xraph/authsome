package authsome_test

// Regression test for the cross-publishable-key session leak: a session minted
// under app A must NOT be honored when the caller presents app B's publishable
// key. See middleware.requestAppIDMismatch and the guards in
// middleware/auth.go.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

const (
	xpkPlatformAppIDStr = "aapp_01jf0000000000000000000000"
	xpkTenantAppIDStr   = "aapp_01jf0000000000000000000099"
	xpkPlatformKey      = "pk_test_authsome_xpk_platform"
	xpkTenantKey        = "pk_test_authsome_xpk_tenant"
)

// crossPKRouter wires the publishable-key middleware ahead of the session auth
// middleware (the production ordering) and a protected /test endpoint that
// 401s when no authenticated user landed on the context.
func crossPKRouter(eng *authsome.Engine) forge.Router {
	router := forge.NewRouter()
	router.Use(middleware.PublishableKeyMiddleware(eng, eng.Logger()))
	router.Use(middleware.AuthMiddlewareWithStrategies(
		eng.ResolveSessionByToken,
		eng.ResolveUser,
		eng.Strategies(),
		eng.Logger(),
	))
	router.GET("/test", func(ctx forge.Context) error {
		if _, ok := middleware.UserFrom(ctx.Context()); !ok {
			return ctx.NoContent(http.StatusUnauthorized)
		}
		return ctx.NoContent(http.StatusOK)
	})
	return router
}

// TestCrossPublishableKeySessionRejected pins that a session issued under the
// tenant app is rejected when the caller switches to the platform app's
// publishable key, while same-key and no-key requests still succeed.
func TestCrossPublishableKeySessionRejected(t *testing.T) {
	s := memory.New()
	now := time.Now()

	platformID, err := id.ParseAppID(xpkPlatformAppIDStr)
	require.NoError(t, err)
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID:             platformID,
		Name:           "Platform",
		Slug:           "platform",
		PublishableKey: xpkPlatformKey,
		IsPlatform:     true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}))

	tenantID, err := id.ParseAppID(xpkTenantAppIDStr)
	require.NoError(t, err)
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID:             tenantID,
		Name:           "Tenant",
		Slug:           "tenant",
		PublishableKey: xpkTenantKey,
		IsPlatform:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}))

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(xpkPlatformAppIDStr),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })
	secutil.RelaxAuthDefaults(t, eng)

	// Sign up + sign in a user under the TENANT app → session bound to tenant.
	_, _, err = eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    tenantID,
		Email:    "tenant-user@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	_, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    tenantID,
		Email:    "tenant-user@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, sess.Token)
	require.Equal(t, tenantID.String(), sess.AppID.String(), "session must be bound to the tenant app")

	router := crossPKRouter(eng)

	call := func(pk string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+sess.Token)
		if pk != "" {
			req.Header.Set(middleware.PublishableKeyHeader, pk)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	// Same app's pk → accepted.
	assert.Equal(t, http.StatusOK, call(xpkTenantKey),
		"session must be accepted under its own app's publishable key")

	// Different app's pk → rejected (the bug).
	assert.Equal(t, http.StatusUnauthorized, call(xpkPlatformKey),
		"tenant session must NOT be honored under the platform app's publishable key")

	// No pk → unchanged (bearer-only flows unaffected by the lenient policy).
	assert.Equal(t, http.StatusOK, call(""),
		"bearer-only request without a publishable key must still authenticate")
}
