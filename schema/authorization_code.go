package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// AuthorizationCode represents an OAuth2/OIDC authorization code
type AuthorizationCode struct {
	bun.BaseModel `bun:"table:authorization_codes"`

	ID        xid.ID    `bun:",pk"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	// OAuth2/OIDC fields
	Code         string    `bun:",unique,notnull"` // The authorization code
	ClientID     string    `bun:",notnull"`        // OAuth client ID
	UserID       xid.ID    `bun:",notnull"`        // User who authorized
	RedirectURI  string    `bun:",notnull"`        // Redirect URI used in auth request
	Scope        string    `bun:",notnull"`        // Requested scopes
	State        string    `bun:""`                // State parameter from auth request
	Nonce        string    `bun:""`                // Nonce for OIDC
	CodeChallenge string   `bun:""`                // PKCE code challenge
	CodeChallengeMethod string `bun:""`            // PKCE challenge method (S256, plain)
	ExpiresAt    time.Time `bun:",notnull"`        // Code expiration (typically 10 minutes)
	Used         bool      `bun:",notnull,default:false"` // Whether code has been exchanged
	UsedAt       *time.Time `bun:""`               // When code was used
}

// IsExpired checks if the authorization code has expired
func (ac *AuthorizationCode) IsExpired() bool {
	return time.Now().After(ac.ExpiresAt)
}

// IsValid checks if the authorization code is valid (not expired and not used)
func (ac *AuthorizationCode) IsValid() bool {
	return !ac.Used && !ac.IsExpired()
}