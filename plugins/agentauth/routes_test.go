package agentauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
)

// ──────────────────────────────────────────────────
// Test harness
// ──────────────────────────────────────────────────

// newRoutesTestPlugin builds a plugin wired to a stub engine (OnInit run, so
// p.basePath and p.engine are set the way production does) with its routes
// registered on a fresh router. Nothing in stubEngine implements
// PermissionChecker and its AuthRegistry() is nil, so — exactly like
// production when those container entries are missing — SessionGuard and
// AdminGuard contribute no middleware at all. Every handler's own
// middleware.UserIDFrom check is therefore the only thing standing between
// an anonymous request and the handler running; that is precisely what I1
// exercises.
func newRoutesTestPlugin(t *testing.T) (*agentauth.MemoryStore, forge.Router) {
	t.Helper()
	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	eng := &stubEngine{
		store:    memory.New(),
		registry: plugin.NewRegistry(log.NewNoopLogger()),
		bus:      hook.NewBus(log.NewNoopLogger()),
		logger:   log.NewNoopLogger(),
	}
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return agentStore, mux
}

// authsomeBasePathEngine overrides stubEngine.BasePath() to return the real
// production default ("/authsome" — set by extension/extension.go), so a
// test can prove OnInit does NOT fold it into p.basePath. The router in
// these tests plays the part of the group extension.go already creates at
// that base path; RegisterRoutes must add only the version segment on top
// of it, not the base path a second time.
type authsomeBasePathEngine struct {
	*stubEngine
}

func (e *authsomeBasePathEngine) BasePath() string { return "/authsome" }

// adminCtx builds a context carrying a resolved identity: userID from
// session auth, appID and orgID the way authprovider/session.go populates
// them from the resolved session (WithAppID/WithOrgID) once real auth
// middleware runs. A zero orgID is left off entirely, matching an
// app-scoped session with no org.
func adminCtx(userID id.UserID, appID id.AppID, orgID id.OrgID) context.Context {
	ctx := middleware.WithUserID(context.Background(), userID)
	ctx = middleware.WithAppID(ctx, appID)
	if !orgID.IsNil() {
		ctx = middleware.WithOrgID(ctx, orgID)
	}
	return ctx
}

// doRequest issues an HTTP request against mux, JSON-encoding body when
// non-nil. Used for both query-string GETs (pass the query in path) and
// JSON-bodied POST/PUT/PATCH/DELETE.
func doRequest(ctx context.Context, t *testing.T, mux forge.Router, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	r := bytes.NewReader(nil)
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequestWithContext(ctx, method, path, r)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// ──────────────────────────────────────────────────
// C1 — routes must mount under exactly /v1, not doubled with engine.BasePath()
// ──────────────────────────────────────────────────

// TestRegisterRoutes_MountsUnderV1Only proves the actual bug: OnInit used to
// set p.basePath from engine.BasePath(), which defaults to "/authsome" — the
// same base path extension/extension.go already groups the router at before
// handing it to plugins. That doubled up to "/authsome/v1/me/agents" (and,
// worse, "/authsome/authsome/me/agents" when BasePath() truly is the
// default, since RegisterRoutes' own literal segments never used a version
// prefix without this fix). p.basePath must be exactly "/v1" regardless of
// what the engine's own base path is.
func TestRegisterRoutes_MountsUnderV1Only(t *testing.T) {
	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	inner := &stubEngine{
		store:    memory.New(),
		registry: plugin.NewRegistry(log.NewNoopLogger()),
		bus:      hook.NewBus(log.NewNoopLogger()),
		logger:   log.NewNoopLogger(),
	}
	eng := &authsomeBasePathEngine{stubEngine: inner}
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	ctx := middleware.WithUserID(context.Background(), id.NewUserID())

	w := doRequest(ctx, t, mux, http.MethodGet, "/v1/me/agents", nil)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "GET /v1/me/agents must be mounted")

	w = doRequest(ctx, t, mux, http.MethodGet, "/authsome/v1/me/agents", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "routes must not be doubled under engine.BasePath()")

	w = doRequest(ctx, t, mux, http.MethodGet, "/me/agents", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "routes must not be mounted at the router root either")
}

