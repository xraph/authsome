package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// FormSchema represents app-specific form configurations
type FormSchema struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:form_schemas,alias:fs"`

	ID          xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID       xid.ID                 `json:"appID" bun:"app_id,notnull,type:varchar(20)"` // App-scoped form schemas
	Type        string                 `json:"type" bun:"type,notnull"`                     // signup, signin, profile, etc.
	Name        string                 `json:"name" bun:"name,notnull"`
	Description string                 `json:"description" bun:"description"`
	Schema      map[string]interface{} `json:"schema" bun:"schema,type:jsonb,notnull"`
	IsActive    bool                   `json:"isActive" bun:"is_active,notnull,default:true"`
	Version     int                    `json:"version" bun:"version,notnull,default:1"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// FormField represents a single form field configuration
type FormField struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // text, email, password, select, checkbox, etc.
	Label       string                 `json:"label"`
	Placeholder string                 `json:"placeholder"`
	Required    bool                   `json:"required"`
	Validation  map[string]interface{} `json:"validation"`
	Options     []string               `json:"options,omitempty"` // For select fields
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FormSubmission represents a form submission
type FormSubmission struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:form_submissions,alias:fsub"`

	ID           xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	FormSchemaID xid.ID                 `json:"formSchemaId" bun:"form_schema_id,notnull,type:varchar(20)"`
	UserID       *xid.ID                `json:"userId" bun:"user_id,type:varchar(20)"` // Optional for anonymous submissions
	SessionID    *xid.ID                `json:"sessionId" bun:"session_id,type:varchar(20)"`
	Data         map[string]interface{} `json:"data" bun:"data,type:jsonb,notnull"`
	IPAddress    string                 `json:"ipAddress" bun:"ip_address"`
	UserAgent    string                 `json:"userAgent" bun:"user_agent"`
	Status       string                 `json:"status" bun:"status,notnull,default:'submitted'"` // submitted, processed, failed

	// Relations
	FormSchema *FormSchema `bun:"rel:belongs-to,join:form_schema_id=id"`
	User       *User       `bun:"rel:belongs-to,join:user_id=id"`
	Session    *Session    `bun:"rel:belongs-to,join:session_id=id"`
}
