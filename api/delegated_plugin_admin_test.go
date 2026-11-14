package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/plugins/sharedsignals"
	"github.com/xraph/authsome/plugins/waitlist"
)

// The organization, waitlist and shared-signals admin groups each build their
// own middleware chain rather than going through plugin.AdminGuard. A real
// agent token, resolved by the engine's auth middleware and the group's own
// session guard, gets 403 on each and changes nothing; the owner's own token
// still gets through.
func TestPluginAdminGroups_RefuseAgentToken(t *testing.T) {
	agents := agentauth.New(agentauth.WithStore(agentauth.NewMemoryStore()))
	wl := waitlist.New()
	wlStore := waitlist.NewMemoryStore()
	wl.SetWaitlistStore(wlStore)
	ssf := sharedsignals.New()
	ssf.SetStore(sharedsignals.NewMemoryStore())
	a, eng := newBootstrappedAPI(t,
		authsome.WithPlugin(agents),
		authsome.WithPlugin(organization.New()),
		authsome.WithPlugin(wl),
		authsome.WithPlugin(ssf),
	)
	ctx := context.Background()

	root := forge.NewRouter()
	root.UseGlobal(eng.AuthMiddleware())
	require.NoError(t, a.RegisterRoutes(root))
	for _, rp := range eng.Plugins().RouteProviders() {
		require.NoError(t, rp.RegisterRoutes(root))
	}
	handler := root.Handler()

	_, ownerToken, _ := signUp(t, eng, "plugin-admin-owner@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(), UserID: owner, OrgID: id.NewOrgID(),
		Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, agents.Store().CreateAgentGrant(ctx, grant))
	agentSess, err := agents.IssueAgentSession(ctx, grant, agentauth.IssueMeta{})
	require.NoError(t, err)

	call := func(token, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		var req *http.Request
		if body == "" {
			req = httptest.NewRequestWithContext(ctx, method, path, nil)
		} else {
			req = httptest.NewRequestWithContext(ctx, method, path, jsonBody(t, rawJSON(body)))
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(middleware.PublishableKeyHeader, testPublishableKey)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	t.Run("organization admin", func(t *testing.T) {
		rec := call(agentSess.Token, http.MethodGet, "/v1/admin/orgs?app_id="+appID.String(), "")
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "delegated session")

		rec = call(ownerToken, http.MethodGet, "/v1/admin/orgs?app_id="+appID.String(), "")
		assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("waitlist admin", func(t *testing.T) {
		entry := &waitlist.WaitlistEntry{
			ID: id.NewWaitlistID(), AppID: appID, Email: "wl-agent@test.com", Status: waitlist.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, wlStore.CreateEntry(ctx, entry))
		path := "/v1/admin/waitlist/" + entry.ID.String() + "/approve"

		rec := call(agentSess.Token, http.MethodPost, path, `{}`)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "delegated session")
		got, err := wlStore.GetEntry(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, waitlist.StatusPending, got.Status, "the entry stays pending")

		// The seeded roles grant nobody manage:waitlist, so the owner stops at
		// the permission check. What matters here is that the owner's own
		// session gets past the delegated-session deny to reach it.
		rec = call(ownerToken, http.MethodPost, path, `{}`)
		assert.NotContains(t, rec.Body.String(), "delegated session", "body=%s", rec.Body.String())
	})

	t.Run("shared signals admin", func(t *testing.T) {
		rec := call(agentSess.Token, http.MethodPost, "/v1/ssf/admin/streams",
			`{"name":"agent","issuer":"https://tx.example.com","jwks_uri":"https://tx.example.com/jwks"}`)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "delegated session")

		rec = call(ownerToken, http.MethodGet, "/v1/ssf/admin/streams", "")
		require.Equal(t, http.StatusOK, rec.Code, "the owner still lists streams; body=%s", rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "tx.example.com", "no stream was created")
	})
}
