package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// TeamMember represents the team_members table
type TeamMember struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:team_members,alias:tm"`

	ID       xid.ID `bun:"id,pk,type:varchar(20)"`
	TeamID   xid.ID `bun:"team_id,notnull,type:varchar(20)"`
	MemberID xid.ID `bun:"member_id,notnull,type:varchar(20)"`

	// Provisioning tracking
	ProvisionedBy *string `json:"provisionedBy,omitempty" bun:"provisioned_by,type:varchar(50)"` // e.g., "scim"

	// Relations
	Team   *Team   `bun:"rel:belongs-to,join:team_id=id"`
	Member *Member `bun:"rel:belongs-to,join:member_id=id"`
}
