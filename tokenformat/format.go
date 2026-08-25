// Package tokenformat provides pluggable token generation strategies.
// Access tokens can be opaque (hex) or JWT; refresh tokens are always opaque.
package tokenformat

import (
	"errors"
	"time"
)

// Sentinel errors.
var (
	ErrInvalidToken  = errors.New("tokenformat: invalid token")
	ErrTokenExpired  = errors.New("tokenformat: token expired")
	ErrUnsignedToken = errors.New("tokenformat: unsigned or tampered token")
)

// ActClaim is the RFC 8693 `act` claim: the party acting on behalf of the
// subject. It nests, so a chain of delegations is a chain of ActClaims with
// the immediate actor outermost. Subject carries a principal.Ref in its
// "kind:id" string form.
type ActClaim struct {
	Subject string    `json:"sub"`
	Act     *ActClaim `json:"act,omitempty"`
}

// TokenClaims carries the identity payload embedded in an access token.
type TokenClaims struct {
	UserID    string   `json:"sub"`
	AppID     string   `json:"app_id"`
	EnvID     string   `json:"env_id,omitempty"`
	OrgID     string   `json:"org_id,omitempty"`
	SessionID string   `json:"sid"`
	Scopes    []string `json:"scopes,omitempty"`
	// Audience holds the resource identifiers this token was granted for
	// (RFC 8707), emitted as the `aud` claim. Empty means unrestricted.
	//
	// This overrides JWTConfig.Audience when set. The config value is a single
	// static audience for a whole app, which predates resource indicators and
	// stays as the fallback so existing deployments are unaffected.
	Audience []string `json:"aud,omitempty"`
	// DPoPJKT is the RFC 7638 thumbprint this token is bound to (RFC 9449).
	// Empty means an unbound bearer token. Serialised as the cnf.jkt claim.
	DPoPJKT   string `json:"-"`
	IssuedAt  time.Time
	ExpiresAt time.Time

	// Act is present for a delegated token and nil otherwise. Impersonation
	// emits no act claim at all (RFC 8693 section 1.1), which is why the full
	// record lives on the session row rather than in the token.
	Act *ActClaim `json:"act,omitempty"`

	// PrincipalKind names the kind of caller this token belongs to, using the
	// values the principal package defines: "user", "agent", "workload" or
	// "service_account". Empty means user, so tokens minted before this field
	// existed keep validating.
	//
	// It is a string and not principal.Kind because tokenformat sits below
	// principal in the import graph, and a claims struct is a wire format that
	// should not drag a domain package in behind it.
	PrincipalKind string `json:"pk,omitempty"`

	// PrincipalID is the caller's id, duplicating what sub carries. It is here
	// so a consumer can read the caller without first deciding whether sub
	// means a user. That ambiguity is what made the JWT auth path parse an
	// empty sub and panic on a machine token.
	PrincipalID string `json:"pid,omitempty"`
}

// Format generates and validates access tokens. Refresh tokens are always
// opaque (must be revocable via the store) so they are not part of this
// interface.
type Format interface {
	// Name returns the format identifier ("opaque" or "jwt").
	Name() string

	// GenerateAccessToken produces a token string from claims.
	GenerateAccessToken(claims TokenClaims) (string, error)

	// ValidateAccessToken parses and validates a token, returning the
	// embedded claims. Returns ErrInvalidToken or ErrTokenExpired on
	// failure.
	ValidateAccessToken(token string) (*TokenClaims, error)
}
