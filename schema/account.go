package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Account represents OAuth accounts
type Account struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:accounts,alias:a"`

	ID           xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID       xid.ID     `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	Provider     string     `json:"provider" bun:"provider,notnull"` // google, github, etc.
	ProviderID   string     `json:"providerID" bun:"provider_id,notnull"`
	AccessToken  string     `json:"accessToken" bun:"access_token"`
	RefreshToken string     `json:"refreshToken" bun:"refresh_token"`
	ExpiresAt    *time.Time `json:"expiresAt" bun:"expires_at"`

	// Relations
	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
