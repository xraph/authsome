package agentauth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
)

// withAgentPrincipalSession layers an agent-principal session onto ctx the
// way authprovider/session.go would once a real agent session resolves —
// AgentID and GrantID both set, matching what IssueAgentSession stamps.
func withAgentPrincipalSession(ctx context.Context, userID id.UserID, appID id.AppID) context.Context {
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	}
	return middleware.WithSession(ctx, sess)
}

// Final review item 3: agentauth's own admin and /me routes are guarded by
// plugin.SessionGuard and plugin.AdminGuard, both of which resolve to
// middleware.RequirePermission — not principal-kind aware. Once issuance is
// wired, an agent session belonging to an org admin could reach these routes
// carrying that admin's full permission and no granted scope at all: set its
// own governing org's policy wide open (PUT /admin/agents/policy), unblock
// itself (PATCH /admin/agents/:id/status), or revoke a competing grant
// (DELETE /me/agents/:id). The plugin ships the intersection guard for every
// other route; it must not leave its own policy surface outside it. These
// prove at least one route in each group refuses an agent-principal session
// outright, before the handler runs.
func TestAdminRoutes_RefuseAgentPrincipalSession(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	userID, appID, orgID := id.NewUserID(), id.NewAppID(), id.NewOrgID()
	ctx := adminCtx(userID, appID, orgID)
	ctx = withAgentPrincipalSession(ctx, userID, appID)

	w := doRequest(ctx, t, mux, http.MethodPut, "/v1/admin/agents/policy",
		map[string]any{"org_id": orgID.String(), "mode": "open"})

	assert.Equal(t, http.StatusForbidden, w.Code,
		"an agent-principal session must never be able to set its own governing org's policy")
}

func TestMeRoutes_RefuseAgentPrincipalSession(t *testing.T) {
	agentStore, mux := newRoutesTestPlugin(t)
	userID, appID, orgID := id.NewUserID(), id.NewAppID(), id.NewOrgID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, OrgID: orgID, ClientID: "client_agentguard",
		Name: "Agent", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	require.NoError(t, agentStore.CreateAgent(context.Background(), agent))
	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: agent.ID, UserID: userID,
		OrgID: orgID, Scopes: []string{"invoices:read"},
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), grant))

	ctx := adminCtx(userID, appID, orgID)
	ctx = withAgentPrincipalSession(ctx, userID, appID)

	w := doRequest(ctx, t, mux, http.MethodDelete, "/v1/me/agents/"+grant.ID.String(), nil)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"an agent-principal session must never be able to revoke a delegation, competing or its own")
}
