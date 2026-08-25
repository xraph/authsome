package agentauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// GrantView is one delegation as the granting user sees it.
type GrantView struct {
	ID         string     `json:"id"`
	AgentID    string     `json:"agent_id"`
	AgentName  string     `json:"agent_name"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// ListGrantsResponse is the /v1/me/agents payload.
type ListGrantsResponse struct {
	Grants []GrantView `json:"grants"`
}

// StatusResponse is a generic status response for actions with no body worth
// naming, matching the shape used elsewhere across the plugin surfaces.
type StatusResponse struct {
	Status string `json:"status"`
}

// ListMyGrants returns every active delegation the user has issued.
func (p *Plugin) ListMyGrants(ctx context.Context, userID id.UserID) (*ListGrantsResponse, error) {
	grants, err := p.store.ListGrantsByUser(ctx, userID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: list grants: %w", err))
	}
	now := time.Now()
	out := &ListGrantsResponse{Grants: []GrantView{}}
	for _, g := range grants {
		if !g.IsActive(now) {
			continue
		}
		view := GrantView{
			ID: g.ID.String(), AgentID: g.AgentID.String(),
			Scopes: g.Scopes, ExpiresAt: g.ExpiresAt, LastUsedAt: g.LastUsedAt,
		}
		if a, aerr := p.store.GetAgent(ctx, g.AgentID); aerr == nil {
			view.AgentName = a.Name
		}
		out.Grants = append(out.Grants, view)
	}
	return out, nil
}

// RevokeMyGrant revokes one of the caller's own delegations. It refuses a
// grant belonging to anybody else, and returns the same response for that
// case as for a grant that does not exist at all, so a grant id is never an
// authorization: the endpoint cannot be used to probe whether a given grant
// id is real.
func (p *Plugin) RevokeMyGrant(ctx context.Context, userID id.UserID, grantID id.AgentGrantID) error {
	g, err := p.store.GetAgentGrant(ctx, grantID)
	if errors.Is(err, ErrNotFound) {
		return forge.NotFound("grant not found")
	}
	if err != nil {
		return forge.InternalError(fmt.Errorf("agentauth: load grant: %w", err))
	}
	if g.UserID.String() != userID.String() {
		// Same response as a missing grant, so the endpoint does not confirm
		// that somebody else's grant id exists.
		return forge.NotFound("grant not found")
	}
	return p.RevokeGrant(ctx, grantID)
}

// SetAgentStatus changes an agent's approval state. Blocking also revokes
// every grant the agent holds in that org, since leaving them live would mean
// a blocked agent kept working until its sessions aged out, and deletes the
// sessions those grants issued: without that, a live agent session (which
// carries the delegating human's UserID) keeps authenticating as that human
// on any route not guarded by agentauth.Authorize, for up to its remaining
// TTL, even though the agent that session belongs to has just been blocked.
func (p *Plugin) SetAgentStatus(ctx context.Context, agentID id.AgentID, orgID id.OrgID, status AgentStatus) error {
	a, err := p.store.GetAgent(ctx, agentID)
	if errors.Is(err, ErrNotFound) {
		return forge.NotFound("agent not found")
	}
	if err != nil {
		return forge.InternalError(fmt.Errorf("agentauth: load agent: %w", err))
	}

	a.Status = status
	a.UpdatedAt = time.Now()
	if updateErr := p.store.UpdateAgent(ctx, a); updateErr != nil {
		return forge.InternalError(fmt.Errorf("agentauth: update agent: %w", updateErr))
	}

	if status != StatusBlocked {
		return nil
	}
	revoked, err := p.store.RevokeGrantsByAgent(ctx, agentID, orgID)
	if err != nil {
		return forge.InternalError(fmt.Errorf("agentauth: revoke agent grants: %w", err))
	}
	// The cache is keyed by grant id and the revoked grants are not enumerated
	// here, so clear it wholesale. Blocking an agent is a rare admin action,
	// so the cost of a cold cache does not matter.
	p.cache.clear()
	return p.sweepSessions(ctx, revoked)
}

// RegisterAgent creates a new org-registered agent. An org admin registering
// an agent has already decided to trust it, so it starts approved, unlike a
// self-registered agent (created elsewhere, outside this task's scope) which
// would start pending review.
//
// ClientID must be unique: Evaluate resolves an agent by ClientID through
// GetAgentByClientID, and a store that let two agents share one would make
// that resolution nondeterministic — whichever record a range-based lookup
// happens to visit first decides whether a blocked agent's client is treated
// as blocked. CreateAgent enforces the invariant; this maps its ErrConflict
// onto an HTTP 409 rather than the generic 500 forge.InternalError would give
// every other store failure.
func (p *Plugin) RegisterAgent(ctx context.Context, in *Agent) (*Agent, error) {
	if in.ClientID == "" || in.Name == "" || in.AppID.IsNil() {
		return nil, forge.BadRequest("app_id, client_id and name are required")
	}
	now := time.Now()
	a := &Agent{
		ID:          id.NewAgentID(),
		AppID:       in.AppID,
		OrgID:       in.OrgID,
		ClientID:    in.ClientID,
		Name:        in.Name,
		Description: in.Description,
		LogoURI:     in.LogoURI,
		Origin:      OriginOrgRegistered,
		Status:      StatusApproved,
		CreatedBy:   in.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := p.store.CreateAgent(ctx, a); err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, forge.NewHTTPError(http.StatusConflict, "an agent is already registered for this client_id")
		}
		return nil, forge.InternalError(fmt.Errorf("agentauth: create agent: %w", err))
	}
	return a, nil
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleListMyGrants(ctx forge.Context, _ *apitypes.Empty) (*ListGrantsResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	resp, err := p.ListMyGrants(ctx.Context(), userID)
	if err != nil {
		return nil, err
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// RevokeMyGrantRequest binds the path parameter for DELETE /me/agents/:id.
type RevokeMyGrantRequest struct {
	GrantID string `path:"id" description:"Agent grant identifier"`
}

func (p *Plugin) handleRevokeMyGrant(ctx forge.Context, req *RevokeMyGrantRequest) (*StatusResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	grantID, err := id.ParseAgentGrantID(req.GrantID)
	if err != nil {
		return nil, forge.BadRequest("invalid grant id")
	}
	if err := p.RevokeMyGrant(ctx.Context(), userID, grantID); err != nil {
		return nil, err
	}
	resp := &StatusResponse{Status: "revoked"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// callerAppID resolves the authenticated caller's app from context. The
// admin agent routes are tenant-scoped, but HasPermission("write", "agent")
// carries no app or org dimension of its own (rbac/warden_store.go never
// reads a scoping id off the permission check) — so without this, an app_id
// or org_id taken straight from the request body or query would let any
// caller who holds the "agent" permission act against an arbitrary tenant,
// not just their own.
func callerAppID(ctx context.Context) (id.AppID, error) {
	appID, ok := middleware.AppIDFrom(ctx)
	if !ok || appID.IsNil() {
		return id.AppID{}, forge.Forbidden("an application context is required")
	}
	return appID, nil
}

// requestAppMatchesCaller validates an optional app_id field on a request
// against the caller's own app, resolved from context. An empty field is not
// a mismatch — the caller's app from context is authoritative either way,
// this only catches a request that names a *different* app outright.
func requestAppMatchesCaller(ctx context.Context, reqAppID string) error {
	if reqAppID == "" {
		return nil
	}
	appID, err := callerAppID(ctx)
	if err != nil {
		return err
	}
	parsed, perr := id.ParseAppID(reqAppID)
	if perr != nil {
		return forge.BadRequest("invalid app_id")
	}
	if parsed.String() != appID.String() {
		return forge.Forbidden("app_id does not match the authenticated application")
	}
	return nil
}

// callerOrgOrReject resolves the org an admin request may act within. The
// caller's own org, when the caller has one, is a FLOOR — not an optional
// filter a request can opt out of. Treating an omitted org_id as "no org
// scope" let an org-scoped admin holding read/write on agent enumerate every
// org's agents (ListAgents skips the org filter on a nil org) or revoke
// every org's grants on an agent (RevokeGrantsByAgent does the same) just by
// leaving org_id out of the request — the same class of cross-tenant hole
// requestAppMatchesCaller closes for app, one notch smaller: same-app
// cross-org rather than cross-app.
//
// So: when the caller has a real org in context, that org is always what is
// used, regardless of whether reqOrgID was supplied — an empty reqOrgID no
// longer means "go wider", it means "use my own org". A non-empty reqOrgID
// still must agree with it. Only a caller with no org context at all (a
// genuinely app-scoped session) can reach the app-wide zero org, and only
// when it doesn't try to name one.
func callerOrgOrReject(ctx context.Context, reqOrgID string) (id.OrgID, error) {
	callerOrgID, hasOrg := middleware.OrgIDFrom(ctx)
	hasOrg = hasOrg && !callerOrgID.IsNil()

	if hasOrg {
		if reqOrgID != "" {
			parsed, err := id.ParseOrgID(reqOrgID)
			if err != nil {
				return id.Nil, forge.BadRequest("invalid org_id")
			}
			if parsed.String() != callerOrgID.String() {
				return id.Nil, forge.Forbidden("org_id does not match the authenticated organization")
			}
		}
		return callerOrgID, nil
	}

	if reqOrgID == "" {
		return id.Nil, nil
	}
	return id.Nil, forge.Forbidden("org_id does not match the authenticated organization")
}

// requiredCallerOrgID is callerOrgOrReject for an endpoint where an org is
// not optional — a delegation policy always belongs to exactly one org, so
// there is no app-global reading of a request that omits or disagrees with
// the caller's own org.
func requiredCallerOrgID(ctx context.Context, reqOrgID string) (id.OrgID, error) {
	callerOrgID, hasOrg := middleware.OrgIDFrom(ctx)
	if !hasOrg || callerOrgID.IsNil() {
		return id.Nil, forge.Forbidden("an organization context is required to set delegation policy")
	}
	parsed, err := id.ParseOrgID(reqOrgID)
	if err != nil {
		return id.Nil, forge.BadRequest("invalid org_id")
	}
	if parsed.String() != callerOrgID.String() {
		return id.Nil, forge.Forbidden("cannot set delegation policy for another organization")
	}
	return parsed, nil
}

// ListAgentsRequest binds the query parameters for GET /admin/agents. Both
// fields are optional and, when present, must agree with the caller's own
// app/org from context — see requestAppMatchesCaller and callerOrgOrReject.
// Without that check this endpoint let any admin caller enumerate another
// app's agents just by naming its app_id in the query.
type ListAgentsRequest struct {
	AppID string `query:"app_id,omitempty" description:"Application identifier; must match the caller's own app"`
	OrgID string `query:"org_id,omitempty" description:"Organization identifier; must match the caller's own org"`
}

// ListAgentsResponse is the GET /admin/agents payload.
type ListAgentsResponse struct {
	Agents []*Agent `json:"agents"`
}

func (p *Plugin) handleListAgents(ctx forge.Context, req *ListAgentsRequest) (*ListAgentsResponse, error) {
	if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if err := requestAppMatchesCaller(ctx.Context(), req.AppID); err != nil {
		return nil, err
	}
	appID, err := callerAppID(ctx.Context())
	if err != nil {
		return nil, err
	}
	orgID, err := callerOrgOrReject(ctx.Context(), req.OrgID)
	if err != nil {
		return nil, err
	}
	agents, err := p.store.ListAgents(ctx.Context(), appID, orgID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: list agents: %w", err))
	}
	if agents == nil {
		agents = []*Agent{}
	}
	resp := &ListAgentsResponse{Agents: agents}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// RegisterAgentRequest binds the body for POST /admin/agents. AppID and OrgID
// are optional and, when present, must agree with the caller's own app/org
// from context (see requestAppMatchesCaller and callerOrgOrReject) — the
// agent is always registered under the caller's actual app regardless.
type RegisterAgentRequest struct {
	AppID       string `json:"app_id,omitempty" description:"Application identifier; must match the caller's own app"`
	OrgID       string `json:"org_id,omitempty" description:"Organization identifier; must match the caller's own org"`
	ClientID    string `json:"client_id" description:"The oauth2provider client id this agent authenticates as"`
	Name        string `json:"name" description:"Human-readable agent name"`
	Description string `json:"description,omitempty"`
	LogoURI     string `json:"logo_uri,omitempty"`
}

func (p *Plugin) handleRegisterAgent(ctx forge.Context, req *RegisterAgentRequest) (*Agent, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if err := requestAppMatchesCaller(ctx.Context(), req.AppID); err != nil {
		return nil, err
	}
	appID, err := callerAppID(ctx.Context())
	if err != nil {
		return nil, err
	}
	orgID, err := callerOrgOrReject(ctx.Context(), req.OrgID)
	if err != nil {
		return nil, err
	}
	a, err := p.RegisterAgent(ctx.Context(), &Agent{
		AppID: appID, OrgID: orgID, ClientID: req.ClientID,
		Name: req.Name, Description: req.Description, LogoURI: req.LogoURI,
		CreatedBy: userID,
	})
	if err != nil {
		return nil, err
	}
	return nil, ctx.JSON(http.StatusCreated, a)
}

// SetAgentStatusRequest binds the path and body for PATCH /admin/agents/:id/status.
type SetAgentStatusRequest struct {
	AgentID string `path:"id" description:"Agent identifier"`
	OrgID   string `json:"org_id,omitempty" description:"Organization whose grants on this agent are revoked when blocking; must match the caller's own org"`
	Status  string `json:"status" description:"pending, approved or blocked"`
}

func (p *Plugin) handleSetAgentStatus(ctx forge.Context, req *SetAgentStatusRequest) (*StatusResponse, error) {
	if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	agentID, err := id.ParseAgentID(req.AgentID)
	if err != nil {
		return nil, forge.BadRequest("invalid agent id")
	}
	switch AgentStatus(req.Status) {
	case StatusPending, StatusApproved, StatusBlocked:
	default:
		return nil, forge.BadRequest(fmt.Sprintf("invalid status %q", req.Status))
	}
	appID, err := callerAppID(ctx.Context())
	if err != nil {
		return nil, err
	}
	// The target agent must belong to the caller's own app. Without this, an
	// admin in app A holding "write agent" could block (or approve) an agent
	// belonging to app B just by guessing or enumerating its agent id.
	agent, err := p.store.GetAgent(ctx.Context(), agentID)
	if errors.Is(err, ErrNotFound) {
		return nil, forge.NotFound("agent not found")
	}
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: load agent: %w", err))
	}
	if agent.AppID.String() != appID.String() {
		// Same response as a missing agent: a cross-tenant admin caller must
		// not be able to tell "exists in another app" apart from "doesn't
		// exist" by probing this endpoint.
		return nil, forge.NotFound("agent not found")
	}
	orgID, err := callerOrgOrReject(ctx.Context(), req.OrgID)
	if err != nil {
		return nil, err
	}
	if err := p.SetAgentStatus(ctx.Context(), agentID, orgID, AgentStatus(req.Status)); err != nil {
		return nil, err
	}
	resp := &StatusResponse{Status: "updated"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// PutOrgPolicyRequest binds the body for PUT /admin/agents/policy. OrgID is
// required and must match the caller's own org from context — see
// requiredCallerOrgID. Without that check, any admin caller holding
// "write agent" in any org could flip a different org's delegation policy
// wide open, since HasPermission carries no org dimension of its own.
type PutOrgPolicyRequest struct {
	OrgID             string   `json:"org_id" description:"Organization identifier; must match the caller's own org"`
	Mode              string   `json:"mode" description:"open, allowlist or blocked"`
	MaxGrantTTLSecond int64    `json:"max_grant_ttl_seconds,omitempty" description:"Must not be negative"`
	AllowedScopes     []string `json:"allowed_scopes,omitempty"`
}

func (p *Plugin) handlePutOrgPolicy(ctx forge.Context, req *PutOrgPolicyRequest) (*OrgAgentPolicy, error) {
	if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	orgID, err := requiredCallerOrgID(ctx.Context(), req.OrgID)
	if err != nil {
		return nil, err
	}
	mode := PolicyMode(req.Mode)
	switch mode {
	case ModeOpen, ModeAllowlist, ModeBlocked:
	default:
		return nil, forge.BadRequest(fmt.Sprintf("invalid policy mode %q", req.Mode))
	}
	if req.MaxGrantTTLSecond < 0 {
		// clampTTL (grant.go) treats a non-positive MaxGrantTTL as "no
		// ceiling" rather than "no time at all", so a negative value here
		// would silently loosen the org's cap instead of tightening it —
		// exactly backwards from what an operator setting it would expect.
		return nil, forge.BadRequest("max_grant_ttl_seconds must not be negative")
	}
	policy := &OrgAgentPolicy{
		OrgID:         orgID,
		Mode:          mode,
		MaxGrantTTL:   time.Duration(req.MaxGrantTTLSecond) * time.Second,
		AllowedScopes: req.AllowedScopes,
	}
	if err := p.store.PutOrgPolicy(ctx.Context(), policy); err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: put org policy: %w", err))
	}
	return nil, ctx.JSON(http.StatusOK, policy)
}
