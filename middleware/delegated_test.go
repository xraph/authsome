package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

func TestDenyDelegatedSession(t *testing.T) {
	user := id.NewUserID()
	cases := map[string]struct {
		sess *session.Session
		want int
	}{
		"no session":      {nil, http.StatusOK},
		"user's own":      {&session.Session{ID: id.NewSessionID(), UserID: user}, http.StatusOK},
		"agent":           {&session.Session{ID: id.NewSessionID(), UserID: user, PrincipalKind: session.PrincipalKindAgent}, http.StatusForbidden},
		"service account": {&session.Session{ID: id.NewSessionID(), PrincipalKind: session.PrincipalKindServiceAccount, ServiceAccountID: id.NewServiceAccountID()}, http.StatusOK},
		"agent-kind service account acting as itself": {&session.Session{ID: id.NewSessionID(), PrincipalKind: session.PrincipalKindAgent, ServiceAccountID: id.NewServiceAccountID()}, http.StatusOK},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ran := false
			router := forge.NewRouter()
			require.NoError(t, router.POST("/mint", func(ctx forge.Context) error {
				ran = true
				return ctx.JSON(http.StatusOK, map[string]string{"ok": "1"})
			}, forge.WithMiddleware(middleware.DenyDelegatedSession())))

			ctx := context.Background()
			if tc.sess != nil {
				ctx = middleware.WithSession(ctx, tc.sess)
			}
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/mint", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tc.want, rec.Code, "body=%s", rec.Body.String())
			assert.Equal(t, tc.want == http.StatusOK, ran, "the handler runs only when the session is the user's own")
		})
	}
}

func TestRefuseDelegated(t *testing.T) {
	assert.NoError(t, middleware.RefuseDelegated(context.Background()))
	own := middleware.WithSession(context.Background(), &session.Session{UserID: id.NewUserID()})
	assert.NoError(t, middleware.RefuseDelegated(own))
	agent := middleware.WithSession(context.Background(), &session.Session{
		UserID: id.NewUserID(), PrincipalKind: session.PrincipalKindAgent,
	})
	assert.ErrorIs(t, middleware.RefuseDelegated(agent), middleware.ErrDelegatedSession)
}