// TestRegisterRoutes_AllSixPathsAreReachable is the coverage requirement:
// nothing before this fix round exercised RegisterRoutes or any handleX
// function at all, and every Critical the reviewer found lived in exactly
// that untested layer. This drives every one of the six routes through a
// real router at least once.
func TestRegisterRoutes_AllSixPathsAreReachable(t *testing.T) {
	agentStore, mux := newRoutesTestPlugin(t)
	userID, appID, orgID := id.NewUserID(), id.NewAppID(), id.NewOrgID()
	ctx := adminCtx(userID, appID, orgID)

	// handleRevokeMyGrant and handleSetAgentStatus both correctly answer 404
	// for a genuinely nonexistent grant/agent, which makes a random id
	// useless for telling "route not registered" apart from "route
	// registered, resource legitimately missing" by status code alone. Seed
	// a real grant and a real agent so a 404 here can only mean the former.
	realAgent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, OrgID: orgID, ClientID: "client_route_reach",
		Name: "Reachability Agent", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	require.NoError(t, agentStore.CreateAgent(context.Background(), realAgent))
	realGrant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: realAgent.ID, UserID: userID,
		OrgID: orgID, Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), realGrant))

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"list my grants", http.MethodGet, "/v1/me/agents", nil},
		{"revoke my grant", http.MethodDelete, "/v1/me/agents/" + realGrant.ID.String(), nil},
		{"list agents", http.MethodGet, "/v1/admin/agents", nil},
		{"register agent", http.MethodPost, "/v1/admin/agents", map[string]any{"client_id": "client_reachable", "name": "N"}},
		{"set agent status", http.MethodPatch, "/v1/admin/agents/" + realAgent.ID.String() + "/status", map[string]any{"status": "blocked"}},
		{"put org policy", http.MethodPut, "/v1/admin/agents/policy", map[string]any{"org_id": orgID.String(), "mode": "open"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(ctx, t, mux, tc.method, tc.path, tc.body)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route must be registered: %s %s (body: %s)", tc.method, tc.path, w.Body.String())
		})
	}
}

// ──────────────────────────────────────────────────
// C2 — a client_id must resolve to at most one agent
// ──────────────────────────────────────────────────

func TestRegisterAgent_RejectsDuplicateClientID(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	ctx := adminCtx(id.NewUserID(), id.NewAppID(), id.OrgID{})

	first := doRequest(ctx, t, mux, http.MethodPost, "/v1/admin/agents",
		map[string]any{"client_id": "client_dup", "name": "Agent One"})
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	second := doRequest(ctx, t, mux, http.MethodPost, "/v1/admin/agents",
		map[string]any{"client_id": "client_dup", "name": "Agent Two"})
	assert.Equal(t, http.StatusConflict, second.Code,
		"a client_id that already resolves to an agent must be rejected, not silently duplicated")
}

// TestBlockedAgent_StaysBlockedAfterDuplicateRegistrationAttempt is the
// concrete failure mode the duplicate check exists to prevent: without it,
// GetAgentByClientID ranges over a Go map and returns whichever of the two
// agents sharing a ClientID it visits first, in nondeterministic order —
// so a blocked agent's client id would resolve as blocked only about half
// the time Evaluate (or, here, GetAgentByClientID itself) is asked. This
// guard-verifies deterministically: rejecting the second registration
// outright, rather than relying on which of two records a lookup happens to
// return, is what removes the coin flip.
func TestBlockedAgent_StaysBlockedAfterDuplicateRegistrationAttempt(t *testing.T) {
	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	appID, orgID, userID := id.NewAppID(), id.NewOrgID(), id.NewUserID()

	original, err := p.RegisterAgent(context.Background(), &agentauth.Agent{
		AppID: appID, OrgID: orgID, ClientID: "client_blockme", Name: "Original", CreatedBy: userID,
	})
	require.NoError(t, err)
	require.NoError(t, p.SetAgentStatus(context.Background(), original.ID, orgID, agentauth.StatusBlocked))

	_, err = p.RegisterAgent(context.Background(), &agentauth.Agent{
		AppID: appID, OrgID: orgID, ClientID: "client_blockme", Name: "Impersonator", CreatedBy: userID,
	})
	require.Error(t, err, "re-registering an already-used client_id must be rejected outright")

	got, err := agentStore.GetAgentByClientID(context.Background(), "client_blockme")
	require.NoError(t, err)
	assert.Equal(t, original.ID.String(), got.ID.String(), "the client_id must still resolve to the original agent")
	assert.Equal(t, agentauth.StatusBlocked, got.Status, "the original blocked agent must still be the one found, and must still be blocked")
}

