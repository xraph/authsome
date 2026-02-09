package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Account represents OAuth accounts.
type Account struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:accounts,alias:a"`

	ID           xid.ID     `bun:"id,pk,type:varchar(20)"           json:"id"`
	AppID        xid.ID     `bun:"app_id,notnull,type:varchar(20)"  json:"appID"`
	UserID       xid.ID     `bun:"user_id,notnull,type:varchar(20)" json:"userID"`
	Provider     string     `bun:"provider,notnull"                 json:"provider"` // google, github, etc.
	ProviderID   string     `bun:"provider_id,notnull"              json:"providerID"`
	AccessToken  string     `bun:"access_token"                     json:"accessToken"`
	RefreshToken string     `bun:"refresh_token"                    json:"refreshToken"`
	ExpiresAt    *time.Time `bun:"expires_at"                       json:"expiresAt"`

	// Relations
	App  *App  `bun:"rel:belongs-to,join:app_id=id"`
	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
