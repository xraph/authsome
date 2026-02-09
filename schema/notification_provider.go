package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationProvider represents a notification provider configuration in the database
// Providers can be configured at app-level or organization-level.
type NotificationProvider struct {
	bun.BaseModel `bun:"table:notification_providers,alias:np"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	AppID          xid.ID         `bun:"app_id,notnull,type:varchar(20)"                       json:"appId"`
	OrganizationID *xid.ID        `bun:"organization_id,type:varchar(20)"                      json:"organizationId,omitempty"` // Nullable for app-level providers
	ProviderType   string         `bun:"provider_type,notnull"                                 json:"providerType"`             // email, sms, push
	ProviderName   string         `bun:"provider_name,notnull"                                 json:"providerName"`             // smtp, sendgrid, twilio, etc.
	Config         map[string]any `bun:"config,type:jsonb,notnull"                             json:"config"`                   // Encrypted provider configuration
	IsActive       bool           `bun:"is_active,notnull,default:true"                        json:"isActive"`
	IsDefault      bool           `bun:"is_default,notnull,default:false"                      json:"isDefault"` // Default provider for this type
	Metadata       map[string]any `bun:"metadata,type:jsonb"                                   json:"metadata,omitempty"`
	CreatedAt      time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time      `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt      *time.Time     `bun:"deleted_at,soft_delete,nullzero"                       json:"-"`

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"          json:"app,omitempty"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}
