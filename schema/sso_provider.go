package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"time"
)

// SSOProvider stores per-organization SSO provider configuration
type SSOProvider struct {
	bun.BaseModel `bun:"table:sso_providers"`

	ID        xid.ID `bun:",pk"`
	CreatedAt time.Time
	UpdatedAt time.Time

	OrganizationID xid.ID // org scoped; optional for standalone
	ProviderID     string // e.g., "okta-saml" or "google-oidc"
	Type           string // "saml" or "oidc"
	Domain         string // org domain match

	// SAML config (basic fields; extend as needed)
	SAMLEntryPoint string
	SAMLIssuer     string
	SAMLCert       string

	// OIDC config (basic fields; extend as needed)
	OIDCClientID     string
	OIDCClientSecret string
	OIDCIssuer       string
	OIDCRedirectURI  string
}
