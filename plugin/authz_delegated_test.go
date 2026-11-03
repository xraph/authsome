package plugin_test

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
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
)

// AdminGuard is what every plugin admin group mounts (OAuth2 clients, SSO
// connections, social providers, agents). Its permission check runs against
// the session's user, which for an agent is the delegating admin, so the guard
// itself refuses an agent or delegated session.
func TestAdminGuard_RefusesDelegatedSession(t *testing.T) {
	user := id.NewUserID()
	cases := map[string]struct {
		sess *session.Session
		want int
	}{
		"agent":      {&session.Session{ID: id.NewSessionID(), UserID: user, PrincipalKind: session.PrincipalKindAgent}, http.StatusForbidden},
		"user's own": {&session.Session{ID: id.NewSessionID(), UserID: user}, http.StatusOK},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			router := forge.NewRouter()
			g := router.Group("/v1/admin/things", forge.WithGroupMiddleware(plugin.AdminGuard(nil, "manage", "thing")...))
			require.NoError(t, g.POST("", func(ctx forge.Context) error {
				return ctx.JSON(http.StatusOK, map[string]string{"ok": "1"})
			}))

			ctx := middleware.WithSession(context.Background(), tc.sess)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/admin/things", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code, "body=%s", rec.Body.String())
		})
	}
}
