package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SocialAccount links a user to an OAuth provider account
type SocialAccount struct {
	bun.BaseModel `bun:"table:social_accounts"`

	ID        xid.ID    `bun:",pk"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	// User relationship
	UserID             xid.ID  `bun:",notnull"`
	User               *User   `bun:"rel:belongs-to,join:user_id=id"`
	AppID              xid.ID  `bun:"app_id,notnull"`                        // Platform app (required)
	UserOrganizationID *xid.ID `bun:"user_organization_id,type:varchar(20)"` // User-created org (optional)

	// Provider information
	Provider   string `bun:",notnull"` // google, github, microsoft, etc.
	ProviderID string `bun:",notnull"` // Provider's unique user ID
	Email      string `bun:""`         // Email from provider (may differ from user.Email)
	Name       string `bun:""`         // Display name from provider
	Avatar     string `bun:""`         // Profile picture URL

	// OAuth tokens
	AccessToken      string     `bun:",notnull"`                  // Current access token
	RefreshToken     string     `bun:""`                          // Refresh token (if provided)
	TokenType        string     `bun:",notnull,default:'Bearer'"` // Token type
	ExpiresAt        *time.Time `bun:""`                          // Access token expiration
	RefreshExpiresAt *time.Time `bun:""`                          // Refresh token expiration
	Scope            string     `bun:""`                          // Granted scopes (comma-separated)

	// ID Token (for OIDC providers)
	IDToken string `bun:"type:text"` // Full ID token JWT

	// Provider-specific data (JSON)
	RawUserInfo string `bun:"type:jsonb"` // Raw user profile from provider

	// Account status
	Revoked   bool       `bun:",notnull,default:false"` // Whether tokens were revoked
	RevokedAt *time.Time `bun:""`                       // When account was disconnected
}

// IsTokenExpired checks if the access token has expired
func (sa *SocialAccount) IsTokenExpired() bool {
	if sa.ExpiresAt == nil {
		return false // No expiration
	}
	return time.Now().After(*sa.ExpiresAt)
}

// IsRefreshTokenValid checks if refresh token is valid
func (sa *SocialAccount) IsRefreshTokenValid() bool {
	if sa.RefreshToken == "" || sa.Revoked {
		return false
	}
	if sa.RefreshExpiresAt != nil {
		return time.Now().Before(*sa.RefreshExpiresAt)
	}
	return true // No expiration set
}

// NeedsRefresh checks if the access token needs refreshing
func (sa *SocialAccount) NeedsRefresh() bool {
	return sa.IsTokenExpired() && sa.IsRefreshTokenValid()
}
