package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationTemplateVersion represents a version snapshot of a notification template
type NotificationTemplateVersion struct {
	bun.BaseModel `bun:"table:notification_template_versions,alias:ntv"`

	ID         xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	TemplateID xid.ID                 `bun:"template_id,notnull,type:varchar(20)" json:"templateId"`
	Version    int                    `bun:"version,notnull" json:"version"` // Version number
	Subject    string                 `bun:"subject" json:"subject,omitempty"`
	Body       string                 `bun:"body,notnull" json:"body"`
	Variables  []string               `bun:"variables,array" json:"variables"`
	Changes    string                 `bun:"changes" json:"changes,omitempty"`                       // Description of what changed
	ChangedBy  *xid.ID                `bun:"changed_by,type:varchar(20)" json:"changedBy,omitempty"` // User who made the change
	Metadata   map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt  time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	RestoredAt *time.Time             `bun:"restored_at" json:"restoredAt,omitempty"` // When this version was restored (if ever)

	// Relations
	Template *NotificationTemplate `bun:"rel:belongs-to,join:template_id=id" json:"template,omitempty"`
	User     *User                 `bun:"rel:belongs-to,join:changed_by=id" json:"user,omitempty"`
}
