package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// AuditEvent represents the audit_events table.
type AuditEvent struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:audit_events,alias:ae"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)"           json:"id"`
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)"  json:"appID"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // User-created organization (optional)
	EnvironmentID  *xid.ID `bun:"environment_id,type:varchar(20)"  json:"environmentID"`            // Environment scoping
	UserID         *xid.ID `bun:"user_id,type:varchar(20)"         json:"userID"`
	Action         string  `bun:"action,notnull"                   json:"action"`
	Resource       string  `bun:"resource,notnull"                 json:"resource"`
	Source         string  `bun:"source,notnull,default:'system'"  json:"source"` // Audit source: system, application, plugin
	IPAddress      string  `bun:"ip_address"                       json:"ipAddress"`
	UserAgent      string  `bun:"user_agent"                       json:"userAgent"`
	Metadata       string  `bun:"metadata"                         json:"metadata"`

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
}
