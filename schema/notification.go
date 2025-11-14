package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationTemplate represents a notification template in the database
type NotificationTemplate struct {
	bun.BaseModel `bun:"table:notification_templates,alias:nt"`

	ID          xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID       xid.ID                 `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	TemplateKey string                 `bun:"template_key,notnull" json:"templateKey"` // e.g., "auth.welcome", "auth.mfa_code"
	Name        string                 `bun:"name,notnull" json:"name"`
	Type        string                 `bun:"type,notnull" json:"type"`
	Language    string                 `bun:"language,notnull,default:'en'" json:"language"`
	Subject     string                 `bun:"subject" json:"subject,omitempty"`
	Body        string                 `bun:"body,notnull" json:"body"`
	Variables   []string               `bun:"variables,array" json:"variables"`
	Metadata    map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	Active      bool                   `bun:"active,notnull,default:true" json:"active"`
	IsDefault   bool                   `bun:"is_default,notnull,default:false" json:"isDefault"`     // Is this a default template
	IsModified  bool                   `bun:"is_modified,notnull,default:false" json:"isModified"`   // Has it been modified from default
	DefaultHash string                 `bun:"default_hash" json:"defaultHash"`                       // Hash of default content for comparison
	CreatedAt   time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt   *time.Time             `bun:"deleted_at,soft_delete,nullzero" json:"-"`
}

// Notification represents a notification instance in the database
type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:n"`

	ID          xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID       xid.ID                 `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	TemplateID  *xid.ID                `bun:"template_id,type:varchar(20)" json:"templateId,omitempty"`
	Type        string                 `bun:"type,notnull" json:"type"`
	Recipient   string                 `bun:"recipient,notnull" json:"recipient"`
	Subject     string                 `bun:"subject" json:"subject,omitempty"`
	Body        string                 `bun:"body,notnull" json:"body"`
	Status      string                 `bun:"status,notnull" json:"status"`
	Error       string                 `bun:"error" json:"error,omitempty"`
	ProviderID  string                 `bun:"provider_id" json:"providerId,omitempty"`
	Metadata    map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	SentAt      *time.Time             `bun:"sent_at" json:"sentAt,omitempty"`
	DeliveredAt *time.Time             `bun:"delivered_at" json:"deliveredAt,omitempty"`
	CreatedAt   time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relations
	Template *NotificationTemplate `bun:"rel:belongs-to,join:template_id=id" json:"template,omitempty"`
}
