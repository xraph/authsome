package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// FormSchema represents app-specific form configurations.
type FormSchema struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:form_schemas,alias:fs"`

	ID          xid.ID         `bun:"id,pk,type:varchar(20)"          json:"id"`
	AppID       xid.ID         `bun:"app_id,notnull,type:varchar(20)" json:"appID"` // App-scoped form schemas
	Type        string         `bun:"type,notnull"                    json:"type"`  // signup, signin, profile, etc.
	Name        string         `bun:"name,notnull"                    json:"name"`
	Description string         `bun:"description"                     json:"description"`
	Schema      map[string]any `bun:"schema,type:jsonb,notnull"       json:"schema"`
	IsActive    bool           `bun:"is_active,notnull,default:true"  json:"isActive"`
	Version     int            `bun:"version,notnull,default:1"       json:"version"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// FormField represents a single form field configuration.
type FormField struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"` // text, email, password, select, checkbox, etc.
	Label       string         `json:"label"`
	Placeholder string         `json:"placeholder"`
	Required    bool           `json:"required"`
	Validation  map[string]any `json:"validation"`
	Options     []string       `json:"options,omitempty"` // For select fields
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// FormSubmission represents a form submission.
type FormSubmission struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:form_submissions,alias:fsub"`

	ID           xid.ID         `bun:"id,pk,type:varchar(20)"                  json:"id"`
	FormSchemaID xid.ID         `bun:"form_schema_id,notnull,type:varchar(20)" json:"formSchemaId"`
	UserID       *xid.ID        `bun:"user_id,type:varchar(20)"                json:"userId"` // Optional for anonymous submissions
	SessionID    *xid.ID        `bun:"session_id,type:varchar(20)"             json:"sessionId"`
	Data         map[string]any `bun:"data,type:jsonb,notnull"                 json:"data"`
	IPAddress    string         `bun:"ip_address"                              json:"ipAddress"`
	UserAgent    string         `bun:"user_agent"                              json:"userAgent"`
	Status       string         `bun:"status,notnull,default:'submitted'"      json:"status"` // submitted, processed, failed

	// Relations
	FormSchema *FormSchema `bun:"rel:belongs-to,join:form_schema_id=id"`
	User       *User       `bun:"rel:belongs-to,join:user_id=id"`
	Session    *Session    `bun:"rel:belongs-to,join:session_id=id"`
}
