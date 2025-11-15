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

	ID    xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID xid.ID `bun:"app_id,notnull,type:varchar(20)" json:"appID"`

	// Token fields
	AccessToken      string     `bun:"access_token,unique,notnull" json:"-"`          // The access token
	RefreshToken     string     `bun:"refresh_token,unique" json:"-"`                 // Optional refresh token
	TokenType        string     `bun:"token_type,notnull,default:'Bearer'" json:"tokenType"` // Token type (Bearer)
	ClientID         string     `bun:"client_id,notnull" json:"clientID"`             // OAuth client ID
	UserID           xid.ID     `bun:"user_id,notnull,type:varchar(20)" json:"userID"` // User who owns the token
	Scope            string     `bun:"scope,notnull" json:"scope"`                    // Granted scopes
	ExpiresAt        time.Time  `bun:"expires_at,notnull" json:"expiresAt"`           // Token expiration
	RefreshExpiresAt *time.Time `bun:"refresh_expires_at" json:"refreshExpiresAt,omitempty"` // Refresh token expiration
	Revoked          bool       `bun:"revoked,notnull,default:false" json:"revoked"`  // Whether token is revoked
	RevokedAt        *time.Time `bun:"revoked_at" json:"revokedAt,omitempty"`         // When token was revoked

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// IsExpired checks if the access token has expired
func (ot *OAuthToken) IsExpired() bool {
	return time.Now().After(ot.ExpiresAt)
}

// IsValid checks if the access token is valid (not expired and not revoked)
func (ot *OAuthToken) IsValid() bool {
	return !ot.Revoked && !ot.IsExpired()
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
