package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// MemberRole represents the role of a member in an app
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// IsValid checks if the role is valid
func (r MemberRole) IsValid() bool {
	switch r {
	case MemberRoleOwner, MemberRoleAdmin, MemberRoleMember:
		return true
	}
	return false
}

// String returns the string representation
func (r MemberRole) String() string {
	return string(r)
}

// MemberStatus represents the status of a member
type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "active"
	MemberStatusSuspended MemberStatus = "suspended"
	MemberStatusPending   MemberStatus = "pending"
)

// IsValid checks if the status is valid
func (s MemberStatus) IsValid() bool {
	switch s {
	case MemberStatusActive, MemberStatusSuspended, MemberStatusPending:
		return true
	}
	return false
}

// String returns the string representation
func (s MemberStatus) String() string {
	return string(s)
}

// Member represents the app member table
type Member struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:members,alias:m"`

	ID       xid.ID       `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID    xid.ID       `json:"appID" bun:"organization_id,notnull,type:varchar(20)"` // Column still named organization_id for migration compatibility
	UserID   xid.ID       `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	Role     MemberRole   `json:"role" bun:"role,notnull"`
	Status   MemberStatus `json:"status" bun:"status,notnull,default:'active'"`
	JoinedAt time.Time    `json:"joinedAt" bun:"joined_at,nullzero,notnull,default:current_timestamp"` // when the member joined

	// Relations
	App   *App   `json:"app,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	User  *User  `json:"user,omitempty" bun:"rel:belongs-to,join:user_id=id"`
	Teams []Team `json:"teams,omitempty" bun:"m2m:team_members,join:Member=Team"`
}
