package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Team represents the team table (belongs to App)
type Team struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:teams,alias:t"`

	ID          xid.ID `bun:"id,pk,type:varchar(20)"`
	AppID       xid.ID `bun:"organization_id,notnull,type:varchar(20)"` // Column still named organization_id for migration compatibility
	Name        string `bun:"name,notnull"`
	Description string `bun:"description"`

	// Relations
	App *App `bun:"rel:belongs-to,join:organization_id=id"`
}
