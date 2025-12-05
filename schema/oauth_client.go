package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// OAuthClient stores registered OAuth/OIDC clients
type OAuthClient struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:oauth_clients,alias:oc"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appID"`
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // null = app-level
	Name           string  `bun:"name,notnull" json:"name"`
	ClientID       string  `bun:"client_id,notnull,unique" json:"clientID"`
	ClientSecret   string  `bun:"client_secret,notnull" json:"-"`

	// OAuth2/OIDC Configuration
	RedirectURI            string   `bun:"redirect_uri,notnull" json:"redirectURI"` // Legacy single URI, kept for backward compatibility
	RedirectURIs           []string `bun:"redirect_uris,array,type:text[]" json:"redirectURIs"`
	PostLogoutRedirectURIs []string `bun:"post_logout_redirect_uris,array,type:text[]" json:"postLogoutRedirectURIs,omitempty"`
	GrantTypes             []string `bun:"grant_types,array,type:text[]" json:"grantTypes"`             // Default: ["authorization_code", "refresh_token"] - set in application code
	ResponseTypes          []string `bun:"response_types,array,type:text[]" json:"responseTypes"`       // Default: ["code"] - set in application code
	AllowedScopes          []string `bun:"allowed_scopes,array,type:text[]" json:"allowedScopes,omitempty"`

	// Client Authentication & Security
	TokenEndpointAuthMethod string `bun:"token_endpoint_auth_method,default:'client_secret_basic'" json:"tokenEndpointAuthMethod"` // client_secret_basic, client_secret_post, none
	ApplicationType         string `bun:"application_type,default:'web'" json:"applicationType"`                                    // web, native, spa
	RequirePKCE             bool   `bun:"require_pkce,default:false" json:"requirePKCE"`
	RequireConsent          bool   `bun:"require_consent,default:true" json:"requireConsent"`
	TrustedClient           bool   `bun:"trusted_client,default:false" json:"trustedClient"`

	// Client Metadata (RFC 7591)
	LogoURI  string   `bun:"logo_uri" json:"logoURI,omitempty"`
	PolicyURI string  `bun:"policy_uri" json:"policyURI,omitempty"`
	TosURI    string  `bun:"tos_uri" json:"tosURI,omitempty"`
	Contacts  []string `bun:"contacts,array,type:text[]" json:"contacts,omitempty"`

	// Flexible metadata storage
	Metadata map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
}
