package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// OAuthToken represents an OAuth2/OIDC access token
type OAuthToken struct {
	bun.BaseModel `bun:"table:oauth_tokens"`

	ID        xid.ID    `bun:",pk"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	// Token fields
	AccessToken  string    `bun:",unique,notnull"` // The access token
	RefreshToken string    `bun:",unique"`         // Optional refresh token
	TokenType    string    `bun:",notnull,default:'Bearer'"` // Token type (Bearer)
	ClientID     string    `bun:",notnull"`        // OAuth client ID
	UserID       xid.ID    `bun:",notnull"`        // User who owns the token
	Scope        string    `bun:",notnull"`        // Granted scopes
	ExpiresAt    time.Time `bun:",notnull"`        // Token expiration
	RefreshExpiresAt *time.Time `bun:""`           // Refresh token expiration
	Revoked      bool      `bun:",notnull,default:false"` // Whether token is revoked
	RevokedAt    *time.Time `bun:""`              // When token was revoked
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