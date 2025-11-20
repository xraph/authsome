package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// OAuthToken represents an OAuth2/OIDC access token
type OAuthToken struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:oauth_tokens,alias:ot"`

	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appID"`
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // Optional org context
	SessionID      *xid.ID `bun:"session_id,type:varchar(20)" json:"sessionID,omitempty"`           // Linked session for lifecycle management

	// Token fields
	AccessToken  string `bun:"access_token,unique,notnull" json:"-"`                 // The access token
	RefreshToken string `bun:"refresh_token,unique" json:"-"`                        // Optional refresh token
	IDToken      string `bun:"id_token" json:"-"`                                    // OIDC ID token
	TokenType    string `bun:"token_type,notnull,default:'Bearer'" json:"tokenType"` // Token type (Bearer)
	TokenClass   string `bun:"token_class,default:'access_token'" json:"tokenClass"` // access_token, refresh_token, id_token
	ClientID     string `bun:"client_id,notnull" json:"clientID"`                    // OAuth client ID
	UserID       xid.ID `bun:"user_id,notnull,type:varchar(20)" json:"userID"`       // User who owns the token
	Scope        string `bun:"scope,notnull" json:"scope"`                           // Granted scopes

	// JWT Claims
	JTI       string     `bun:"jti,unique" json:"jti,omitempty"`                      // JWT ID for token revocation by ID
	Issuer    string     `bun:"issuer" json:"issuer,omitempty"`                       // Token issuer
	Audience  []string   `bun:"audience,array,type:text[]" json:"audience,omitempty"` // Token audience (aud claim)
	NotBefore *time.Time `bun:"not_before" json:"notBefore,omitempty"`                // Token validity start time (nbf claim)

	// Authentication context
	AuthTime *time.Time `bun:"auth_time" json:"authTime,omitempty"`        // When user authenticated
	ACR      string     `bun:"acr" json:"acr,omitempty"`                   // Authentication context class reference
	AMR      []string   `bun:"amr,array,type:text[]" json:"amr,omitempty"` // Authentication methods references

	// Lifecycle
	ExpiresAt        time.Time  `bun:"expires_at,notnull" json:"expiresAt"`                  // Token expiration
	RefreshExpiresAt *time.Time `bun:"refresh_expires_at" json:"refreshExpiresAt,omitempty"` // Refresh token expiration
	Revoked          bool       `bun:"revoked,notnull,default:false" json:"revoked"`         // Whether token is revoked
	RevokedAt        *time.Time `bun:"revoked_at" json:"revokedAt,omitempty"`                // When token was revoked

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Session      *Session      `bun:"rel:belongs-to,join:session_id=id"`
}

// IsExpired checks if the access token has expired
func (ot *OAuthToken) IsExpired() bool {
	return time.Now().After(ot.ExpiresAt)
}

// IsValid checks if the access token is valid (not expired and not revoked)
func (ot *OAuthToken) IsValid() bool {
	if ot.Revoked {
		return false
	}
	if ot.IsExpired() {
		return false
	}
	// Check not before if set
	if ot.NotBefore != nil && time.Now().Before(*ot.NotBefore) {
		return false
	}
	return true
}

// IsRefreshValid checks if the refresh token is valid
func (ot *OAuthToken) IsRefreshValid() bool {
	if ot.RefreshToken == "" || ot.Revoked {
		return false
	}
	if ot.RefreshExpiresAt != nil {
		return time.Now().Before(*ot.RefreshExpiresAt)
	}
	return true
}
