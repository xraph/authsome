package principal

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when a principal or delegation does not exist.
var ErrNotFound = errors.New("principal: not found")

// GrantKind distinguishes the two ways one principal comes to act for another.
type GrantKind string

const (
	// GrantDelegation is an agent or workload acting for a user who granted
	// it that authority. Both parties are checked, so the decision narrows.
	GrantDelegation GrantKind = "delegation"
	// GrantImpersonation is an admin acting as a user. The actor is not
	// independently checked, because impersonating somebody is precisely the
	// request to evaluate as them. The gate sits on the Impersonate call.
	GrantImpersonation GrantKind = "impersonation"
)

// Delegation records that Actor may act on behalf of Subject.
//
// It is the durable, revocable, auditable half of the actor chain. The chain
// on a session says who is acting. The grant says they were allowed to.
type Delegation struct {
	ID    id.DelegationID `json:"id"`
	AppID id.AppID        `json:"app_id"`
	OrgID id.OrgID        `json:"org_id,omitempty"`

	Actor   Ref `json:"actor"`
	Subject Ref `json:"subject"`

	GrantKind GrantKind `json:"grant_kind"`
	// Scopes narrows what the actor may do while acting. An empty list places
	// no restriction of its own, leaving the actor's own scopes as the only
	// limit.
	Scopes []string `json:"scopes,omitempty"`
	// GrantedBy is who consented. For a delegation that is normally the
	// subject; for an impersonation it is the admin who initiated it.
	GrantedBy Ref `json:"granted_by"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// IsActive reports whether d may be exercised at time at.
func (d *Delegation) IsActive(at time.Time) bool {
	if d.RevokedAt != nil {
		return false
	}
	return d.ExpiresAt == nil || !at.After(*d.ExpiresAt)
}

// AllowsScope reports whether scope falls inside d's scope filter.
func (d *Delegation) AllowsScope(scope string) bool {
	if len(d.Scopes) == 0 {
		return true
	}
	for _, s := range d.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// DelegationQuery filters a delegation listing.
type DelegationQuery struct {
	AppID      id.AppID
	OrgID      id.OrgID
	Actor      *Ref
	Subject    *Ref
	GrantKind  GrantKind
	ActiveOnly bool
	ActiveAsOf time.Time
	Limit      int
}

// Query filters a principal listing.
type Query struct {
	AppID      id.AppID
	Kind       Kind
	OwnerUser  *id.UserID
	Parent     *Ref
	ActiveOnly bool
	ActiveAsOf time.Time
	Limit      int
}

// Store is the persistence interface for principals and their delegations.
//
// The principal read side reads the same rows the serviceaccount store
// writes. It is separate because callers resolving a caller mid-request want a
// Principal and not a ServiceAccount, and because a user principal is
// assembled from a different table entirely.
type Store interface {
	// GetPrincipal resolves any principal by ref. Users are assembled from
	// the user table; every other kind comes from the service accounts table.
	GetPrincipal(ctx context.Context, ref Ref) (*Principal, error)
	// ListPrincipals returns principals matching q.
	ListPrincipals(ctx context.Context, q *Query) ([]*Principal, error)

	// CreateDelegation stores a new grant.
	CreateDelegation(ctx context.Context, d *Delegation) error
	// GetDelegation returns a grant by ID.
	GetDelegation(ctx context.Context, delID id.DelegationID) (*Delegation, error)
	// FindActiveDelegation returns the live grant letting actor act for
	// subject under grantKind, or ErrNotFound.
	FindActiveDelegation(ctx context.Context, appID id.AppID, actor, subject Ref, grantKind GrantKind) (*Delegation, error)
	// ListDelegations returns grants matching q.
	ListDelegations(ctx context.Context, q *DelegationQuery) ([]*Delegation, error)
	// RevokeDelegation marks a grant revoked at the given time. Revoking an
	// already-revoked grant is not an error.
	RevokeDelegation(ctx context.Context, delID id.DelegationID, at time.Time) error
}
