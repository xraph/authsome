package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationTemplate represents a notification template in the database
type NotificationTemplate struct {
	bun.BaseModel `bun:"table:notification_templates,alias:nt"`

	ID             xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	OrganizationID string                 `bun:"organization_id,notnull" json:"organization_id"`
	TemplateKey    string                 `bun:"template_key,notnull" json:"template_key"` // e.g., "auth.welcome", "auth.mfa_code"
	Name           string                 `bun:"name,notnull" json:"name"`
	Type           string                 `bun:"type,notnull" json:"type"`
	Language       string                 `bun:"language,notnull,default:'en'" json:"language"`
	Subject        string                 `bun:"subject" json:"subject,omitempty"`
	Body           string                 `bun:"body,notnull" json:"body"`
	Variables      []string               `bun:"variables,array" json:"variables"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	Active         bool                   `bun:"active,notnull,default:true" json:"active"`
	CreatedAt      time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt      *time.Time             `bun:"deleted_at,soft_delete,nullzero" json:"-"`
}

// Notification represents a notification instance in the database
type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:n"`

	ID             xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	OrganizationID string                 `bun:"organization_id,notnull" json:"organization_id"`
	TemplateID     *xid.ID                `bun:"template_id,type:varchar(20)" json:"template_id,omitempty"`
	Type           string                 `bun:"type,notnull" json:"type"`
	Recipient      string                 `bun:"recipient,notnull" json:"recipient"`
	Subject        string                 `bun:"subject" json:"subject,omitempty"`
	Body           string                 `bun:"body,notnull" json:"body"`
	Status         string                 `bun:"status,notnull" json:"status"`
	Error          string                 `bun:"error" json:"error,omitempty"`
	ProviderID     string                 `bun:"provider_id" json:"provider_id,omitempty"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	SentAt         *time.Time             `bun:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt    *time.Time             `bun:"delivered_at" json:"delivered_at,omitempty"`
	CreatedAt      time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	Template *NotificationTemplate `bun:"rel:belongs-to,join:template_id=id" json:"template,omitempty"`
}