// Package sso provides enterprise Single Sign-On (SSO) authentication via
// SAML 2.0 and OpenID Connect (OIDC) protocols.
//
// Usage:
//
//	p := sso.New(sso.Config{
//	    Providers: []sso.Provider{
//	        sso.NewOIDCProvider(sso.OIDCConfig{
//	            Name:         "okta",
//	            Issuer:       "https://mycompany.okta.com",
//	            ClientID:     "client-id",
//	            ClientSecret: "client-secret",
//	            RedirectURL:  "https://app.com/api/auth/sso/okta/callback",
//	        }),
//	    },
//	})
package sso

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
)

// Provider is the interface for SSO identity providers.
type Provider interface {
	// Name returns the unique identifier for this provider (e.g. "okta", "azure-ad").
	Name() string

	// Protocol returns the SSO protocol ("oidc" or "saml").
	Protocol() string

	// LoginURL returns the URL to redirect the user to for authentication.
	LoginURL(state string) (string, error)

	// HandleCallback processes the callback from the identity provider
	// and returns the authenticated user's identity.
	HandleCallback(ctx context.Context, params map[string]string) (*User, error)
}

// User represents the identity returned by an SSO provider.
type User struct {
	ProviderUserID string            `json:"provider_user_id"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Groups         []string          `json:"groups,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	// EmailVerified is the provider's own statement about the address, when
	// it made one: OIDC's email_verified claim. Nil means the provider said
	// nothing; an explicit false refuses the login.
	EmailVerified *bool `json:"email_verified,omitempty"`
}

type Connection struct {
	ID       id.SSOConnectionID `json:"id"`
	AppID    id.AppID           `json:"app_id"`
	EnvID    string             `json:"env_id,omitempty"`
	OrgID    id.OrgID           `json:"org_id,omitempty"`
	Provider string             `json:"provider"`
	Protocol string             `json:"protocol"`
	Domain   string             `json:"domain"`
	// DisplayName is an optional admin-set label for the connection (e.g. "Okta",
	// "Google Workspace"), so multiple connections for the same domain can be told
	// apart. Cosmetic — never used for routing.
	DisplayName  string `json:"display_name,omitempty"`
	MetadataURL  string `json:"metadata_url,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"-"`
	Issuer       string `json:"issuer,omitempty"`
	Active       bool   `json:"active"`
	// Enforced requires users on this connection's domain to sign in via SSO —
	// password login is blocked for them (workspace owners/admins excepted). See
	// the plugin's OnBeforeSignIn.
	Enforced bool `json:"enforced"`

	// AllowedDomains lists email domains, besides Domain, that this
	// connection may assert. A login whose email is outside Domain and this
	// list is refused before any account is touched.
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	// TrustedFederation adopts the identity provider's subject as the local
	// user id when a user is first created through this connection. Off by
	// default: an IdP that can pick subjects can otherwise pick local ids.
	TrustedFederation bool `json:"trusted_federation,omitempty"`

	// SAML-specific configuration. Populated only for SAML connections.
	IDPMetadataXML    string            `json:"idp_metadata_xml,omitempty"`
	IDPSSOURL         string            `json:"idp_sso_url,omitempty"`
	IDPCertificate    string            `json:"idp_certificate,omitempty"`
	EntityID          string            `json:"entity_id,omitempty"`
	ACSURL            string            `json:"acs_url,omitempty"`
	SPCertificate     string            `json:"sp_certificate,omitempty"`
	SPPrivateKey      string            `json:"-"` // secret — never serialized
	SignRequests      bool              `json:"sign_requests,omitempty"`
	AttributeMappings map[string]string `json:"attribute_mappings,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store persists SSO connections.
type Store interface {
	CreateConnection(ctx context.Context, c *Connection) error
	GetConnection(ctx context.Context, connID id.SSOConnectionID) (*Connection, error)
	GetConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*Connection, error)
	// GetConnectionByDomainAndOrg resolves the active connection for a domain
	// within a specific org (workspace). Multi-tenant SSO allows the same domain
	// in several orgs, so lookups that must land on one workspace use this.
	GetConnectionByDomainAndOrg(ctx context.Context, appID id.AppID, orgID id.OrgID, domain string) (*Connection, error)
	GetConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*Connection, error)
	ListConnections(ctx context.Context, appID id.AppID) ([]*Connection, error)
	UpdateConnection(ctx context.Context, c *Connection) error
	DeleteConnection(ctx context.Context, connID id.SSOConnectionID) error

	// GetIdentity resolves the user bound to a provider subject on one
	// connection. CreateIdentity records the binding and fails on a subject
	// the connection already holds.
	GetIdentity(ctx context.Context, connID id.SSOConnectionID, subject string) (*Identity, error)
	CreateIdentity(ctx context.Context, ident *Identity) error
}
