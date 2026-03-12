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
	HandleCallback(ctx context.Context, params map[string]string) (*SSOUser, error)
}

// SSOUser represents the identity returned by an SSO provider.
type SSOUser struct {
	ProviderUserID string            `json:"provider_user_id"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Groups         []string          `json:"groups,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
}

// SSOConnection represents a stored SSO connection for a tenant.
type SSOConnection struct {
	ID           id.SSOConnectionID `json:"id"`
	AppID        id.AppID           `json:"app_id"`
	OrgID        id.OrgID           `json:"org_id,omitempty"`
	Provider     string             `json:"provider"`
	Protocol     string             `json:"protocol"`
	Domain       string             `json:"domain"`
	MetadataURL  string             `json:"metadata_url,omitempty"`
	ClientID     string             `json:"client_id,omitempty"`
	ClientSecret string             `json:"-"`
	Issuer       string             `json:"issuer,omitempty"`
	Active       bool               `json:"active"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// Store persists SSO connections.
type Store interface {
	CreateSSOConnection(ctx context.Context, c *SSOConnection) error
	GetSSOConnection(ctx context.Context, connID id.SSOConnectionID) (*SSOConnection, error)
	GetSSOConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*SSOConnection, error)
	GetSSOConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*SSOConnection, error)
	ListSSOConnections(ctx context.Context, appID id.AppID) ([]*SSOConnection, error)
	UpdateSSOConnection(ctx context.Context, c *SSOConnection) error
	DeleteSSOConnection(ctx context.Context, connID id.SSOConnectionID) error
}
