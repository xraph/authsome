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

	// App context
	AppID xid.ID `bun:"app_id,notnull,type:varchar(20)" json:"appID"`

	// OAuth2/OIDC fields
	Code                string     `bun:"code,unique,notnull" json:"code"`                            // The authorization code
	ClientID            string     `bun:"client_id,notnull" json:"clientID"`                          // OAuth client ID
	UserID              xid.ID     `bun:"user_id,notnull,type:varchar(20)" json:"userID"`             // User who authorized
	RedirectURI         string     `bun:"redirect_uri,notnull" json:"redirectURI"`                    // Redirect URI used in auth request
	Scope               string     `bun:"scope,notnull" json:"scope"`                                 // Requested scopes
	State               string     `bun:"state" json:"state,omitempty"`                               // State parameter from auth request
	Nonce               string     `bun:"nonce" json:"nonce,omitempty"`                               // Nonce for OIDC
	CodeChallenge       string     `bun:"code_challenge" json:"codeChallenge,omitempty"`              // PKCE code challenge
	CodeChallengeMethod string     `bun:"code_challenge_method" json:"codeChallengeMethod,omitempty"` // PKCE challenge method (S256, plain)
	ExpiresAt           time.Time  `bun:"expires_at,notnull" json:"expiresAt"`                        // Code expiration (typically 10 minutes)
	Used                bool       `bun:"used,notnull,default:false" json:"used"`                     // Whether code has been exchanged
	UsedAt              *time.Time `bun:"used_at" json:"usedAt,omitempty"`                            // When code was used

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// IsExpired checks if the authorization code has expired
func (ac *AuthorizationCode) IsExpired() bool {
	return time.Now().After(ac.ExpiresAt)
}

// IsValid checks if the authorization code is valid (not expired and not used)
func (ac *AuthorizationCode) IsValid() bool {
	return !ac.Used && !ac.IsExpired()
}
