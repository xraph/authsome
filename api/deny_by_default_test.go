package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/plugins/consent"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/plugins/sharedsignals"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/sso"
	"github.com/xraph/authsome/plugins/subscription"
	"github.com/xraph/authsome/plugins/waitlist"
)

// agentSafeWrites is every write route a delegated session may call. Keep it
// short: each entry is a route where the delegated caller can only act on its
// own credential, never on what its user owns.
var agentSafeWrites = map[string]bool{
	"POST /v1/signout":      true, // ends the caller's own session, nothing else
	"POST /v1/oauth/revoke": true, // revokes a token the caller itself holds (callerOwnsSession)
}

// denyByDefaultFixture builds the engine with every plugin that serves
// routes, wires it the way the Forge extension does (auth middleware global,
// core and plugin routes behind the delegated-write guard), and mints a real
// agent token for the platform owner.
type denyByDefaultFixture struct {
	eng        *authsome.Engine
	handler    http.Handler
	routes     []forge.RouteInfo
	orgs       *organization.Plugin
	owner      id.UserID
	ownerToken string
	agentToken string
	appID      id.AppID
	agents     *agentauth.Plugin
	grant      *agentauth.AgentGrant
}

// freshAgentToken issues another agent session under the same grant, so a
// route that ends one (signout is agent-safe) doesn't spoil the next case.
func (f *denyByDefaultFixture) freshAgentToken(t *testing.T) string {
	t.Helper()
	sess, err := f.agents.IssueAgentSession(context.Background(), f.grant, agentauth.IssueMeta{})
	require.NoError(t, err)
	return sess.Token
}

func newDenyByDefaultFixture(t *testing.T) *denyByDefaultFixture {
	t.Helper()
	agents := agentauth.New(agentauth.WithStore(agentauth.NewMemoryStore()))
	wl := waitlist.New()
	wl.SetWaitlistStore(waitlist.NewMemoryStore())
	ssf := sharedsignals.New()
	ssf.SetStore(sharedsignals.NewMemoryStore())
	orgs := organization.New()
	a, eng := newBootstrappedAPI(t,
		authsome.WithPlugin(password.New()),
		authsome.WithPlugin(social.New(social.Config{})),
		authsome.WithPlugin(mfa.New(mfa.Config{})),
		authsome.WithPlugin(apikey.New()),
		authsome.WithPlugin(magiclink.New(magiclink.Config{})),
		authsome.WithPlugin(sso.New(sso.Config{})),
		authsome.WithPlugin(passkey.New(passkey.Config{RPID: "localhost"})),
		authsome.WithPlugin(orgs),
		authsome.WithPlugin(scim.New()),
		authsome.WithPlugin(subscription.New()),
		authsome.WithPlugin(phone.New()),
		authsome.WithPlugin(consent.New()),
		authsome.WithPlugin(oauth2provider.New()),
		authsome.WithPlugin(agents),
		authsome.WithPlugin(wl),
		authsome.WithPlugin(ssf),
	)
	ctx := context.Background()

	root := forge.NewRouter()
	root.UseGlobal(eng.AuthMiddleware())
	require.NoError(t, a.RegisterRoutes(root))
	pluginRouter := middleware.GuardDelegatedWrites(root)
	for _, rp := range eng.Plugins().RouteProviders() {
		require.NoError(t, rp.RegisterRoutes(pluginRouter))
	}

	_, ownerToken, _ := signUp(t, eng, "deny-default-owner@test.com", "SecureP@ss123")
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

	return &denyByDefaultFixture{
		eng: eng, handler: root.Handler(), routes: root.Routes(), orgs: orgs,
		owner: owner, ownerToken: ownerToken, agentToken: agentSess.Token, appID: appID,
		agents: agents, grant: grant,
	}
}

