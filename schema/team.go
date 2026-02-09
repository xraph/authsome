package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Team represents the team table (belongs to App).
type Team struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:teams,alias:t"`

	ID          xid.ID         `bun:"id,pk,type:varchar(20)"          json:"id"`
	AppID       xid.ID         `bun:"app_id,notnull,type:varchar(20)" json:"appID"` // App context
	Name        string         `bun:"name,notnull"                    json:"name"`
	Description string         `bun:"description"                     json:"description"`
	Metadata    map[string]any `bun:"metadata,type:jsonb"             json:"metadata"`

	// Provisioning tracking
	ProvisionedBy *string `bun:"provisioned_by,type:varchar(50)" json:"provisionedBy,omitempty"` // e.g., "scim"
	ExternalID    *string `bun:"external_id,type:varchar(255)"   json:"externalID,omitempty"`    // External system ID

	// Relations
	App     *App     `bun:"rel:belongs-to,join:app_id=id"     json:"app,omitempty"`
	Members []Member `bun:"m2m:team_members,join:Team=Member" json:"members,omitempty"`
}
