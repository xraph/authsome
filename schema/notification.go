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
	AppID          xid.ID                 `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	OrganizationID *xid.ID                `bun:"organization_id,type:varchar(20)" json:"organizationId,omitempty"` // Nullable for app-level templates
	TemplateKey    string                 `bun:"template_key,notnull" json:"templateKey"`                          // e.g., "auth.welcome", "auth.mfa_code"
	Name           string                 `bun:"name,notnull" json:"name"`
	Type           string                 `bun:"type,notnull" json:"type"`
	Language       string                 `bun:"language,notnull,default:'en'" json:"language"`
	Subject        string                 `bun:"subject" json:"subject,omitempty"`
	Body           string                 `bun:"body,notnull" json:"body"`
	Variables      []string               `bun:"variables,array" json:"variables"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	Active         bool                   `bun:"active,notnull,default:true" json:"active"`
	IsDefault      bool                   `bun:"is_default,notnull,default:false" json:"isDefault"`   // Is this a default template
	IsModified     bool                   `bun:"is_modified,notnull,default:false" json:"isModified"` // Has it been modified from default
	DefaultHash    string                 `bun:"default_hash" json:"defaultHash"`                     // Hash of default content for comparison

	// Versioning fields
	Version  int     `bun:"version,notnull,default:1" json:"version"`             // Current version number
	ParentID *xid.ID `bun:"parent_id,type:varchar(20)" json:"parentId,omitempty"` // ID of template this was cloned from

	// A/B Testing fields
	ABTestGroup   string `bun:"ab_test_group" json:"abTestGroup,omitempty"`                 // Group identifier for variants
	ABTestEnabled bool   `bun:"ab_test_enabled,notnull,default:false" json:"abTestEnabled"` // Is this variant active in A/B test
	ABTestWeight  int    `bun:"ab_test_weight,notnull,default:100" json:"abTestWeight"`     // Weight for variant selection (0-100)

	// Analytics fields
	SendCount       int64 `bun:"send_count,notnull,default:0" json:"sendCount"`             // Total sends
	OpenCount       int64 `bun:"open_count,notnull,default:0" json:"openCount"`             // Total opens
	ClickCount      int64 `bun:"click_count,notnull,default:0" json:"clickCount"`           // Total clicks
	ConversionCount int64 `bun:"conversion_count,notnull,default:0" json:"conversionCount"` // Total conversions

	CreatedAt time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"-"`
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
