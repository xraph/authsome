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
		return nil, fmt.Errorf("agentauth: list grants: %w", err)
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
		return fmt.Errorf("agentauth: load grant: %w", err)
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
// a blocked agent kept working until its sessions aged out.
func (p *Plugin) SetAgentStatus(ctx context.Context, agentID id.AgentID, orgID id.OrgID, status AgentStatus) error {
	a, err := p.store.GetAgent(ctx, agentID)
	if errors.Is(err, ErrNotFound) {
		return forge.NotFound("agent not found")
	}
	if err != nil {
		return fmt.Errorf("agentauth: load agent: %w", err)
	}

	a.Status = status
	a.UpdatedAt = time.Now()
	if err := p.store.UpdateAgent(ctx, a); err != nil {
		return fmt.Errorf("agentauth: update agent: %w", err)
	}

	if status != StatusBlocked {
		return nil
	}
	if err := p.store.RevokeGrantsByAgent(ctx, agentID, orgID); err != nil {
		return fmt.Errorf("agentauth: revoke agent grants: %w", err)
	}
	// The cache is keyed by grant id and the revoked grants are not enumerated
	// here, so clear it wholesale. Blocking an agent is a rare admin action,
	// so the cost of a cold cache does not matter.
	p.cache.clear()
	return nil
}

// RegisterAgent creates a new org-registered agent. An org admin registering
// an agent has already decided to trust it, so it starts approved, unlike a
// self-registered agent (created elsewhere, outside this task's scope) which
// would start pending review.
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
	return resp, ctx.JSON(http.StatusOK, resp)
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
	return resp, ctx.JSON(http.StatusOK, resp)
}

// ListAgentsRequest binds the query parameters for GET /admin/agents.
type ListAgentsRequest struct {
	AppID string `query:"app_id" description:"Application identifier"`
	OrgID string `query:"org_id,omitempty" description:"Organization identifier"`
}

// ListAgentsResponse is the GET /admin/agents payload.
type ListAgentsResponse struct {
	Agents []*Agent `json:"agents"`
}

func (p *Plugin) handleListAgents(ctx forge.Context, req *ListAgentsRequest) (*ListAgentsResponse, error) {
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}
	orgID := id.Nil
	if req.OrgID != "" {
		orgID, err = id.ParseOrgID(req.OrgID)
		if err != nil {
			return nil, forge.BadRequest("invalid org_id")
		}
	}
	agents, err := p.store.ListAgents(ctx.Context(), appID, orgID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: list agents: %w", err))
	}
	if agents == nil {
		agents = []*Agent{}
	}
	resp := &ListAgentsResponse{Agents: agents}
	return resp, ctx.JSON(http.StatusOK, resp)
}

// RegisterAgentRequest binds the body for POST /admin/agents.
type RegisterAgentRequest struct {
	AppID       string `json:"app_id" description:"Application identifier"`
	OrgID       string `json:"org_id,omitempty" description:"Organization identifier"`
	ClientID    string `json:"client_id" description:"The oauth2provider client id this agent authenticates as"`
	Name        string `json:"name" description:"Human-readable agent name"`
	Description string `json:"description,omitempty"`
	LogoURI     string `json:"logo_uri,omitempty"`
}

func (p *Plugin) handleRegisterAgent(ctx forge.Context, req *RegisterAgentRequest) (*Agent, error) {
	userID, _ := middleware.UserIDFrom(ctx.Context())
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}
	orgID := id.Nil
	if req.OrgID != "" {
		orgID, err = id.ParseOrgID(req.OrgID)
		if err != nil {
			return nil, forge.BadRequest("invalid org_id")
		}
	}
	a, err := p.RegisterAgent(ctx.Context(), &Agent{
		AppID: appID, OrgID: orgID, ClientID: req.ClientID,
		Name: req.Name, Description: req.Description, LogoURI: req.LogoURI,
		CreatedBy: userID,
	})
	if err != nil {
		return nil, err
	}
	return a, ctx.JSON(http.StatusCreated, a)
}

// SetAgentStatusRequest binds the path and body for PATCH /admin/agents/:id/status.
type SetAgentStatusRequest struct {
	AgentID string `path:"id" description:"Agent identifier"`
	OrgID   string `json:"org_id,omitempty" description:"Organization whose grants on this agent are revoked when blocking"`
	Status  string `json:"status" description:"pending, approved or blocked"`
}

func (p *Plugin) handleSetAgentStatus(ctx forge.Context, req *SetAgentStatusRequest) (*StatusResponse, error) {
	agentID, err := id.ParseAgentID(req.AgentID)
	if err != nil {
		return nil, forge.BadRequest("invalid agent id")
	}
	switch AgentStatus(req.Status) {
	case StatusPending, StatusApproved, StatusBlocked:
	default:
		return nil, forge.BadRequest(fmt.Sprintf("invalid status %q", req.Status))
	}
	orgID := id.Nil
	if req.OrgID != "" {
		orgID, err = id.ParseOrgID(req.OrgID)
		if err != nil {
			return nil, forge.BadRequest("invalid org_id")
		}
	}
	if err := p.SetAgentStatus(ctx.Context(), agentID, orgID, AgentStatus(req.Status)); err != nil {
		return nil, err
	}
	resp := &StatusResponse{Status: "updated"}
	return resp, ctx.JSON(http.StatusOK, resp)
}

// PutOrgPolicyRequest binds the body for PUT /admin/agents/policy.
type PutOrgPolicyRequest struct {
	OrgID             string   `json:"org_id" description:"Organization identifier"`
	Mode              string   `json:"mode" description:"open, allowlist or blocked"`
	MaxGrantTTLSecond int64    `json:"max_grant_ttl_seconds,omitempty"`
	AllowedScopes     []string `json:"allowed_scopes,omitempty"`
}

func (p *Plugin) handlePutOrgPolicy(ctx forge.Context, req *PutOrgPolicyRequest) (*OrgAgentPolicy, error) {
	orgID, err := id.ParseOrgID(req.OrgID)
	if err != nil {
		return nil, forge.BadRequest("invalid org_id")
	}
	mode := PolicyMode(req.Mode)
	switch mode {
	case ModeOpen, ModeAllowlist, ModeBlocked:
	default:
		return nil, forge.BadRequest(fmt.Sprintf("invalid policy mode %q", req.Mode))
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
	return policy, ctx.JSON(http.StatusOK, policy)
}
