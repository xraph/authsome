package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/session"
)

// guardedRouter builds a router whose routes all sit behind
// GuardDelegatedWrites: a write, a write marked agent-safe, a read, and a write
// in a nested group, which inherits the guard.
func guardedRouter(t *testing.T, outer ...forge.Middleware) forge.Router {
	t.Helper()
	root := forge.NewRouter()
	for _, mw := range outer {
		root.Use(mw)
	}
	g := middleware.GuardDelegatedWrites(root)
	ok := func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) }
	require.NoError(t, g.POST("/write", ok))
	require.NoError(t, g.POST("/safe", ok, middleware.AgentSafe()))
	require.NoError(t, g.GET("/read", ok))
	require.NoError(t, g.Group("/nested").DELETE("/write", ok))
	return root
}

func TestGuardDelegatedWrites_DenyByDefault(t *testing.T) {
	user := id.NewUserID()
	sessions := map[string]*session.Session{
		"user's own": {ID: id.NewSessionID(), UserID: user},
		"agent":      {ID: id.NewSessionID(), UserID: user, PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID()},
		"service account acting as itself": {
			ID: id.NewSessionID(), PrincipalKind: session.PrincipalKindAgent, ServiceAccountID: id.NewServiceAccountID(),
		},
	}
	routes := []struct{ method, path string }{
		{http.MethodPost, "/write"},
		{http.MethodPost, "/safe"},
		{http.MethodGet, "/read"},
		{http.MethodDelete, "/nested/write"},
	}
	for name, sess := range sessions {
		for _, rt := range routes {
			t.Run(name+" "+rt.method+" "+rt.path, func(t *testing.T) {
				router := guardedRouter(t)
				ctx := middleware.WithSession(context.Background(), sess)
				req := httptest.NewRequestWithContext(ctx, rt.method, rt.path, nil)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				want := http.StatusOK
				if sess.IsDelegated() && rt.method != http.MethodGet && rt.path != "/safe" {
					want = http.StatusForbidden
				}
				assert.Equal(t, want, rec.Code, "body=%s", rec.Body.String())
			})
		}
	}
}

// In client mode the caller arrives through introspection, with no session
// row. The guard reads the marker ClientAuthMiddleware leaves, so the same
// default holds in a client-mode service.
func TestGuardDelegatedWrites_ClientMode(t *testing.T) {
	for name, delegated := range map[string]bool{"user's own": false, "delegated": true} {
		t.Run(name, func(t *testing.T) {
			userID := id.NewUserID().String()
			identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(authclient.IntrospectResponse{
					Active: true, UserID: userID, AppID: id.NewAppID().String(), SessionID: id.NewSessionID().String(),
					PrincipalKind: "user", Delegated: delegated,
				})
			}))
			t.Cleanup(identity.Close)
			router := guardedRouter(t, middleware.ClientAuthMiddleware(authclient.NewClient(identity.URL), log.NewNoopLogger()))

			do := func(method, path string) int {
				req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
				req.Header.Set("Authorization", "Bearer client-token")
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				return rec.Code
			}
			wantWrite := http.StatusOK
			if delegated {
				wantWrite = http.StatusForbidden
			}
			assert.Equal(t, wantWrite, do(http.MethodPost, "/write"))
			assert.Equal(t, wantWrite, do(http.MethodDelete, "/nested/write"))
			assert.Equal(t, http.StatusOK, do(http.MethodPost, "/safe"))
			assert.Equal(t, http.StatusOK, do(http.MethodGet, "/read"))
		})
	}
}
