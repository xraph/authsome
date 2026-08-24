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
	"github.com/xraph/authsome/user"
)

// TestEnforcement_MachineCredentialReachesGuardedRoute is the deliverable of
// the non-human principal enforcement work.
//
// Before it, a machine credential authenticated, carried stamped roles,
// resolved cleanly from its token, and then got a 401 from every guard it
// met, because all three enforcement paths resolved a user and treated a
// caller without one as a caller that failed.
func TestEnforcement_MachineCredentialReachesGuardedRoute(t *testing.T) {
	for _, kind := range []principal.Kind{
		principal.KindService,
		principal.KindWorkload,
		principal.KindAgent,
	} {
		t.Run(string(kind), func(t *testing.T) {
			svcID := id.NewServiceAccountID()
			sess := &session.Session{
				ID:               id.NewSessionID(),
				AppID:            id.NewAppID(),
				PrincipalKind:    kind,
				ServiceAccountID: svcID,
				Roles:            []string{"deployer"},
				Token:            "machine-token",
			}

			mw := middleware.AuthMiddleware(
				func(token string) (*session.Session, error) {
					if token == "machine-token" {
						return sess, nil
					}
					return nil, errors.New("invalid")
				},
				func(string) (*user.User, error) {
					return nil, errors.New("no user behind a machine principal")
				},
				log.NewNoopLogger(),
			)

			var gotRef principal.Ref
			var refOK bool

			router := forge.NewRouter()
			router.Use(mw)
			router.Use(middleware.RequireAuth())
			router.GET("/guarded", func(ctx forge.Context) error {
				s, ok := middleware.SessionFrom(ctx.Context())
				require.True(t, ok, "the session must reach the handler")
				assert.Equal(t, kind, s.PrincipalKind)
				gotRef, refOK = middleware.PrincipalRefFrom(ctx.Context())
				return ctx.NoContent(http.StatusOK)
			})

			req, err := http.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer machine-token")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code,
				"a %s credential must reach a RequireAuth-guarded route", kind)
			require.True(t, refOK, "the handler must see a resolved principal")
			assert.Equal(t, kind, gotRef.Kind)
			assert.Equal(t, svcID.String(), gotRef.ID)
		})
	}
}

// The regression guard, and it matters more than the cases above. A human
// credential must behave exactly as it did before any of this.
func TestEnforcement_HumanCredentialUnchanged(t *testing.T) {
	userID := id.NewUserID()
	appID := id.NewAppID()
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  appID,
		UserID: userID,
		Token:  "human-token",
	}
	u := &user.User{ID: userID, AppID: appID, Email: "human@example.com"}

	mw := middleware.AuthMiddleware(
		func(string) (*session.Session, error) { return sess, nil },
		func(idStr string) (*user.User, error) {
			if idStr == userID.String() {
				return u, nil
			}
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	router.GET("/guarded", func(ctx forge.Context) error {
		got, ok := middleware.UserFrom(ctx.Context())
		require.True(t, ok, "UserFrom must still work for humans")
		assert.Equal(t, "human@example.com", got.Email)

		ref, refOK := middleware.PrincipalRefFrom(ctx.Context())
		require.True(t, refOK)
		assert.Equal(t, principal.KindUser, ref.Kind)
		assert.Equal(t, userID.String(), ref.ID)
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer human-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// Pins the boundary, so widening who counts as authenticated cannot drift
// into "anyone at all".
func TestEnforcement_AnonymousStillRejected(t *testing.T) {
	mw := middleware.AuthMiddleware(
		func(string) (*session.Session, error) { return nil, errors.New("invalid") },
		func(string) (*user.User, error) { return nil, errors.New("not found") },
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	router.GET("/guarded", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
