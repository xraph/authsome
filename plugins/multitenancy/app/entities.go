package app

import (
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Errors
var (
	ErrAppNotFound         = errors.New("app not found")
	ErrMemberNotFound      = errors.New("member not found")
	ErrTeamNotFound        = errors.New("team not found")
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvalidRole         = errors.New("invalid role")
	ErrInvalidStatus       = errors.New("invalid status")
	ErrSlugAlreadyExists   = errors.New("app slug already exists")
	ErrMemberAlreadyExists = errors.New("member already exists")
	ErrInvitationExpired   = errors.New("invitation has expired")
)

// App represents a tenant app (platform-level)
type App struct {
	bun.BaseModel `bun:"table:organizations,alias:a"`

	ID        xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	Name      string                 `bun:"name,notnull" json:"name"`
	Slug      string                 `bun:"slug,unique,notnull" json:"slug"`
	Logo      *string                `bun:"logo" json:"logo,omitempty"`
	Metadata  map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relationships
	Members []Member `bun:"rel:has-many,join:id=organization_id" json:"members,omitempty"`
	Teams   []Team   `bun:"rel:has-many,join:id=organization_id" json:"teams,omitempty"`
}

// Member represents a user's membership in an app
type Member struct {
	bun.BaseModel `bun:"table:members,alias:m"`

	ID        xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID     xid.ID    `bun:"organization_id,notnull,type:varchar(20)" json:"appId"` // Column still named organization_id
	UserID    xid.ID    `bun:"user_id,notnull,type:varchar(20)" json:"userId"`
	Role      string    `bun:"role,notnull" json:"role"`                      // owner, admin, member
	Status    string    `bun:"status,notnull,default:'active'" json:"status"` // active, suspended, pending
	JoinedAt  time.Time `bun:"joined_at,nullzero,notnull,default:current_timestamp" json:"joinedAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relationships
	App   *App   `bun:"rel:belongs-to,join:organization_id=id" json:"app,omitempty"`
	Teams []Team `bun:"m2m:team_members,join:Member=Team" json:"teams,omitempty"`
}

// Team represents a team within an app
type Team struct {
	bun.BaseModel `bun:"table:teams,alias:t"`

	ID          xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID       xid.ID                 `bun:"organization_id,notnull,type:varchar(20)" json:"appId"` // Column still named organization_id
	Name        string                 `bun:"name,notnull" json:"name"`
	Description *string                `bun:"description" json:"description,omitempty"`
	Metadata    map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt   time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relationships
	App     *App     `bun:"rel:belongs-to,join:organization_id=id" json:"app,omitempty"`
	Members []Member `bun:"m2m:team_members,join:Team=Member" json:"members,omitempty"`
}

// TeamMember represents the many-to-many relationship between teams and members
type TeamMember struct {
	bun.BaseModel `bun:"table:team_members,alias:tm"`

	TeamID   xid.ID    `bun:"team_id,pk,type:varchar(20)" json:"teamId"`
	MemberID xid.ID    `bun:"member_id,pk,type:varchar(20)" json:"memberId"`
	Role     string    `bun:"role,notnull,default:'member'" json:"role"` // lead, member
	JoinedAt time.Time `bun:"joined_at,nullzero,notnull,default:current_timestamp" json:"joinedAt"`

	// Relationships
	Team   *Team   `bun:"rel:belongs-to,join:team_id=id" json:"team,omitempty"`
	Member *Member `bun:"rel:belongs-to,join:member_id=id" json:"member,omitempty"`
}

// Invitation represents an invitation to join an app
type Invitation struct {
	bun.BaseModel `bun:"table:invitations,alias:i"`

	ID        xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID     xid.ID                 `bun:"organization_id,notnull,type:varchar(20)" json:"appId"` // Column still named organization_id
	Email     string                 `bun:"email,notnull" json:"email"`
	Role      string                 `bun:"role,notnull" json:"role"`
	Token     string                 `bun:"token,unique,notnull" json:"token"`
	Status    string                 `bun:"status,notnull,default:'pending'" json:"status"` // pending, accepted, declined, expired
	InvitedBy xid.ID                 `bun:"invited_by,notnull,type:varchar(20)" json:"invitedBy"`
	Metadata  map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	ExpiresAt time.Time              `bun:"expires_at,notnull" json:"expiresAt"`
	CreatedAt time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time              `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relationships
	App *App `bun:"rel:belongs-to,join:organization_id=id" json:"app,omitempty"`
}

// Role types
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Status types
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusPending   = "pending"
)

// Invitation status types
const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusDeclined = "declined"
	InvitationStatusExpired  = "expired"
)
