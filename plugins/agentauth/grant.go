package agentauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// Compile-time check: the plugin is the provider's consent gate.
var _ oauth2provider.ConsentGate = (*Plugin)(nil)

// CreateGrantInput describes a delegation a user is about to authorize.
type CreateGrantInput struct {
	AppID        id.AppID
	AgentID      id.AgentID
	UserID       id.UserID
	OrgID        id.OrgID
	Scopes       []string
	ConsentID    id.ConsentID
	RequestedTTL time.Duration
}

// Evaluate implements oauth2provider.ConsentGate. It runs at the moment a
// user consents, which is the only point where an org's stance on a given
// agent is a well-formed question. Registration is app-global.
func (p *Plugin) Evaluate(ctx context.Context, clientID string, _ id.UserID, orgID id.OrgID, scopes []string) error {
	agent, err := p.store.GetAgentByClientID(ctx, clientID)
	if errors.Is(err, ErrNotFound) {
		// Not an agent, just an ordinary OAuth client. Not this gate's business.
		return nil
	}
	if err != nil {
		return fmt.Errorf("agentauth: load agent: %w", err)
	}

	if agent.Status == StatusBlocked {
		return forge.Forbidden("agent is blocked")
	}

	policy, err := p.policyFor(ctx, effectiveOrg(agent, orgID))
	if err != nil {
		return err
	}

	switch policy.Mode {
	case ModeBlocked:
		return forge.Forbidden("this organization does not allow agent delegation")
	case ModeAllowlist:
		if agent.Status != StatusApproved {
			return forge.Forbidden("agent is not approved for this organization")
		}
	case ModeOpen:
		// Any non-blocked agent may be authorized.
	}

	for _, s := range scopes {
		if !p.scopes.Known(s) {
			return forge.BadRequest(fmt.Sprintf("unknown delegation scope %q", s))
		}
		if !scopeAllowed(policy.AllowedScopes, s) {
			return forge.Forbidden(fmt.Sprintf("scope %q is not permitted by organization policy", s))
		}
	}
	return nil
}

// CreateGrant writes the delegation. It refuses a grant with no delegating
// human, which is the invariant the whole authorization model rests on.
func (p *Plugin) CreateGrant(ctx context.Context, in CreateGrantInput) (*AgentGrant, error) {
	if in.UserID.IsNil() {
		return nil, errors.New("agentauth: a grant requires a delegating user")
	}

	policy, err := p.policyFor(ctx, in.OrgID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	g := &AgentGrant{
		ID:        id.NewAgentGrantID(),
		AppID:     in.AppID,
		AgentID:   in.AgentID,
		UserID:    in.UserID,
		OrgID:     in.OrgID,
		Scopes:    in.Scopes,
		ConsentID: in.ConsentID,
		ExpiresAt: now.Add(p.clampTTL(policy, in.RequestedTTL)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.store.CreateAgentGrant(ctx, g); err != nil {
		return nil, fmt.Errorf("agentauth: create grant: %w", err)
	}
	return g, nil
}

// clampTTL takes the shortest of the request, the org ceiling and the plugin
// default. A zero request means "use the default".
func (p *Plugin) clampTTL(policy *OrgAgentPolicy, requested time.Duration) time.Duration {
	ttl := p.grantTTL
	if requested > 0 && requested < ttl {
		ttl = requested
	}
	if policy != nil && policy.MaxGrantTTL > 0 && policy.MaxGrantTTL < ttl {
		ttl = policy.MaxGrantTTL
	}
	return ttl
}

// effectiveOrg decides which organization's policy governs this consent.
//
// The orgID the provider hands us comes from the caller's session, and the
// auth middleware populates it only when that session is org-scoped. An
// app-scoped session therefore arrives with the zero value, and keying policy
// off that would mean an org that set ModeBlocked never had it enforced
// against its own members signing in without an active org. So the agent's own
// record wins when it has one: an org-registered agent carries the org that
// registered it, which is a more trustworthy source than ambient session
// scope, and it cannot be dodged by dropping org context from the request.
//
// When neither source yields an org, no org policy applies. That is the
// single-tenant and app-scoped case, where there is no organization to have an
// opinion. A self-registered agent authorized from a session with no org
// context is the one combination this leaves ungoverned; see the plan's
// deferred notes.
func effectiveOrg(agent *Agent, sessionOrg id.OrgID) id.OrgID {
	if !agent.OrgID.IsNil() {
		return agent.OrgID
	}
	return sessionOrg
}

// policyFor returns the org's policy, defaulting to open when no row exists.
func (p *Plugin) policyFor(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	policy, err := p.store.GetOrgPolicy(ctx, orgID)
	if errors.Is(err, ErrNotFound) {
		return &OrgAgentPolicy{OrgID: orgID, Mode: ModeOpen}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agentauth: load org policy: %w", err)
	}
	return policy, nil
}

// scopeAllowed reports whether a scope sits inside the org's ceiling. An empty
// ceiling means no restriction beyond the registry.
func scopeAllowed(ceiling []string, scope string) bool {
	if len(ceiling) == 0 {
		return true
	}
	for _, s := range ceiling {
		if s == scope {
			return true
		}
	}
	return false
}
