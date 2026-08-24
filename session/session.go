// Package session defines the session domain entity and its store interface.
package session

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Principal kinds a session may carry. An empty PrincipalKind means
// PrincipalKindUser, for rows written before the column existed.
const (
	PrincipalKindUser           = "user"
	PrincipalKindServiceAccount = "service_account"
	// PrincipalKindAgent marks a session issued to a delegated agent. Unlike
	// a service-account session, UserID stays populated with the delegating
	// human, so every consumer that resolves a session's user keeps working.
	PrincipalKindAgent = "agent"
)

// Session represents an authenticated user session.
type Session struct {
	ID     id.SessionID     `json:"id"`
	AppID  id.AppID         `json:"app_id"`
	EnvID  id.EnvironmentID `json:"env_id"`
	UserID id.UserID        `json:"user_id"`
	OrgID  id.OrgID         `json:"org_id,omitempty"`
	// FamilyID groups sessions descended from a single sign-in via
	// successive refresh-token rotations. All sessions sharing a FamilyID
	// can be revoked together when refresh-token replay is detected
	// (RFC 6819 §5.2.2.3). Fresh sign-ins start a new family; rotations
	// inherit the parent's FamilyID. Zero-value on legacy rows is allowed.
	FamilyID              id.SessionFamilyID `json:"family_id,omitempty"`
	Token                 string             `json:"-"`
	RefreshToken          string             `json:"-"`
	IPAddress             string             `json:"ip_address,omitempty"`
	UserAgent             string             `json:"user_agent,omitempty"`
	DeviceID              id.DeviceID        `json:"device_id,omitempty"`
	ImpersonatedBy        id.UserID          `json:"impersonated_by,omitempty"`
	LastActivityAt        time.Time          `json:"last_activity_at,omitempty"`
	ExpiresAt             time.Time          `json:"expires_at"`
	RefreshTokenExpiresAt time.Time          `json:"refresh_token_expires_at"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`

	// Roles holds the role slugs the principal held when this session was
	// issued, stamped once on the way to the store rather than resolved per
	// request. Forge's auth extension reads them off the AuthContext to
	// satisfy the role requirements a route declares, and the generated
	// clients turn the same strings into their capability surface.
	//
	// Stamped means stale: a role granted or revoked after sign-in does not
	// reach an existing session, so a revocation needs the session revoked
	// too. That is the trade this design accepts in exchange for keeping
	// authentication free of an RBAC lookup on every request. Anything that
	// must be current at the instant it is checked belongs in a permission
	// check against warden, not here.
	Roles []string `json:"roles,omitempty"`

	// PrincipalKind identifies the type of principal that owns this session.
	// Valid values are "user" and "service_account". Empty string means "user"
	// for backwards compatibility with existing sessions.
	PrincipalKind string `json:"principal_kind,omitempty"`
	// ServiceAccountID is set when PrincipalKind is "service_account".
	// UserID is left as the zero value in that case.
	ServiceAccountID id.ServiceAccountID `json:"service_account_id,omitempty"`
	// AgentID is set when PrincipalKind is "agent". UserID remains the
	// delegating human.
	AgentID id.AgentID `json:"agent_id,omitempty"`
	// GrantID names the AgentGrant that authorized this session, so revoking
	// that grant can find and delete the sessions it issued.
	GrantID id.AgentGrantID `json:"grant_id,omitempty"`
}