// ──────────────────────────────────────────────────
// C3 — the admin routes must not trust app_id/org_id from the request
// ──────────────────────────────────────────────────

// TestHandlePutOrgPolicy_RefusesCrossTenantWrite is the minimum the reviewer
// asked for: HasPermission("write", "agent") carries no org dimension of its
// own (rbac/warden_store.go never reads UserRole.OrgID), so without
// resolving the caller's org from context and rejecting a mismatch, any
// admin caller in ANY org could flip a different org's delegation policy —
// including to ModeOpen with every scope allowed.
func TestHandlePutOrgPolicy_RefusesCrossTenantWrite(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	myOrg, otherOrg := id.NewOrgID(), id.NewOrgID()
	ctx := adminCtx(id.NewUserID(), id.NewAppID(), myOrg)

	w := doRequest(ctx, t, mux, http.MethodPut, "/v1/admin/agents/policy",
		map[string]any{"org_id": otherOrg.String(), "mode": "open"})

	assert.Equal(t, http.StatusForbidden, w.Code,
		"a caller must not be able to set another organization's delegation policy")
}

func TestHandlePutOrgPolicy_AllowsSameTenantWrite(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	myOrg := id.NewOrgID()
	ctx := adminCtx(id.NewUserID(), id.NewAppID(), myOrg)

	w := doRequest(ctx, t, mux, http.MethodPut, "/v1/admin/agents/policy",
		map[string]any{"org_id": myOrg.String(), "mode": "open"})

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestHandleListAgents_RefusesCrossAppEnumeration closes the enumeration
// hole named explicitly: GET took app_id straight from the query with no
// check against the caller's own app.
func TestHandleListAgents_RefusesCrossAppEnumeration(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	myApp, otherApp := id.NewAppID(), id.NewAppID()
	ctx := adminCtx(id.NewUserID(), myApp, id.OrgID{})

	w := doRequest(ctx, t, mux, http.MethodGet, "/v1/admin/agents?app_id="+otherApp.String(), nil)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"a caller must not be able to list another application's agents by naming its app_id in the query")
}

// TestHandleRegisterAgent_RefusesCrossAppRegistration closes the same shape
// of hole on the write side of the same endpoint pair.
func TestHandleRegisterAgent_RefusesCrossAppRegistration(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	myApp, otherApp := id.NewAppID(), id.NewAppID()
	ctx := adminCtx(id.NewUserID(), myApp, id.OrgID{})

	w := doRequest(ctx, t, mux, http.MethodPost, "/v1/admin/agents",
		map[string]any{"app_id": otherApp.String(), "client_id": "client_crossapp", "name": "N"})

	assert.Equal(t, http.StatusForbidden, w.Code,
		"a caller must not be able to register an agent under another application")
}

