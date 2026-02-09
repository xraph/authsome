package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Organization represents user-created organizations (Clerk-style workspaces) within an environment.
type Organization struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organizations,alias:uo"`

	ID            xid.ID         `bun:"id,pk,type:varchar(20)"                  json:"id"`
	AppID         xid.ID         `bun:"app_id,notnull,type:varchar(20)"         json:"appID"`
	EnvironmentID xid.ID         `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"` // Foreign key to Environment
	Name          string         `bun:"name,notnull"                            json:"name"`
	Slug          string         `bun:"slug,notnull"                            json:"slug"` // Unique within app+environment
	Logo          string         `bun:"logo"                                    json:"logo"`
	Metadata      map[string]any `bun:"metadata,type:jsonb"                     json:"metadata"`
	CreatedBy     xid.ID         `bun:"created_by,notnull,type:varchar(20)"     json:"createdBy"`

	// Relations
	App         *App         `bun:"rel:belongs-to,join:app_id=id"         json:"app"`
	Environment *Environment `bun:"rel:belongs-to,join:environment_id=id" json:"environment"`
}

// OrganizationMember represents membership in user-created organizations.
type OrganizationMember struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_members,alias:uom"`

	ID             xid.ID    `bun:"id,pk,type:varchar(20)"                               json:"id"`
	OrganizationID xid.ID    `bun:"organization_id,notnull,type:varchar(20)"             json:"organizationID"`
	UserID         xid.ID    `bun:"user_id,notnull,type:varchar(20)"                     json:"userID"`
	Role           string    `bun:"role,notnull"                                         json:"role"`   // owner, admin, member
	Status         string    `bun:"status,notnull,default:'active'"                      json:"status"` // active, suspended, pending
	JoinedAt       time.Time `bun:"joined_at,nullzero,notnull,default:current_timestamp" json:"joinedAt"`

	// Relations
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization"`
	User         *User         `bun:"rel:belongs-to,join:user_id=id"         json:"user"`
}

// OrganizationTeam represents teams within user-created organizations.
type OrganizationTeam struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_teams,alias:uot"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"                   json:"id"`
	OrganizationID xid.ID         `bun:"organization_id,notnull,type:varchar(20)" json:"organizationID"`
	Name           string         `bun:"name,notnull"                             json:"name"`
	Description    string         `bun:"description"                              json:"description"`
	Metadata       map[string]any `bun:"metadata,type:jsonb"                      json:"metadata"`

	// Provisioning tracking
	ProvisionedBy *string `bun:"provisioned_by,type:varchar(50)" json:"provisionedBy,omitempty"` // e.g., "scim"
	ExternalID    *string `bun:"external_id,type:varchar(255)"   json:"externalID,omitempty"`    // External system ID

	// Relations
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization"`
}

// OrganizationTeamMember represents team membership within user-created organizations.
type OrganizationTeamMember struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_team_members,alias:uotm"`

	ID       xid.ID    `bun:"id,pk,type:varchar(20)"                               json:"id"`
	TeamID   xid.ID    `bun:"team_id,notnull,type:varchar(20)"                     json:"teamID"`
	MemberID xid.ID    `bun:"member_id,notnull,type:varchar(20)"                   json:"memberID"` // References OrganizationMember
	JoinedAt time.Time `bun:"joined_at,nullzero,notnull,default:current_timestamp" json:"joinedAt"`

	// Provisioning tracking
	ProvisionedBy *string `bun:"provisioned_by,type:varchar(50)" json:"provisionedBy,omitempty"` // e.g., "scim"

	// Relations
	Team   *OrganizationTeam   `bun:"rel:belongs-to,join:team_id=id"   json:"team"`
	Member *OrganizationMember `bun:"rel:belongs-to,join:member_id=id" json:"member"`
}

// OrganizationInvitation represents invitations to user-created organizations.
type OrganizationInvitation struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organization_invitations,alias:uoi"`

	ID             xid.ID     `bun:"id,pk,type:varchar(20)"                   json:"id"`
	OrganizationID xid.ID     `bun:"organization_id,notnull,type:varchar(20)" json:"organizationID"`
	Email          string     `bun:"email,notnull"                            json:"email"`
	Role           string     `bun:"role,notnull"                             json:"role"`
	InviterID      xid.ID     `bun:"inviter_id,notnull,type:varchar(20)"      json:"inviterID"`
	Token          string     `bun:"token,notnull,unique"                     json:"token"`
	ExpiresAt      time.Time  `bun:"expires_at,notnull"                       json:"expiresAt"`
	AcceptedAt     *time.Time `bun:"accepted_at"                              json:"acceptedAt"`
	Status         string     `bun:"status,notnull"                           json:"status"` // pending, accepted, expired, cancelled

	// Relations
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization"`
}
