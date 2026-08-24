// Package principal defines the caller abstraction shared by human users,
// AI agents and workloads. It holds value types only: no store access, no
// HTTP, no engine. Everything above it in the import graph (session, store,
// middleware, plugin) can depend on this package without cycles.
package principal

import (
	"fmt"
	"strings"
	"time"

	"github.com/xraph/authsome/id"
)

// Kind is the sort of caller a principal is.
type Kind string

const (
	// KindUser is a human with an account.
	KindUser Kind = "user"
	// KindAgent is an AI agent or MCP client acting under a registration.
	KindAgent Kind = "agent"
	// KindWorkload is a machine caller with no human behind it: a CI job, a
	// cron, a service calling another service.
	KindWorkload Kind = "workload"
	// KindService is the kind sessions written before agents and workloads
	// existed carry. Retained so those rows keep resolving.
	KindService Kind = "service_account"
)

// IsHuman reports whether k denotes a person.
func (k Kind) IsHuman() bool { return k == KindUser }

// Valid reports whether k is a kind this package knows.
func (k Kind) Valid() bool {
	switch k {
	case KindUser, KindAgent, KindWorkload, KindService:
		return true
	default:
		return false
	}
}

// Ref is the addressable identity of a principal. It is comparable, so it can
// be a map key and can be compared with ==, and it is cheap enough to put on a
// context and to serialize into a token claim.
type Ref struct {
	Kind Kind   `json:"kind"`
	ID   string `json:"id"`
}

// String renders the ref as "kind:id".
func (r Ref) String() string {
	if r.IsZero() {
		return ""
	}
	return string(r.Kind) + ":" + r.ID
}

// IsZero reports whether r addresses nothing.
func (r Ref) IsZero() bool { return r.Kind == "" || r.ID == "" }

// ParseRef parses the "kind:id" form produced by Ref.String.
//
// It splits on the first colon only. A TypeID does not contain one today, but
// refs are compared to make authorization decisions, and splitting on the last
// colon would let an id with a colon in it address a different principal than
// the one that was written.
func ParseRef(s string) (Ref, error) {
	kindStr, idStr, found := strings.Cut(s, ":")
	if !found {
		return Ref{}, fmt.Errorf("principal: parse ref %q: missing kind separator", s)
	}
	if idStr == "" {
		return Ref{}, fmt.Errorf("principal: parse ref %q: empty id", s)
	}
	kind := Kind(kindStr)
	if !kind.Valid() {
		return Ref{}, fmt.Errorf("principal: parse ref %q: unknown kind %q", s, kindStr)
	}
	return Ref{Kind: kind, ID: idStr}, nil
}

// UserRef builds a ref for a human user.
func UserRef(userID id.UserID) Ref {
	return Ref{Kind: KindUser, ID: userID.String()}
}

// Principal is a resolved caller carrying everything an authorization decision
// needs, so callers do not go back to the store mid-decision.
type Principal struct {
	Ref

	AppID  id.AppID
	OrgID  id.OrgID
	EnvID  id.EnvironmentID
	Name   string
	Scopes []string
	Roles  []string

	// Owner is the principal answerable for this one. Nil for users, who are
	// answerable for themselves.
	Owner *Ref
	// Parent is the registered principal that minted this one. Set only on
	// ephemeral children.
	Parent *Ref
	// ExpiresAt is a hard cutoff. Nil means durable.
	ExpiresAt *time.Time
	Disabled  bool
}

// IsExpired reports whether p has passed its cutoff at time at.
func (p *Principal) IsExpired(at time.Time) bool {
	return p.ExpiresAt != nil && at.After(*p.ExpiresAt)
}

// IsActive reports whether p may authenticate at time at.
func (p *Principal) IsActive(at time.Time) bool {
	return !p.Disabled && !p.IsExpired(at)
}

// IsEphemeral reports whether p was minted by another principal.
func (p *Principal) IsEphemeral() bool { return p.Parent != nil }
