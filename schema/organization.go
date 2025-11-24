package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Organization represents user-created organizations (Clerk-style workspaces) within an environment
type Organization struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organizations,alias:uo"`

	ID            xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID         xid.ID                 `json:"appID" bun:"app_id,notnull,type:varchar(20)"`
	EnvironmentID xid.ID                 `json:"environmentID" bun:"environment_id,notnull,type:varchar(20)"` // Foreign key to Environment
	Name          string                 `json:"name" bun:"name,notnull"`
	Slug          string                 `json:"slug" bun:"slug,notnull"` // Unique within app+environment
	Logo          string                 `json:"logo" bun:"logo"`
	Metadata      map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedBy     xid.ID                 `json:"createdBy" bun:"created_by,notnull,type:varchar(20)"`

	// Relations
	App         *App         `json:"app" bun:"rel:belongs-to,join:app_id=id"`
	Environment *Environment `json:"environment" bun:"rel:belongs-to,join:environment_id=id"`
}

// OrganizationMember represents membership in user-created organizations
type OrganizationMember struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_members,alias:uom"`

	ID             xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID    `json:"organizationID" bun:"organization_id,notnull,type:varchar(20)"`
	UserID         xid.ID    `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	Role           string    `json:"role" bun:"role,notnull"`                      // owner, admin, member
	Status         string    `json:"status" bun:"status,notnull,default:'active'"` // active, suspended, pending
	JoinedAt       time.Time `json:"joinedAt" bun:"joined_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Organization *Organization `json:"organization" bun:"rel:belongs-to,join:organization_id=id"`
	User         *User         `json:"user" bun:"rel:belongs-to,join:user_id=id"`
}

// OrganizationTeam represents teams within user-created organizations
type OrganizationTeam struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_teams,alias:uot"`

	ID             xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID                 `json:"organizationID" bun:"organization_id,notnull,type:varchar(20)"`
	Name           string                 `json:"name" bun:"name,notnull"`
	Description    string                 `json:"description" bun:"description"`
	Metadata       map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Provisioning tracking
	ProvisionedBy *string `json:"provisionedBy,omitempty" bun:"provisioned_by,type:varchar(50)"` // e.g., "scim"
	ExternalID    *string `json:"externalID,omitempty" bun:"external_id,type:varchar(255)"`      // External system ID

	// Relations
	Organization *Organization `json:"organization" bun:"rel:belongs-to,join:organization_id=id"`
}

// OrganizationTeamMember represents team membership within user-created organizations
type OrganizationTeamMember struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_team_members,alias:uotm"`

	ID       xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	TeamID   xid.ID    `json:"teamID" bun:"team_id,notnull,type:varchar(20)"`
	MemberID xid.ID    `json:"memberID" bun:"member_id,notnull,type:varchar(20)"` // References OrganizationMember
	JoinedAt time.Time `json:"joinedAt" bun:"joined_at,nullzero,notnull,default:current_timestamp"`

	// Provisioning tracking
	ProvisionedBy *string `json:"provisionedBy,omitempty" bun:"provisioned_by,type:varchar(50)"` // e.g., "scim"

	// Relations
	Team   *OrganizationTeam   `json:"team" bun:"rel:belongs-to,join:team_id=id"`
	Member *OrganizationMember `json:"member" bun:"rel:belongs-to,join:member_id=id"`
}

// OrganizationInvitation represents invitations to user-created organizations
type OrganizationInvitation struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_invitations,alias:uoi"`

	ID             xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID     `json:"organizationID" bun:"organization_id,notnull,type:varchar(20)"`
	Email          string     `json:"email" bun:"email,notnull"`
	Role           string     `json:"role" bun:"role,notnull"`
	InviterID      xid.ID     `json:"inviterID" bun:"inviter_id,notnull,type:varchar(20)"`
	Token          string     `json:"token" bun:"token,notnull,unique"`
	ExpiresAt      time.Time  `json:"expiresAt" bun:"expires_at,notnull"`
	AcceptedAt     *time.Time `json:"acceptedAt" bun:"accepted_at"`
	Status         string     `json:"status" bun:"status,notnull"` // pending, accepted, expired, cancelled

	// Relations
	Organization *Organization `json:"organization" bun:"rel:belongs-to,join:organization_id=id"`
}