// TestHandleSetAgentStatus_RefusesCrossAppTarget covers the PATCH side: the
// target agent must belong to the caller's own app before any status change
// is applied to it.
func TestHandleSetAgentStatus_RefusesCrossAppTarget(t *testing.T) {
	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	otherApp := id.NewAppID()
	victim, err := p.RegisterAgent(context.Background(), &agentauth.Agent{
		AppID: otherApp, ClientID: "client_victim", Name: "Victim",
	})
	require.NoError(t, err)

	eng := &stubEngine{store: memory.New(), registry: plugin.NewRegistry(log.NewNoopLogger()), bus: hook.NewBus(log.NewNoopLogger()), logger: log.NewNoopLogger()}
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	myApp := id.NewAppID()
	ctx := adminCtx(id.NewUserID(), myApp, id.OrgID{})

	w := doRequest(ctx, t, mux, http.MethodPatch, "/v1/admin/agents/"+victim.ID.String()+"/status",
		map[string]any{"status": "blocked"})
	assert.Equal(t, http.StatusNotFound, w.Code,
		"an admin in a different app must not be able to change another app's agent status")

	got, err := agentStore.GetAgent(context.Background(), victim.ID)
	require.NoError(t, err)
	assert.Equal(t, agentauth.StatusApproved, got.Status, "the cross-app attempt must not have taken effect")
}

// ──────────────────────────────────────────────────
// ITEM 1 (fix round 2) — the caller's own org must be a FLOOR, not an
// optional filter a request can opt out of by leaving org_id out. Before
// this fix, callerOrgOrReject treated an omitted org_id as "no org scope"
// and returned the zero org regardless of whether the caller actually had
// one — which MemoryStore.ListAgents and RevokeGrantsByAgent both read as
// "app-wide": an org-scoped admin could enumerate every org's agents, or
// revoke every org's grants on an agent, just by leaving org_id out.
// ──────────────────────────────────────────────────

