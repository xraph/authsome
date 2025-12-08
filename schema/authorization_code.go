package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// AuthorizationCode represents an OAuth2/OIDC authorization code
type AuthorizationCode struct {
	AuditableModel
	bun.BaseModel `bun:"table:authorization_codes"`

	// Context fields
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appID"`
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // Optional org context
	SessionID      *xid.ID `bun:"session_id,type:varchar(20)" json:"sessionID,omitempty"`           // Link to user session

	// OAuth2/OIDC fields
	Code                string `bun:"code,unique,notnull" json:"code"`                            // The authorization code
	ClientID            string `bun:"client_id,notnull" json:"clientID"`                          // OAuth client ID
	UserID              xid.ID `bun:"user_id,notnull,type:varchar(20)" json:"userID"`             // User who authorized
	RedirectURI         string `bun:"redirect_uri,notnull" json:"redirectURI"`                    // Redirect URI used in auth request
	Scope               string `bun:"scope,notnull" json:"scope"`                                 // Requested scopes
	State               string `bun:"state" json:"state,omitempty"`                               // State parameter from auth request
	Nonce               string `bun:"nonce" json:"nonce,omitempty"`                               // Nonce for OIDC
	CodeChallenge       string `bun:"code_challenge" json:"codeChallenge,omitempty"`              // PKCE code challenge
	CodeChallengeMethod string `bun:"code_challenge_method" json:"codeChallengeMethod,omitempty"` // PKCE challenge method (S256, plain)

	// Consent tracking
	ConsentGranted bool   `bun:"consent_granted,default:false" json:"consentGranted"`
	ConsentScopes  string `bun:"consent_scopes" json:"consentScopes,omitempty"` // Scopes user consented to

	// Authentication context
	AuthTime time.Time `bun:"auth_time,notnull" json:"authTime"` // When user authenticated (for max_age checks)

	// Lifecycle
	ExpiresAt time.Time  `bun:"expires_at,notnull" json:"expiresAt"` // Code expiration (typically 10 minutes)
	Used      bool       `bun:"used,notnull,default:false" json:"used"`
	UsedAt    *time.Time `bun:"used_at" json:"usedAt,omitempty"`

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Session      *Session      `bun:"rel:belongs-to,join:session_id=id"`
}

// IsExpired checks if the authorization code has expired
func (ac *AuthorizationCode) IsExpired() bool {
	return time.Now().After(ac.ExpiresAt)
}

// IsValid checks if the authorization code is valid (not expired and not used)
func (ac *AuthorizationCode) IsValid() bool {
	return !ac.Used && !ac.IsExpired()
}