func (f *denyByDefaultFixture) call(t *testing.T, token, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequestWithContext(context.Background(), method, path, nil)
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(middleware.PublishableKeyHeader, testPublishableKey)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

var pathParam = regexp.MustCompile(`:[A-Za-z_]+|\{[^}]+\}`)

// Every write route authsome registers refuses an agent session unless it is
// on the agent-safe list. Enumerating the router is the point: a route added
// tomorrow without the marker shows up here as refused, which is what deny by
// default means.
func TestDenyByDefault_EveryWriteRouteRefusesAgentSession(t *testing.T) {
	f := newDenyByDefaultFixture(t)

	writes := 0
	for _, r := range f.routes {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			continue
		}
		if strings.Contains(r.Path, "*") {
			continue
		}
		writes++
		path := pathParam.ReplaceAllString(r.Path, "x")
		key := r.Method + " " + r.Path
		t.Run(key, func(t *testing.T) {
			rec := f.call(t, f.freshAgentToken(t), r.Method, path, `{}`)
			if agentSafeWrites[key] {
				assert.NotContains(t, rec.Body.String(), "delegated session", "an agent-safe route lets the agent through; body=%s", rec.Body.String())
				return
			}
			assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
			// The guard is an interceptor, so it runs after route middleware.
			// A route whose own permission middleware already refused the
			// delegating user (the billing writes, for this owner) answers
			// with that refusal first. Either way nothing runs.
			body := rec.Body.String()
			assert.True(t, strings.Contains(body, "delegated session") || strings.Contains(body, "insufficient permissions"),
				"refused by the delegated-session deny or an earlier permission check; body=%s", body)
		})
	}
	assert.Greater(t, writes, 50, "the table has to cover the real router, not an empty one")
	for key := range agentSafeWrites {
		found := false
		for _, r := range f.routes {
			if r.Method+" "+r.Path == key {
				found = true
			}
		}
		assert.True(t, found, "agent-safe route %s is registered", key)
	}
}

// The cases the review proved one by one: each refused for the agent with
// nothing written, and each still open to the owner's own session.
func TestDenyByDefault_ProvenCasesWriteNothing(t *testing.T) {
	f := newDenyByDefaultFixture(t)
	ctx := context.Background()

	t.Run("DELETE /v1/me", func(t *testing.T) {
		rec := f.call(t, f.agentToken, http.MethodDelete, "/v1/me", "")
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		_, err := f.eng.AdminGetUser(ctx, f.owner)
		require.NoError(t, err, "the account still exists")
	})

	t.Run("POST /v1/environments", func(t *testing.T) {
		before, err := f.eng.ListEnvironments(ctx, f.appID)
		require.NoError(t, err)
		body := `{"name":"Agent","slug":"agent-env","type":"development"}`

		rec := f.call(t, f.agentToken, http.MethodPost, "/v1/environments", body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		after, err := f.eng.ListEnvironments(ctx, f.appID)
		require.NoError(t, err)
		assert.Len(t, after, len(before), "no environment is created")

		rec = f.call(t, f.ownerToken, http.MethodPost, "/v1/environments", body)
		assert.Less(t, rec.Code, 300, "the owner still creates one; body=%s", rec.Body.String())
	})

	t.Run("POST /v1/orgs", func(t *testing.T) {
		before, err := f.orgs.ListUserOrganizations(ctx, f.owner)
		require.NoError(t, err)
		body := `{"name":"Agent Org","slug":"agent-org"}`

		rec := f.call(t, f.agentToken, http.MethodPost, "/v1/orgs", body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		after, err := f.orgs.ListUserOrganizations(ctx, f.owner)
		require.NoError(t, err)
		assert.Len(t, after, len(before), "no org is created")

		rec = f.call(t, f.ownerToken, http.MethodPost, "/v1/orgs", body)
		assert.Less(t, rec.Code, 300, "the owner still creates one; body=%s", rec.Body.String())
	})

	t.Run("POST /v1/webhooks", func(t *testing.T) {
		before, err := f.eng.ListWebhooks(ctx, f.appID)
		require.NoError(t, err)
		body := `{"url":"https://hooks.example.com/agent","events":["user.created"]}`

		rec := f.call(t, f.agentToken, http.MethodPost, "/v1/webhooks", body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		after, err := f.eng.ListWebhooks(ctx, f.appID)
		require.NoError(t, err)
		assert.Len(t, after, len(before), "no webhook is created")
	})

	t.Run("POST /v1/billing/plans", func(t *testing.T) {
		body := `{"name":"Agent Plan","slug":"agent-plan","currency":"usd"}`
		rec := f.call(t, f.agentToken, http.MethodPost, "/v1/billing/plans", body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())

		list := f.call(t, f.ownerToken, http.MethodGet, "/v1/billing/plans?app_id="+f.appID.String(), "")
		assert.NotContains(t, list.Body.String(), "agent-plan", "no plan is created")
	})

	t.Run("reads stay open", func(t *testing.T) {
		rec := f.call(t, f.agentToken, http.MethodGet, "/v1/me", "")
		assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("signout is agent-safe", func(t *testing.T) {
		rec := f.call(t, f.agentToken, http.MethodPost, "/v1/signout", "")
		assert.Less(t, rec.Code, 300, "the agent can end its own session; body=%s", rec.Body.String())
		me := f.call(t, f.agentToken, http.MethodGet, "/v1/me", "")
		assert.NotEqual(t, http.StatusOK, me.Code, "and the token stops working")
	})
}
