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
		// A genuine store failure denies. It must never collapse into the
		// not-found branch above and allow the request through.
		return forge.InternalError(fmt.Errorf("agentauth: load agent: %w", err))
	}

	if agent.Status == StatusBlocked {
		return forge.Forbidden("agent is blocked")
	}

	for _, org := range governingOrgs(agent, orgID) {
		policy, err := p.policyFor(ctx, org)
		if err != nil {
			return err
		}
		if err := p.checkPolicy(agent, policy, scopes); err != nil {
			return err
		}
	}
	return nil
}

// CreateGrant writes the delegation. It refuses a grant with no delegating
// human, which is the invariant the whole authorization model rests on. It
// also re-runs the same policy check Evaluate performs, once per governing
// org, rather than trusting that every caller ran Evaluate first: CreateGrant
// is exported, so nothing at the type level stops a caller from reaching it
// directly, and the invariant that a scope or a blocked policy never reaches
// a stored grant has to hold regardless of call order.
func (p *Plugin) CreateGrant(ctx context.Context, in CreateGrantInput) (*AgentGrant, error) {
	// IsNil rules out the zero value; the prefix check rules out a non-zero
	// ID that merely isn't a user id. id.UserID is a type alias, so nothing
	// at compile time stops an org or agent id from being passed here.
	if in.UserID.IsNil() || in.UserID.Prefix() != id.PrefixUser {
		return nil, errors.New("agentauth: a grant requires a delegating user")
	}

	agent, err := p.store.GetAgent(ctx, in.AgentID)
	if errors.Is(err, ErrNotFound) {
		return nil, forge.BadRequest("unknown agent")
	}
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: load agent: %w", err))
	}
	if agent.Status == StatusBlocked {
		return nil, forge.Forbidden("agent is blocked")
	}

	orgs := governingOrgs(agent, in.OrgID)
	ttl := in.RequestedTTL
	for _, org := range orgs {
		policy, err := p.policyFor(ctx, org)
		if err != nil {
			return nil, err
		}
		if err := p.checkPolicy(agent, policy, in.Scopes); err != nil {
			return nil, err
		}
		// Folding the running ttl back in as the "requested" value narrows it
		// by each governing org's ceiling in turn, so the org with the
		// tightest MaxGrantTTL always wins regardless of how many orgs apply.
		ttl = p.clampTTL(policy, ttl)
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
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.store.CreateAgentGrant(ctx, g); err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: create grant: %w", err))
	}
	return g, nil
}

// checkPolicy applies one org's policy against the agent and the requested
// scopes. Evaluate and CreateGrant both call it, once per governing org, so
// neither can write or authorize a delegation the other would have refused.
func (p *Plugin) checkPolicy(agent *Agent, policy *OrgAgentPolicy, scopes []string) error {
	switch policy.Mode {
	case ModeBlocked:
		return forge.Forbidden("this organization does not allow agent delegation")
	case ModeAllowlist:
		if agent.Status != StatusApproved {
			return forge.Forbidden("agent is not approved for this organization")
		}
	case ModeOpen:
		// Any non-blocked agent may be authorized.
	default:
		// An unrecognized mode denies. A policy nobody can interpret must not
		// be treated as permission: PolicyMode is a bare string whose zero
		// value is "", and nothing upstream of this switch can guarantee the
		// value is one of the three known constants (a partial update that
		// only touches MaxGrantTTL and re-Puts the struct is enough to carry
		// a garbled Mode). Falling open here would let exactly that kind of
		// write silently disarm an org's block.
		return forge.Forbidden("agent delegation policy for this organization is not recognized")
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

// clampTTL takes the shortest of the request, the org ceiling and the plugin
// default. A zero request means "use the default". If every input folds to a
// non-positive duration — for example WithDefaultGrantTTL(0) with no org
// ceiling to save it — it falls back to the package default rather than
// producing a grant that is born expired.
func (p *Plugin) clampTTL(policy *OrgAgentPolicy, requested time.Duration) time.Duration {
	ttl := p.grantTTL
	if requested > 0 && requested < ttl {
		ttl = requested
	}
	if policy != nil && policy.MaxGrantTTL > 0 && policy.MaxGrantTTL < ttl {
		ttl = policy.MaxGrantTTL
	}
	if ttl <= 0 {
		ttl = defaultGrantTTL
	}
	return ttl
}

// governingOrgs returns the distinct, non-nil organizations whose policy
// governs a delegation: the org that registered the agent, and the org the
// consenting session is scoped to, if either is present. Both apply when
// both are present and different, and the delegation must clear every one of
// them — either org's policy is a well-formed "no" regardless of what the
// other says.
//
// This is deliberately not "pick a winner". Preferring only the agent's org
// closes the dodge where an app-scoped session (zero session org) authorizes
// an agent whose own org is blocked, but it reopens a worse hole: an org that
// blocks delegation would only be able to enforce that block against agents
// it registered itself, never against an agent registered elsewhere and used
// by one of its own members. Preferring only the session org reopens the
// original hole the same way. Checking both is the only option that lets
// each org's policy mean what it says regardless of which side of the
// delegation an org is standing on.
//
// When neither source yields an org, there is no organization to have an
// opinion — the single-tenant and app-scoped case — and the sole element
// returned is the zero org id, which resolves to the open default.
func governingOrgs(agent *Agent, sessionOrg id.OrgID) []id.OrgID {
	var orgs []id.OrgID
	if !agent.OrgID.IsNil() {
		orgs = append(orgs, agent.OrgID)
	}
	if !sessionOrg.IsNil() && sessionOrg.String() != agent.OrgID.String() {
		orgs = append(orgs, sessionOrg)
	}
	if len(orgs) == 0 {
		orgs = append(orgs, id.OrgID{})
	}
	return orgs
}

// policyFor returns the org's policy, defaulting to open when no row exists.
func (p *Plugin) policyFor(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	policy, err := p.store.GetOrgPolicy(ctx, orgID)
	if errors.Is(err, ErrNotFound) {
		return &OrgAgentPolicy{OrgID: orgID, Mode: ModeOpen}, nil
	}
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("agentauth: load org policy: %w", err))
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
