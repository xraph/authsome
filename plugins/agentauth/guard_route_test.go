package agentauth_test

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
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// Every authsome router is deny by default for a delegated session on a
// write. A route behind Guard enforces the agent's grant itself, so
// GuardRoute marks it agent-safe as well, and the agent's grant is what
// decides. Guard on its own leaves the route closed to agents.
func TestGuardRoute_OpensAWriteRouteToTheGrantOnly(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	// The user may do both, so only the grant separates the two routes.
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice":  true,
		userID.String() + "|write|invoice": true,
	}})

	root := forge.NewRouter()
	g := middleware.GuardDelegatedWrites(root)
	ok := func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) }
	require.NoError(t, g.POST("/granted", ok, p.GuardRoute("read", "invoice")...))
	require.NoError(t, g.POST("/outside-grant", ok, p.GuardRoute("write", "invoice")...))
	require.NoError(t, g.POST("/guard-only", ok, forge.WithMiddleware(p.Guard("read", "invoice"))))

	do := func(path string) int {
		ctx := middleware.WithSession(context.Background(), sess)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, do("/granted"), "the grant covers read on invoice")
	assert.Equal(t, http.StatusForbidden, do("/outside-grant"), "the grant does not cover write")
	assert.Equal(t, http.StatusForbidden, do("/guard-only"), "without the agent-safe marker the route stays closed")
}

// A session minted by token exchange is delegated but not an agent session,
// so Authorize used to let it straight through, and GuardRoute's agent-safe
// marker then let it write with no scope check at all. It now has to hold a
// registered scope that covers the route, and its user has to be allowed too,
// the same intersection an agent gets.
func TestGuardRoute_ExchangedSessionNeedsCoveringScope(t *testing.T) {
	p, _, _, userID := agentSetup(t, nil, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice":  true,
		userID.String() + "|write|invoice": true,
	}})

	ran := map[string]bool{}
	root := forge.NewRouter()
	g := middleware.GuardDelegatedWrites(root)
	handler := func(name string) func(forge.Context) error {
		return func(ctx forge.Context) error {
			ran[name] = true
			return ctx.NoContent(http.StatusOK)
		}
	}
	require.NoError(t, g.POST("/read", handler("read"), p.GuardRoute("read", "invoice")...))
	require.NoError(t, g.POST("/write", handler("write"), p.GuardRoute("write", "invoice")...))

	exchanged := func(scopes ...string) *session.Session {
		return &session.Session{
			ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: userID,
			Actors:       principal.Chain{{Kind: principal.KindWorkload, ID: "svc_actor"}},
			ActorGrant:   principal.GrantDelegation,
			DelegationID: id.NewDelegationID(),
			Scopes:       scopes,
		}
	}
	do := func(sess *session.Session, path string) int {
		ctx := middleware.WithSession(context.Background(), sess)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, req)
		return rec.Code
	}

	readOnly := exchanged("invoices:read")
	assert.Equal(t, http.StatusForbidden, do(readOnly, "/write"), "invoices:read does not cover write")
	assert.False(t, ran["write"], "the handler never runs")
	assert.Equal(t, http.StatusOK, do(readOnly, "/read"), "invoices:read covers read")

	assert.Equal(t, http.StatusForbidden, do(exchanged(), "/read"), "no scopes cover nothing")
	assert.Equal(t, http.StatusForbidden, do(exchanged("repo:read"), "/read"), "an unregistered scope covers nothing")

	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{}})
	assert.Equal(t, http.StatusForbidden, do(readOnly, "/read"), "the user has to be allowed as well")
}
