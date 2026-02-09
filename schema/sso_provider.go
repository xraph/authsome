package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SSOProvider stores SSO provider configuration with multi-tenant scoping.
type SSOProvider struct {
	bun.BaseModel `bun:"table:sso_providers"`

	ID        xid.ID `bun:",pk"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// Multi-tenant scoping: App → Environment → Organization
	AppID          xid.ID  `bun:",notnull"`  // Platform tenant (required)
	EnvironmentID  xid.ID  `bun:",notnull"`  // Environment within app (required)
	OrganizationID *xid.ID `bun:",nullzero"` // End-user workspace (optional for app-level providers)

	ProviderID string `bun:",notnull"` // e.g., "okta-saml" or "google-oidc"
	Type       string `bun:",notnull"` // "saml" or "oidc"
	Domain     string // org domain match for auto-discovery

	// Attribute mapping from SSO assertions to user fields
	// Maps user field names to SSO attribute names
	// Example: {"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"}
	AttributeMapping map[string]string `bun:"type:jsonb"`

	// SAML configuration
	SAMLEntryPoint string // IdP SSO URL
	SAMLIssuer     string // IdP Entity ID
	SAMLCert       string // IdP signing certificate (PEM format)

	// OIDC configuration
	OIDCClientID     string // OAuth2 client ID
	OIDCClientSecret string // OAuth2 client secret
	OIDCIssuer       string // OIDC issuer URL (e.g., https://idp.example.com)
	OIDCRedirectURI  string // Callback URL for this provider
}
