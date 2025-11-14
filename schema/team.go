package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Team represents the team table (belongs to App)
type Team struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:teams,alias:t"`

	ID          xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID       xid.ID                 `json:"appID" bun:"organization_id,notnull,type:varchar(20)"` // Column still named organization_id for migration compatibility
	Name        string                 `json:"name" bun:"name,notnull"`
	Description string                 `json:"description" bun:"description"`
	Metadata    map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	App     *App     `json:"app,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Members []Member `json:"members,omitempty" bun:"m2m:team_members,join:Team=Member"`
}
