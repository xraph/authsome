package passkey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// The registration handlers refuse a delegated session on their own, so a
// mount that forgets the route guard still adds no passkey.
func TestRegisterHandlers_RefuseAgentSessionWithoutRouteGuard(t *testing.T) {
	p := New(Config{RPID: "localhost"})
	router := forge.NewRouter()
	require.NoError(t, router.POST("/bare/begin", p.handleRegisterBegin))
	require.NoError(t, router.POST("/bare/finish", p.handleRegisterFinish))

	for _, path := range []string{"/bare/begin", "/bare/finish"} {
		u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), Email: "pk@example.com"}
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:3000")
		ctx := middleware.WithUser(req.Context(), u)
		ctx = middleware.WithUserID(ctx, u.ID)
		ctx = middleware.WithSession(ctx, &session.Session{
			ID: id.NewSessionID(), UserID: u.ID, CreatedAt: time.Now(),
			PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
		})
		rec := httptest.NewRecorder()
		router.Handler().ServeHTTP(rec, req.WithContext(ctx))
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s body=%s", path, rec.Body.String())
	}
}
