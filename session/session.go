// Package session defines the session domain entity and its store interface.
package session

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
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

	// Audience holds the resource identifiers this session's access token was
	// granted for (RFC 8707). Empty means unrestricted, which is what every
	// session issued before resource indicators existed carries, and what any
	// client that sends no `resource` parameter still gets.
	//
	// This is the opaque-token half of the `aud` claim. A JWT carries its
	// audience inside the token and is checked without a store read, while an
	// opaque token is only a session lookup key, so the same value has to live
	// here for introspection and for the middleware audience check to see it.
	Audience []string `json:"aud,omitempty"`

	// Scopes holds the OAuth scopes this session was issued with, stamped at
	// issuance. Same trade as Roles: authoritative for what this token may
	// do, stale with respect to anything granted afterwards.
	//
	// Empty does not mean "may do anything". A session minted by password
	// sign-in carries no scopes, and the RFC 8693 token exchange grant treats
	// that as "no subject-side ceiling" while still bounding the result by the
	// delegation grant and the client's own registered scopes.
	Scopes []string `json:"scopes,omitempty"`

	// PrincipalKind identifies the type of principal that owns this session:
	// principal.KindUser, principal.KindAgent, principal.KindWorkload, or
	// principal.KindService. Empty string means KindUser, for backwards
	// compatibility with sessions written before this field existed.
	PrincipalKind principal.Kind `json:"principal_kind,omitempty"`
	// ServiceAccountID is set when PrincipalKind is anything other than
	// KindUser, i.e. for the three non-human kinds, which all share the
	// service-account identity space. UserID is left as the zero value in
	// that case.
	ServiceAccountID id.ServiceAccountID `json:"service_account_id,omitempty"`

	// Actors is the chain of principals acting on the subject's behalf,
	// ordered nearest-caller-first. Empty on an ordinary session, where the
	// subject is calling directly.
	//
	// This subsumes the old ImpersonatedBy field. An impersonation is a chain
	// of one user actor with ActorGrant set to impersonation, which is why
	// ImpersonatedBy is now derived rather than stored.
	Actors principal.Chain `json:"actors,omitempty"`

	// ActorGrant records which sort of grant put the actors on this session.
	// Empty when Actors is empty.
	ActorGrant principal.GrantKind `json:"actor_grant,omitempty"`

	// DelegationID is the grant this session was minted against. Zero for an
	// ordinary session.
	DelegationID id.DelegationID `json:"delegation_id,omitempty"`
}

// Subject returns the principal this session is for.
//
// An empty PrincipalKind means "user", which is what every row written before
// the field existed carries. Normalizing it on read would make a legacy row
// indistinguishable from one deliberately stamped, so the zero value is
// interpreted here and not rewritten in the store.
func (s *Session) Subject() principal.Ref {
	switch s.PrincipalKind {
	case "", principal.KindUser:
		return principal.Ref{Kind: principal.KindUser, ID: s.UserID.String()}
	default:
		return principal.Ref{Kind: s.PrincipalKind, ID: s.ServiceAccountID.String()}
	}
}

// IsHumanPrincipal reports whether a person owns this session.
func (s *Session) IsHumanPrincipal() bool {
	return s.PrincipalKind == "" || s.PrincipalKind == principal.KindUser
}

// AuthzActors returns the actors Warden must independently authorize.
//
// Empty for an impersonation, and that is the point. Impersonating somebody is
// precisely the request to evaluate as them, so the admin's own permissions
// are not intersected in. The gate for impersonation sits on Engine.Impersonate,
// which is where an admin is checked once, rather than on every subsequent
// permission check made while impersonating.
func (s *Session) AuthzActors() principal.Chain {
	if s.ActorGrant == principal.GrantImpersonation {
		return nil
	}
	return s.Actors
}

// ImpersonatedBy returns the admin acting as this session's user, or the zero
// ID when nobody is.
//
// Derived from Actors rather than stored. The grant kind is checked first: an
// agent acting for a user is also a session with two principals, and reporting
// that as impersonation would put an admin-takeover banner and an admin-
// severity audit record on ordinary delegated traffic.
func (s *Session) ImpersonatedBy() id.UserID {
	if s.ActorGrant != principal.GrantImpersonation {
		return id.Nil
	}
	for i := len(s.Actors) - 1; i >= 0; i-- {
		if s.Actors[i].Kind != principal.KindUser {
			continue
		}
		uid, err := id.ParseUserID(s.Actors[i].ID)
		if err != nil {
			continue
		}
		return uid
	}
	return id.Nil
}

// SetImpersonatedBy marks this session as adminID acting as its user.
func (s *Session) SetImpersonatedBy(adminID id.UserID) {
	if adminID.IsNil() {
		return
	}
	s.Actors = principal.Chain{{Kind: principal.KindUser, ID: adminID.String()}}
	s.ActorGrant = principal.GrantImpersonation
}

// MarshalJSON keeps impersonated_by on the wire now that it is no longer a
// struct field. Consumers outside this repository read that key, and the
// chain is an addition for them, not a replacement.
func (s Session) MarshalJSON() ([]byte, error) {
	type alias Session
	out := struct {
		alias
		// No ,omitempty here on purpose. Before this change ImpersonatedBy
		// was an id.UserID, a struct, and encoding/json never omits a struct
		// for omitempty, so the key was always present, holding "" when
		// nobody was impersonating. Dropping the key on an unimpersonated
		// session would move the wire format, which the store models and
		// external consumers both depend on staying put.
		ImpersonatedBy string `json:"impersonated_by"`
	}{alias: alias(s)}
	if imp := s.ImpersonatedBy(); !imp.IsNil() {
		out.ImpersonatedBy = imp.String()
	}
	return json.Marshal(out)
}

// UnmarshalJSON accepts either representation: a payload carrying only the
// legacy impersonated_by key rebuilds the chain from it, and one carrying
// actors is taken as written.
func (s *Session) UnmarshalJSON(data []byte) error {
	type alias Session
	in := struct {
		*alias
		ImpersonatedBy string `json:"impersonated_by,omitempty"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if len(s.Actors) == 0 && in.ImpersonatedBy != "" {
		uid, err := id.ParseUserID(in.ImpersonatedBy)
		if err != nil {
			return fmt.Errorf("session: parse impersonated_by: %w", err)
		}
		s.SetImpersonatedBy(uid)
	}
	return nil
}
