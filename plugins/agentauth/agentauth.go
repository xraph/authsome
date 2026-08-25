// Package agentauth adds delegated agent identity to authsome. An agent is a
// non-human principal that always acts on behalf of a human, which is what
// separates it from a serviceaccount.ServiceAccount.
package agentauth

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when an agent or grant cannot be found.
var ErrNotFound = errors.New("agentauth: not found")

// AgentOrigin records how an agent came to exist.
type AgentOrigin string

const (
	OriginSelfRegistered AgentOrigin = "self_registered"
	OriginOrgRegistered  AgentOrigin = "org_registered"
	OriginFirstParty     AgentOrigin = "first_party"
)

// AgentStatus is an agent's approval state.
type AgentStatus string

const (
	StatusPending  AgentStatus = "pending"
	StatusApproved AgentStatus = "approved"
	StatusBlocked  AgentStatus = "blocked"
)

// PolicyMode is an org's stance on agent delegation.
type PolicyMode string

const (
	ModeOpen      PolicyMode = "open"
	ModeAllowlist PolicyMode = "allowlist"
	ModeBlocked   PolicyMode = "blocked"
)

// Agent is a non-human principal that always acts for a human. It is keyed to
// an oauth2provider client by ClientID rather than embedded in it, so this
// package never migrates that plugin's schema.
type Agent struct {
	ID          id.AgentID  `json:"id"`
	AppID       id.AppID    `json:"app_id"`
	OrgID       id.OrgID    `json:"org_id,omitempty"`
	ClientID    string      `json:"client_id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	LogoURI     string      `json:"logo_uri,omitempty"`
	Origin      AgentOrigin `json:"origin"`
	Status      AgentStatus `json:"status"`
	CreatedBy   id.UserID   `json:"created_by,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// AgentGrant is one user's delegation to one agent. UserID is never the zero
// value and ExpiresAt is a value rather than a pointer, so both invariants
// this package depends on are carried by the type.
type AgentGrant struct {
	ID         id.AgentGrantID `json:"id"`
	AppID      id.AppID        `json:"app_id"`
	AgentID    id.AgentID      `json:"agent_id"`
	UserID     id.UserID       `json:"user_id"`
	OrgID      id.OrgID        `json:"org_id,omitempty"`
	Scopes     []string        `json:"scopes"`
	ConsentID  id.ConsentID    `json:"consent_id,omitempty"`
	ExpiresAt  time.Time       `json:"expires_at"`
	LastUsedAt *time.Time      `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// IsActive reports whether the grant may authorize a request at now.
func (g *AgentGrant) IsActive(now time.Time) bool {
	if g.RevokedAt != nil {
		return false
	}
	return now.Before(g.ExpiresAt)
}

// OrgAgentPolicy is an org's gate on agent delegation.
type OrgAgentPolicy struct {
	OrgID         id.OrgID      `json:"org_id"`
	Mode          PolicyMode    `json:"mode"`
	MaxGrantTTL   time.Duration `json:"max_grant_ttl"`
	AllowedScopes []string      `json:"allowed_scopes,omitempty"`
}

// Store persists agents, grants and org policy.
type Store interface {
	CreateAgent(ctx context.Context, a *Agent) error
	GetAgent(ctx context.Context, agentID id.AgentID) (*Agent, error)
	GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error)
	UpdateAgent(ctx context.Context, a *Agent) error
	ListAgents(ctx context.Context, appID id.AppID, orgID id.OrgID) ([]*Agent, error)

	CreateAgentGrant(ctx context.Context, g *AgentGrant) error
	GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error)
	GetActiveGrant(ctx context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error)
	ListGrantsByUser(ctx context.Context, userID id.UserID) ([]*AgentGrant, error)
	UpdateAgentGrant(ctx context.Context, g *AgentGrant) error
	RevokeAgentGrant(ctx context.Context, grantID id.AgentGrantID) error
	RevokeGrantsByUser(ctx context.Context, userID id.UserID) error
	RevokeGrantsByUserOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) error
	RevokeGrantsByOrg(ctx context.Context, orgID id.OrgID) error
	RevokeGrantsByAgent(ctx context.Context, agentID id.AgentID, orgID id.OrgID) error

	GetOrgPolicy(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error)
	PutOrgPolicy(ctx context.Context, p *OrgAgentPolicy) error
}