// listedAgentIDs decodes a GET /v1/admin/agents response into just the ids,
// which is all these tests need to assert against.
func listedAgentIDs(t *testing.T, w *httptest.ResponseRecorder) []string {
	t.Helper()
	var resp struct {
		Agents []struct {
			ID string `json:"id"`
		} `json:"agents"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	ids := make([]string, len(resp.Agents))
	for i, a := range resp.Agents {
		ids[i] = a.ID
	}
	return ids
}

func TestHandleListAgents_OrgScopedAdminOmittingOrgIDSeesOnlyOwnOrg(t *testing.T) {
	agentStore, mux := newRoutesTestPlugin(t)
	appID, myOrg, otherOrg := id.NewAppID(), id.NewOrgID(), id.NewOrgID()
	mine := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, OrgID: myOrg, ClientID: "client_org_floor_mine",
		Name: "Mine", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	theirs := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, OrgID: otherOrg, ClientID: "client_org_floor_theirs",
		Name: "Theirs", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	require.NoError(t, agentStore.CreateAgent(context.Background(), mine))
	require.NoError(t, agentStore.CreateAgent(context.Background(), theirs))

	// Org-scoped admin (context carries myOrg), org_id left out of the query.
	ctx := adminCtx(id.NewUserID(), appID, myOrg)
	w := doRequest(ctx, t, mux, http.MethodGet, "/v1/admin/agents", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	ids := listedAgentIDs(t, w)
	assert.Contains(t, ids, mine.ID.String(), "the caller's own org's agent must still be visible")
	assert.NotContains(t, ids, theirs.ID.String(),
		"omitting org_id must not widen an org-scoped admin's reach to another org in the same app")
}

func TestHandleListAgents_AdminWithNoOrgContextSeesAppWide(t *testing.T) {
	agentStore, mux := newRoutesTestPlugin(t)
	appID, orgA, orgB := id.NewAppID(), id.NewOrgID(), id.NewOrgID()
	a1 := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, OrgID: orgA, ClientID: "client_appwide_list_a",
		Name: "A", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	a2 := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, OrgID: orgB, ClientID: "client_appwide_list_b",
		Name: "B", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	require.NoError(t, agentStore.CreateAgent(context.Background(), a1))
	require.NoError(t, agentStore.CreateAgent(context.Background(), a2))

	// A genuinely app-scoped caller: no org in context at all, not merely an
	// omitted org_id on an org-scoped session.
	ctx := adminCtx(id.NewUserID(), appID, id.OrgID{})
	w := doRequest(ctx, t, mux, http.MethodGet, "/v1/admin/agents", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	ids := listedAgentIDs(t, w)
	assert.Contains(t, ids, a1.ID.String(), "a caller with no org context at all must still reach the app-wide listing")
	assert.Contains(t, ids, a2.ID.String())
}

func TestHandleSetAgentStatus_OrgScopedAdminOmittingOrgIDOnlyRevokesOwnOrgsGrants(t *testing.T) {
	agentStore, mux := newRoutesTestPlugin(t)
	appID, myOrg, otherOrg := id.NewAppID(), id.NewOrgID(), id.NewOrgID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, ClientID: "client_org_floor_status",
		Name: "Shared Agent", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	require.NoError(t, agentStore.CreateAgent(context.Background(), agent))
	inMyOrg := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: myOrg, Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	inOtherOrg := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: otherOrg, Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), inMyOrg))
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), inOtherOrg))

	// Org-scoped admin, org_id left out of the body.
	ctx := adminCtx(id.NewUserID(), appID, myOrg)
	w := doRequest(ctx, t, mux, http.MethodPatch, "/v1/admin/agents/"+agent.ID.String()+"/status",
		map[string]any{"status": "blocked"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	mine, err := agentStore.GetAgentGrant(context.Background(), inMyOrg.ID)
	require.NoError(t, err)
	assert.NotNil(t, mine.RevokedAt, "the caller's own org's grant on this agent must be revoked")

	other, err := agentStore.GetAgentGrant(context.Background(), inOtherOrg.ID)
	require.NoError(t, err)
	assert.Nil(t, other.RevokedAt,
		"omitting org_id must not let an org-scoped admin revoke another org's grants on the same agent — this is the destructive half of the same hole")
}

func TestHandleSetAgentStatus_AdminWithNoOrgContextRevokesAppWide(t *testing.T) {
	agentStore, mux := newRoutesTestPlugin(t)
	appID, orgA, orgB := id.NewAppID(), id.NewOrgID(), id.NewOrgID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, ClientID: "client_appwide_status",
		Name: "Shared Agent", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}
	require.NoError(t, agentStore.CreateAgent(context.Background(), agent))
	inOrgA := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: orgA, Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	inOrgB := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: orgB, Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), inOrgA))
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), inOrgB))

	// A genuinely app-scoped caller: no org in context at all.
	ctx := adminCtx(id.NewUserID(), appID, id.OrgID{})
	w := doRequest(ctx, t, mux, http.MethodPatch, "/v1/admin/agents/"+agent.ID.String()+"/status",
		map[string]any{"status": "blocked"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	a, err := agentStore.GetAgentGrant(context.Background(), inOrgA.ID)
	require.NoError(t, err)
	assert.NotNil(t, a.RevokedAt, "a caller with no org context at all must still be able to perform the app-wide block")
	b, err := agentStore.GetAgentGrant(context.Background(), inOrgB.ID)
	require.NoError(t, err)
	assert.NotNil(t, b.RevokedAt)
}

// TestHandleListAgents_OrgScopedAdminCannotNameAnotherOrgEither is the write
// side of the same helper: even with an explicit org_id in the request, an
// org-scoped admin still cannot escape their own org by naming a different
// one outright (this half was already covered by the original C3 fix; kept
// here as a companion to the omitted-org_id cases above so the full
// behavior of callerOrgOrReject's floor is documented in one place).
func TestHandleListAgents_OrgScopedAdminCannotNameAnotherOrgEither(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	appID, myOrg, otherOrg := id.NewAppID(), id.NewOrgID(), id.NewOrgID()
	ctx := adminCtx(id.NewUserID(), appID, myOrg)

	w := doRequest(ctx, t, mux, http.MethodGet, "/v1/admin/agents?org_id="+otherOrg.String(), nil)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"an org-scoped admin must not be able to name a different org's id either")
}

// ──────────────────────────────────────────────────
// I1 — the admin handlers must fail closed on their own, since SessionGuard
// and AdminGuard both no-op when the engine has no auth registry / RBAC
// (exactly what stubEngine models here, matching a real deployment missing
// those container entries — extension/extension.go wires the registry
// inside a deferred recover that swallows a missing entry silently).
// ──────────────────────────────────────────────────

func TestAdminHandlers_RejectUnauthenticated(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	orgID := id.NewOrgID()

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"list agents", http.MethodGet, "/v1/admin/agents", nil},
		{"register agent", http.MethodPost, "/v1/admin/agents", map[string]any{"client_id": "c", "name": "n"}},
		{"set agent status", http.MethodPatch, "/v1/admin/agents/" + id.NewAgentID().String() + "/status", map[string]any{"status": "blocked"}},
		{"put org policy", http.MethodPut, "/v1/admin/agents/policy", map[string]any{"org_id": orgID.String(), "mode": "open"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(context.Background(), t, mux, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"an anonymous request must be refused by the handler itself, since middleware alone cannot be relied on")
		})
	}
}

func TestMeHandlers_RejectUnauthenticated(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)

	w := doRequest(context.Background(), t, mux, http.MethodGet, "/v1/me/agents", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = doRequest(context.Background(), t, mux, http.MethodDelete, "/v1/me/agents/"+id.NewAgentGrantID().String(), nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ──────────────────────────────────────────────────
// I3 — bulk revocation must sweep the sessions it revokes, exactly like
// RevokeGrant already does for a single grant.
// ──────────────────────────────────────────────────

// TestSetAgentStatus_BlockingKillsLiveSessions guard-verifies I3's fix on
// the admin block path: without RevokeGrantsByAgent's returned ids being
// swept through DeleteSessionsByGrant, a live agent session survives
// blocking for up to its remaining TTL, authenticating as the delegating
// human on any route agentauth.Authorize does not guard.
func TestSetAgentStatus_BlockingKillsLiveSessions(t *testing.T) {
	p, grant, eng := issuanceSetup(t, &recordingHooks{})
	// SetAgentStatus loads the agent record itself (to stamp Status/UpdatedAt
	// on it), which issuanceSetup's grant does not come with — it wires only
	// the grant, not a backing Agent row.
	require.NoError(t, p.Store().CreateAgent(context.Background(), &agentauth.Agent{
		ID: grant.AgentID, AppID: grant.AppID, OrgID: grant.OrgID, ClientID: "client_block_sessions",
		Name: "Blockable Agent", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
	}))
	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err)
	_, err = eng.store.GetSession(context.Background(), sess.ID)
	require.NoError(t, err, "the session must exist before the agent is blocked")

	require.NoError(t, p.SetAgentStatus(context.Background(), grant.AgentID, grant.OrgID, agentauth.StatusBlocked))

	_, err = eng.store.GetSession(context.Background(), sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound,
		"blocking an agent must kill the live sessions its grants issued")
}

// TestOnBeforeMemberRemove_KillsLiveSessions is the same property for org
// departure: leaving an org revokes the member's grants in that org, and
// must also kill the sessions those grants already issued.
func TestOnBeforeMemberRemove_KillsLiveSessions(t *testing.T) {
	p, grant, eng := issuanceSetup(t, &recordingHooks{})
	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err)
	_, err = eng.store.GetSession(context.Background(), sess.ID)
	require.NoError(t, err, "the session must exist before the member leaves")

	require.NoError(t, p.OnBeforeMemberRemove(context.Background(), &organization.Member{
		ID: id.NewMemberID(), OrgID: grant.OrgID, UserID: grant.UserID,
	}))

	_, err = eng.store.GetSession(context.Background(), sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound,
		"a member leaving an org must kill the live sessions their grants already issued")
}

// ──────────────────────────────────────────────────
// M4 — a negative max_grant_ttl_seconds must be rejected, not silently
// turned into "no ceiling" by clampTTL's non-positive-means-unset reading.
// ──────────────────────────────────────────────────

func TestHandlePutOrgPolicy_RejectsNegativeMaxGrantTTL(t *testing.T) {
	_, mux := newRoutesTestPlugin(t)
	orgID := id.NewOrgID()
	ctx := adminCtx(id.NewUserID(), id.NewAppID(), orgID)

	w := doRequest(ctx, t, mux, http.MethodPut, "/v1/admin/agents/policy",
		map[string]any{"org_id": orgID.String(), "mode": "open", "max_grant_ttl_seconds": -1})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
