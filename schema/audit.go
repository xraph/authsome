package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// AuditEvent represents the audit_events table
type AuditEvent struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:audit_events,alias:ae"`

	ID             xid.ID  `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID          xid.ID  `json:"appID" bun:"app_id,notnull,type:varchar(20)"`
	OrganizationID *xid.ID `json:"organizationID,omitempty" bun:"organization_id,type:varchar(20)"` // User-created organization (optional)
	EnvironmentID  *xid.ID `json:"environmentID" bun:"environment_id,type:varchar(20)"`             // Environment scoping
	UserID         *xid.ID `json:"userID" bun:"user_id,type:varchar(20)"`
	Action         string  `json:"action" bun:"action,notnull"`
	Resource       string  `json:"resource" bun:"resource,notnull"`
	IPAddress      string  `json:"ipAddress" bun:"ip_address"`
	UserAgent      string  `json:"userAgent" bun:"user_agent"`
	Metadata       string  `json:"metadata" bun:"metadata"`

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
}
