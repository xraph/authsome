package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/user"
)

func TestPrincipalContextCarriers(t *testing.T) {
	ctx := context.Background()

	_, ok := middleware.PrincipalFrom(ctx)
	assert.False(t, ok)

	p := &principal.Principal{Ref: principal.Ref{Kind: principal.KindWorkload, ID: "svc_ci"}}
	ctx = middleware.WithPrincipal(ctx, p)

	got, ok := middleware.PrincipalFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, p, got)

	// The same value must be readable through the principal package, so a
	// plugin can get the caller without importing middleware.
	viaPackage, ok := principal.FromContext(ctx)
	require.True(t, ok, "middleware and principal must share one context key")
	assert.Equal(t, p, viaPackage)
}

func TestActorsContextCarrier(t *testing.T) {
	ctx := context.Background()
	chain := principal.Chain{{Kind: principal.KindAgent, ID: "svc_a"}}
	ctx = middleware.WithActors(ctx, chain)

	got, ok := middleware.ActorsFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, chain, got)

	viaPackage, ok := principal.ActorsFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, chain, viaPackage)
}

func TestImpersonatorStillLandsOnContext(t *testing.T) {
	admin := id.NewUserID()
	ctx := middleware.WithImpersonator(context.Background(), admin)

	got, ok := middleware.ImpersonatorFrom(ctx)
	require.True(t, ok, "the impersonator carrier must keep working")
	assert.Equal(t, admin.String(), got.String())
}

// ──────────────────────────────────────────────────
// End-to-end wiring: both auth paths populate the principal
// ──────────────────────────────────────────────────

// TestSessionPathResolvesPrincipal exercises the bearer-session path
// (AuthMiddleware) with a non-human session and a PrincipalResolver wired
// through SessionBindingConfig. It proves the resolved value lands on the
// context under the shared key: readable both through
// middleware.PrincipalFrom and principal.FromContext.
func TestSessionPathResolvesPrincipal(t *testing.T) {
	svcID := id.NewServiceAccountID()
	testSession := &session.Session{
		ID:               id.NewSessionID(),
		AppID:            id.NewAppID(),
		PrincipalKind:    principal.KindWorkload,
		ServiceAccountID: svcID,
		Token:            "svc-token",
	}
	wantPrincipal := &principal.Principal{
		Ref: principal.Ref{Kind: principal.KindWorkload, ID: svcID.String()},
	}

	mw := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == "svc-token" {
				return testSession, nil
			}
			return nil, errors.New("invalid")
		},
		func(_ string) (*user.User, error) {
			// AuthMiddleware's bearer-session path resolves the user
			// unconditionally, same as before this change (untouched
			// behavior). A workload session carries no UserID, so this
			// always misses: the assertions below only care that the
			// principal still lands on the context regardless.
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			PrincipalResolver: func(ref principal.Ref) (*principal.Principal, error) {
				assert.Equal(t, wantPrincipal.Ref, ref)
				return wantPrincipal, nil
			},
		},
	)

	var (
		gotMiddleware *principal.Principal
		gotPackage    *principal.Principal
		okMiddleware  bool
		okPackage     bool
	)
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotMiddleware, okMiddleware = middleware.PrincipalFrom(ctx.Context())
		gotPackage, okPackage = principal.FromContext(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer svc-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, okMiddleware)
	require.True(t, okPackage, "principal.FromContext must see what AuthMiddleware wrote")
	assert.Equal(t, wantPrincipal, gotMiddleware)
	assert.Equal(t, wantPrincipal, gotPackage)
}

// TestStrategyPathResolvesPrincipal exercises the strategy fallback path
// (AuthMiddlewareWithStrategies) with a strategy result carrying a
// non-human session, proving tryStrategyAuth resolves the principal the
// same way the session path does.
func TestStrategyPathResolvesPrincipal(t *testing.T) {
	svcID := id.NewServiceAccountID()
	strategySession := &session.Session{
		ID:               id.NewSessionID(),
		AppID:            id.NewAppID(),
		PrincipalKind:    principal.KindService,
		ServiceAccountID: svcID,
	}
	wantPrincipal := &principal.Principal{
		Ref: principal.Ref{Kind: principal.KindService, ID: svcID.String()},
	}

	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("no bearer session")
		},
		func(_ string) (*user.User, error) {
			t.Fatal("resolveUser should not be called for a non-human strategy result")
			return nil, nil
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return &strategy.Result{Session: strategySession}, nil
			},
		},
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			PrincipalResolver: func(ref principal.Ref) (*principal.Principal, error) {
				assert.Equal(t, wantPrincipal.Ref, ref)
				return wantPrincipal, nil
			},
		},
	)

	var got *principal.Principal
	var ok bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		got, ok = principal.FromContext(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, ok)
	assert.Equal(t, wantPrincipal, got)
}

// TestPrincipalResolutionFailureIsEnrichmentNotRejection proves the design
// point: a principal-store failure is logged and passed over, not turned
// into a request failure. The session already authenticated the caller.
func TestPrincipalResolutionFailureIsEnrichmentNotRejection(t *testing.T) {
	testSession := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: id.NewUserID(),
		Token:  "human-token",
	}
	testUser := &user.User{ID: testSession.UserID}

	mw := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == "human-token" {
				return testSession, nil
			}
			return nil, errors.New("invalid")
		},
		func(_ string) (*user.User, error) {
			return testUser, nil
		},
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			PrincipalResolver: func(principal.Ref) (*principal.Principal, error) {
				return nil, errors.New("principal store unavailable")
			},
		},
	)

	var gotUser *user.User
	var userOK bool
	var principalOK bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		_, principalOK = middleware.PrincipalFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer human-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "a principal-resolution failure must not fail the request")
	require.True(t, userOK, "the user, resolved separately, must still be set")
	assert.Equal(t, testUser, gotUser)
	assert.False(t, principalOK, "no principal should be on the context when resolution failed")
}
