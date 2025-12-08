package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Session represents the session table
type Session struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:sessions,alias:s"`

	ID    xid.ID `bun:"id,pk,type:varchar(20)"`
	Token string `bun:"token,notnull,unique"`

	// App-centric context
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)"`
	EnvironmentID  *xid.ID `bun:"environment_id,type:varchar(20)"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)"`

	UserID    xid.ID    `bun:"user_id,notnull,type:varchar(20)"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	IPAddress string    `bun:"ip_address"`
	UserAgent string    `bun:"user_agent"`

	// Refresh token support (Option 3)
	RefreshToken          *string    `bun:"refresh_token,unique"`     // Long-lived refresh token
	RefreshTokenExpiresAt *time.Time `bun:"refresh_token_expires_at"` // Refresh token expiry
	LastRefreshedAt       *time.Time `bun:"last_refreshed_at"`        // When was access token last refreshed

	// Relations
	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
